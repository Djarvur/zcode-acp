#!/usr/bin/env node
// scripts/check-mise-tool-versions.mjs — compare mise-resolved tool versions
// with the single-source pins in mise.toml. This gate never installs, trusts,
// or falls back to a bare PATH command.

import { spawnSync } from "node:child_process";
import { readFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const SCRIPT_PATH = fileURLToPath(import.meta.url);
const REPO_ROOT = path.resolve(path.dirname(SCRIPT_PATH), "..");
const VERSION_PATTERN = /^\d+(?:\.\d+){0,2}$/;
const PROBE_TIMEOUT_MS = 30_000;
const REQUIRED_TOOLS = ["go", "golangci-lint", "goreleaser", "node", "bash"];
const VERSION_ARGS = {
  go: ["version"],
  "golangci-lint": ["--version"],
  goreleaser: ["--version"],
  node: ["--version"],
  bash: ["--version"],
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

function readRequiredToolPins(document) {
  const lines = document.split(/\r?\n/);
  const start = lines.findIndex((line) => /^\[tools\]\s*$/.test(line));
  if (start === -1) throw new Error("mise.toml is missing the [tools] table");

  let end = lines.length;
  for (let index = start + 1; index < lines.length; index += 1) {
    if (/^\[[^\]]+\]\s*$/.test(lines[index])) {
      end = index;
      break;
    }
  }
  const toolsTable = lines.slice(start + 1, end).join("\n");
  const pins = new Map();

  for (const tool of REQUIRED_TOOLS) {
    const entryPattern = new RegExp(
      `^[\\t ]*${escapeRegExp(tool)}[\\t ]*=[\\t ]*(.*?)[\\t ]*$`,
      "gm",
    );
    const entries = [...toolsTable.matchAll(entryPattern)];
    if (entries.length !== 1) {
      throw new Error(`mise.toml [tools] must define ${tool} exactly once`);
    }

    const value = /^"([^"]+)"$/.exec(entries[0][1])?.[1];
    if (!value || !VERSION_PATTERN.test(value)) {
      throw new Error(`mise.toml [tools].${tool} must be a quoted numeric version`);
    }
    pins.set(tool, value);
  }

  return pins;
}

function writeOutput(stream, text) {
  stream.write(text.endsWith("\n") ? text : `${text}\n`);
}

function probeTool(tool, required) {
  const result = spawnSync("mise", ["exec", "--", tool, ...VERSION_ARGS[tool]], {
    cwd: REPO_ROOT,
    encoding: "utf8",
    timeout: PROBE_TIMEOUT_MS,
    maxBuffer: 1024 * 1024,
  });
  const rawOutput = `${result.stdout ?? ""}${result.stderr ?? ""}`;
  const resolved = resolvedVersionFromOutput(rawOutput);

  if (rawOutput) writeOutput(process.stdout, rawOutput);
  writeOutput(process.stdout, `${tool}: required=${required} resolved=${resolved ?? "<missing>"}`);

  if (result.error) {
    writeOutput(process.stderr, `${tool}: mise probe failed: ${result.error.message}`);
    return false;
  }
  if (result.status !== 0) {
    writeOutput(process.stderr, `${tool}: mise probe exited with status ${result.status}`);
    return false;
  }
  if (!resolved) {
    writeOutput(process.stderr, `${tool}: mise probe did not report a dotted version`);
    return false;
  }
  if (!requiredVersionMatches(required, resolved)) {
    writeOutput(process.stderr, `${tool}: resolved ${resolved} does not match required ${required}`);
    return false;
  }
  return true;
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
    pins = readRequiredToolPins(readFileSync(path.join(REPO_ROOT, "mise.toml"), "utf8"));
  } catch (error) {
    writeOutput(process.stderr, `tool-version gate: ${error.message}`);
    process.exitCode = 1;
    return;
  }

  const selected = requested.length > 0 ? requested : REQUIRED_TOOLS;
  let passed = true;
  for (const tool of selected) {
    if (!probeTool(tool, pins.get(tool))) passed = false;
  }
  if (!passed) process.exitCode = 1;
}

if (process.argv[1] && path.resolve(process.argv[1]) === SCRIPT_PATH) main();
