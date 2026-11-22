// experiments/vm-e2e-script.test.mjs — the no-keyring stage contract of
// scripts/vm-e2e.sh (QA-02 / D-05, plan 07-07 task 2). The script's client-
// facing no-keyring contract is the four coded sentinels of
// internal/interact/keyringprobe.go:44-60 — asserted VERBATIM, never
// paraphrased. Run: node --test experiments/
//
// Static tests read the script and assert its contract without running the
// wrapper (they hold on any machine, CI included). The live stripped-PATH arm
// is GUARDED behind ZCODE_ACP_VM_E2E_BIN (the test/realdesktop precedent:
// the guarded harness runs before the human's field run) — it needs a built
// binary and a machine that has secret-tool on its PATH to prove that
// stripping PATH yields the binary-missing arm.

import { test } from 'node:test';
import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import path from 'node:path';

const scriptPath = path.join(
  path.dirname(fileURLToPath(import.meta.url)),
  '..',
  'scripts',
  'vm-e2e.sh',
);

// The four coded sentinels — byte-for-byte from
// internal/interact/keyringprobe.go (ErrKeyringBinaryMissing,
// ErrKeyringNoDesktopSession, ErrKeyringNoEntry, ErrKeyringLookupTimeout).
// The no-desktop-session sentence is source-concatenated across two Go lines
// but prints as one; the joined form below is the runtime form.
const SENTINELS = {
  binaryMissing:
    'keyring probe: secret-tool not found (install libsecret-tools on Ubuntu: sudo apt install libsecret-tools)',
  noDesktopSession:
    'keyring probe: no desktop session available (the keyring bus cannot autolaunch headless — run inside your desktop session)',
  noEntry:
    'keyring probe: no ZCode desktop login found in the keyring (log in via the ZCode desktop first)',
  lookupTimeout:
    'keyring probe: keyring lookup did not answer within its bound',
};

const scriptSource = () => readFileSync(scriptPath, 'utf8');

test('no-keyring stage asserts the four coded keyring sentinels verbatim', () => {
  const src = scriptSource();

  // Each sentinel sentence must appear character-for-character in the
  // script's expected-output table — grep -F fixed-string matching material,
  // never a paraphrase.
  for (const [arm, sentence] of Object.entries(SENTINELS)) {
    assert.ok(
      src.includes(sentence),
      `the ${arm} sentinel sentence is missing from scripts/vm-e2e.sh (must be verbatim from keyringprobe.go): ${sentence}`,
    );
  }

  // All four assertions are present as separate sentences (>= 4 occurrences
  // of the shared prefix — one per sentinel).
  const count = (src.match(/keyring probe: /g) ?? []).length;
  assert.ok(
    count >= 4,
    `expected >= 4 'keyring probe: ' occurrences (one per sentinel), got ${count}`,
  );
});

test('no-keyring stage wraps the attempt in a bounded timeout(1)', () => {
  const src = scriptSource();

  // The stage bounds its attempt with timeout(1) over a declared bound —
  // belt-and-braces over the probe's own 5s KeyringProbeTimeout (T-07-21:
  // a misclassified attempt can never hang the stage).
  assert.ok(
    /TIMEOUT_ARM=\d+/.test(src),
    'the no-keyring attempt bound (TIMEOUT_ARM=<seconds>) is not declared',
  );
  assert.ok(
    src.includes('timeout "$TIMEOUT_ARM"'),
    'the no-keyring attempt is not wrapped in timeout "$TIMEOUT_ARM"',
  );
});

test('live stripped-PATH arm surfaces the binary-missing sentinel (guarded)', { skip: !process.env.ZCODE_ACP_VM_E2E_BIN }, () => {
  const bin = process.env.ZCODE_ACP_VM_E2E_BIN;

  // The stage's first arm strips secret-tool from the child's PATH and runs
  // the coding-plan authenticate: the probe must surface the exact
  // binary-missing sentence and the script's check must PASS on matching it.
  const res = spawnSync('bash', [scriptPath, '--stage', 'no-keyring', '--binary', bin], {
    encoding: 'utf8',
    timeout: 300_000,
  });

  const out = `${res.stdout ?? ''}${res.stderr ?? ''}`;
  assert.ok(
    out.includes(SENTINELS.binaryMissing),
    `the live stripped-PATH arm did not surface the binary-missing sentinel verbatim; stage output:\n${out.slice(0, 2000)}`,
  );
  assert.ok(
    /PASS no-keyring/.test(out),
    `the no-keyring stage did not PASS its binary-missing check; stage output:\n${out.slice(0, 2000)}`,
  );
  assert.strictEqual(res.status, 0, `stage exit ${res.status}; output:\n${out.slice(0, 2000)}`);
});
