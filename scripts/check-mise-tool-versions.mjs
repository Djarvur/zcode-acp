#!/usr/bin/env node
// scripts/check-mise-tool-versions.mjs — compare the four mise-managed tool
// pins plus the PATH-level Bash requirement with mise.toml. This gate never
// installs or trusts tools.

import { spawnSync } from "node:child_process";
import { readFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const SCRIPT_PATH = fileURLToPath(import.meta.url);
const REPO_ROOT = path.resolve(path.dirname(SCRIPT_PATH), "..");
const VERSION_PATTERN = /^\d+(?:\.\d+){0,2}$/;
const PROBE_TIMEOUT_MS = 30_000;
const MISE_TOOLS = ["go", "golangci-lint", "goreleaser", "node"];
const REQUIRED_TOOLS = [...MISE_TOOLS, "bash"];
const BASH_REQUIREMENT_ENV = "ZCODE_ACP_REQUIRED_BASH";
const VERSION_ARGS = {
  go: ["version"],
  "golangci-lint": ["--version"],
  goreleaser: ["--version"],
  node: ["--version"],
};

export function resolvedVersionFromOutput(output) {
  if (typeof output !== "string") return null;
  const match = /(?:^|[^0-9.])(?:go|v)?(\d+\.\d+(?:\.\d+)?)(?!\.\d)/.exec(output);
  return match?.[1] ?? null;
}

export function requiredVersionMatches(required, resolved) {
  if (
    typeof required !== "string" ||
    typeof resolved !== "string" ||
    !VERSION_PATTERN.test(required) ||
    !VERSION_PATTERN.test(resolved)
  ) {
    return false;
  }

  const requiredParts = required.split(".");
  const resolvedParts = resolved.split(".");
  if (requiredParts.length === 3) return required === resolved;
  return requiredParts.every((part, index) => part === resolvedParts[index]);
}

function escapeRegExp(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

function readTomlTable(document, tableName) {
  const lines = document.split(/\r?\n/);
  const start = lines.findIndex((line) => line.trim() === `[${tableName}]`);
  if (start === -1) throw new Error(`mise.toml is missing the [${tableName}] table`);

  let end = lines.length;
  for (let index = start + 1; index < lines.length; index += 1) {
    if (/^\[[^\]]+\]\s*$/.test(lines[index])) {
      end = index;
      break;
    }
  }
  return lines.slice(start + 1, end).join("\n");
}

function readRequiredPin(document, tableName, key) {
  const table = readTomlTable(document, tableName);
  const entryPattern = new RegExp(
    `^[\\t ]*${escapeRegExp(key)}[\\t ]*=[\\t ]*(.*?)[\\t ]*$`,
    "gm",
  );
  const entries = [...table.matchAll(entryPattern)];
  if (entries.length !== 1) {
    throw new Error(`mise.toml [${tableName}] must define ${key} exactly once`);
  }

  const value = /^"([^"]+)"$/.exec(entries[0][1])?.[1];
  if (!value || !VERSION_PATTERN.test(value)) {
    throw new Error(`mise.toml [${tableName}].${key} must be a quoted numeric version`);
  }
  return value;
}

function readRequiredPins(document) {
  const pins = new Map();
  for (const tool of MISE_TOOLS) {
    pins.set(tool, readRequiredPin(document, "tools", tool));
  }

  const bashPin = readRequiredPin(document, "env", BASH_REQUIREMENT_ENV);
  if (bashPin.split(".").length === 3) {
    throw new Error(
      `mise.toml [env].${BASH_REQUIREMENT_ENV} must be a major or major.minor line`,
    );
  }
  pins.set("bash", bashPin);
  return pins;
}

function writeOutput(stream, text) {
  stream.write(text.endsWith("\n") ? text : `${text}\n`);
}

function probeVersion(label, command, args, required, source) {
  const result = spawnSync(command, args, {
    cwd: REPO_ROOT,
    encoding: "utf8",
    timeout: PROBE_TIMEOUT_MS,
    maxBuffer: 1024 * 1024,
  });
  const rawOutput = `${result.stdout ?? ""}${result.stderr ?? ""}`;
  const resolved = resolvedVersionFromOutput(rawOutput);

  if (rawOutput) writeOutput(process.stdout, rawOutput);
  writeOutput(process.stdout, `${label}: required=${required} resolved=${resolved ?? "<missing>"}`);

  if (result.error) {
    writeOutput(process.stderr, `${label}: ${source} probe failed: ${result.error.message}`);
    return false;
  }
  if (result.status !== 0) {
    writeOutput(process.stderr, `${label}: ${source} probe exited with status ${result.status}`);
    return false;
  }
  if (!resolved) {
    writeOutput(process.stderr, `${label}: ${source} probe did not report a dotted version`);
    return false;
  }
  if (!requiredVersionMatches(required, resolved)) {
    writeOutput(process.stderr, `${label}: resolved ${resolved} does not match required ${required}`);
    return false;
  }
  return true;
}

function probeMiseTool(tool, required) {
  return probeVersion(tool, "mise", ["exec", "--", tool, ...VERSION_ARGS[tool]], required, "mise");
}

function probeBash(required) {
  return probeVersion("bash", "bash", ["--version"], required, "PATH");
}

function main() {
  const requested = process.argv.slice(2);
  const unknown = requested.filter((tool) => !REQUIRED_TOOLS.includes(tool));
  if (unknown.length > 0) {
    writeOutput(process.stderr, `unknown tool(s): ${unknown.join(", ")}`);
    process.exitCode = 1;
    return;
  }

  let pins;
  try {
    pins = readRequiredPins(readFileSync(path.join(REPO_ROOT, "mise.toml"), "utf8"));
  } catch (error) {
    writeOutput(process.stderr, `tool-version gate: ${error.message}`);
    process.exitCode = 1;
    return;
  }

  const selected = requested.length > 0 ? requested : REQUIRED_TOOLS;
  let passed = true;
  for (const tool of selected) {
    const required = pins.get(tool);
    const ok = tool === "bash" ? probeBash(required) : probeMiseTool(tool, required);
    if (!ok) passed = false;
  }
  if (!passed) process.exitCode = 1;
}

if (process.argv[1] && path.resolve(process.argv[1]) === SCRIPT_PATH) main();
