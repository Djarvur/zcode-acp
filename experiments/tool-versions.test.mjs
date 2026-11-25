import { test } from "node:test";
import assert from "node:assert/strict";
import {
  requiredVersionMatches,
  resolvedVersionFromOutput,
} from "../scripts/check-mise-tool-versions.mjs";

test("resolvedVersionFromOutput extracts each tool's dotted version", () => {
  const cases = [
    ["go version go1.27.4 darwin/arm64", "1.27.4"],
    ["v26.13.1", "26.13.1"],
    ["GNU bash, version 5.3.20(1)-release", "5.3.20"],
    ["golangci-lint has version 2.15.0 built with go1.27.4", "2.15.0"],
    ["goreleaser version 2.20.2", "2.20.2"],
  ];

  for (const [output, expected] of cases) {
    assert.equal(resolvedVersionFromOutput(output), expected, output);
  }
});

test("resolvedVersionFromOutput rejects missing and malformed output", () => {
  assert.equal(resolvedVersionFromOutput(""), null);
  assert.equal(resolvedVersionFromOutput("no version here"), null);
  assert.equal(resolvedVersionFromOutput("version 1.2.3.4"), null);
  assert.equal(resolvedVersionFromOutput(null), null);
});

test("requiredVersionMatches accepts exact three-component pins", () => {
  for (const version of ["1.27.4", "2.15.0", "2.20.2", "5.3.20"]) {
    assert.equal(requiredVersionMatches(version, version), true, version);
  }
  assert.equal(requiredVersionMatches("1.27.4", "1.27"), false);
});

test("requiredVersionMatches accepts only the pinned one- or two-component line", () => {
  assert.equal(requiredVersionMatches("1.27", "1.27.4"), true);
  assert.equal(requiredVersionMatches("5.3", "5.3.20"), true);
  assert.equal(requiredVersionMatches("26", "26.13.1"), true);
  assert.equal(requiredVersionMatches("1.27", "1.28.4"), false);
  assert.equal(requiredVersionMatches("26", "27.13.1"), false);
});

test("requiredVersionMatches rejects wrong patches, malformed pins, and missing versions", () => {
  assert.equal(requiredVersionMatches("2.15.0", "2.15.1"), false);
  assert.equal(requiredVersionMatches("1.27.4", "2.27.4"), false);
  assert.equal(requiredVersionMatches("1.27.4", ""), false);
  assert.equal(requiredVersionMatches("", "1.27.4"), false);
  assert.equal(requiredVersionMatches("latest", "1.27.4"), false);
  assert.equal(requiredVersionMatches("1.27.4", null), false);
});
