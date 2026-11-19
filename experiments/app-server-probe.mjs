#!/usr/bin/env node
// experiments/app-server-probe.mjs — scenario-flow spawn/JSONL harness (Pattern 2,
// validated live against the shipped 3.18.2 binary during phase research).
//
// NEVER wire this into CI: it spawns the real desktop bundle and (for the turn
// flow) spends account quota — see experiments/README.md ("Ground rules").
// No experiment here ever mutates account state: there is deliberately no
// api_keys POST anywhere in this file.
//
// Usage:
//   node experiments/app-server-probe.mjs \
//     --electron <bundle-root>/zcode \
//     --cjs <bundle-root>/resources/glm/zcode.cjs \
//     --builtin <builtin-config-file> \
//     --flow <flow-name> \
//     --raw-out <scratch-path-outside-repo> \
//     --window-ms 12000 [--quiet-ms 2500] [--workspace-dir <dir>]
//
// Behavior:
//   - spawns `<electron> <cjs> app-server --stdio` with ELECTRON_RUN_AS_NODE=1 and
//     ZCODE_BUILTIN_PROVIDER_CONFIG_FILE=<builtin>, stdio pipes, own process group;
//   - HOLDS STDIN OPEN until the collection window ends — writing a request and
//     closing stdin early makes the app-server exit cleanly BEFORE answering
//     (exit 0 + empty capture is the warning sign);
//   - appends timestamped stdout lines to the raw file (one JSON object per wire
//     line: {"ts":"<ISO>","line":"<raw wire line>"});
//   - mirrors child stderr to this process's stderr, capped at 64 KiB;
//   - the FIRST line of the raw file is the capture header JSON with the builtin
//     revision READ AT CAPTURE TIME (stale-evidence guard against desktop
//     auto-update mid-phase);
//   - clean shutdown: stdin.end(), then a process-group SIGKILL backstop.
//
// Machine paths arrive as CLI arguments only — nothing machine-private is
// hardcoded here, and a --raw-out INSIDE this repository is refused (raw captures
// are born outside the repo; experiments/scrub-capture.mjs is the only path into
// experiments/captures/).

import { spawn } from 'node:child_process';
import { appendFileSync, readFileSync, writeFileSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const SCRIPT_DIR = dirname(fileURLToPath(import.meta.url));
const REPO_ROOT = path.resolve(SCRIPT_DIR, '..');
const STDERR_CAP_BYTES = 64 * 1024;
const GRACE_MS = 4000; // between stdin.end() and the SIGKILL backstop

// ---------------------------------------------------------------------------
// Flow registry — the dispatch seam. Plans 01-02 add `subscribe` and `turn`
// here; each flow returns a list of { atMs, msg } steps to send.
// ---------------------------------------------------------------------------
const FLOWS = {
  probe(ctx) {
    return [
      { atMs: 1500, msg: { id: 1, method: 'provider/refreshRegistry', params: { reason: 'probe' } } },
    ];
  },

  // E3 (DISC-04): path-as-key accepted, then an arbitrary host-chosen string key.
  // Registry-level only — no turn, no quota.
  workspacekey(ctx) {
    const ws = ctx.workspaceDir;
    return [
      {
        atMs: 1500,
        msg: { id: 1, method: 'session/create', params: { workspace: { workspacePath: ws, workspaceKey: ws } } },
      },
      {
        atMs: 4000,
        msg: {
          id: 2,
          method: 'session/create',
          params: { workspace: { workspacePath: ws, workspaceKey: 'probe-arbitrary-host-key-01' } },
        },
      },
    ];
  },
};

// Experiment ids per flow (D-04: [V-EXP <id>] resolves to
// experiments/captures/<id>*). Flows added by plan 01-02 are pre-registered here
// so the seam is explicit.
const CAPTURE_IDS = {
  probe: 'EXP-PROBE',
  workspacekey: 'EXP-WORKSPACEKEY-01',
  subscribe: 'EXP-SUBSCRIBE-01', // flow lands in plan 01-02
  turn: 'EXP-TURN-01', // flow lands in plan 01-02
};

// --- args ------------------------------------------------------------------
function parseArgs(argv) {
  const out = {
    electron: null,
    cjs: null,
    builtin: null,
    flow: null,
    rawOut: null,
    windowMs: null,
    quietMs: 2500,
    workspaceDir: null,
  };
  const required = ['electron', 'cjs', 'builtin', 'flow', 'rawOut', 'windowMs'];
  const map = {
    '--electron': 'electron',
    '--cjs': 'cjs',
    '--builtin': 'builtin',
    '--flow': 'flow',
    '--raw-out': 'rawOut',
    '--window-ms': 'windowMs',
    '--quiet-ms': 'quietMs',
    '--workspace-dir': 'workspaceDir',
  };
  for (let i = 0; i < argv.length; i++) {
    const key = map[argv[i]];
    if (!key || i + 1 >= argv.length) {
      process.stderr.write(`app-server-probe: missing or unknown flag: ${argv[i]}\n`);
      return null;
    }
    out[key] = argv[++i];
  }
  for (const k of required) {
    if (out[k] == null) {
      process.stderr.write(`app-server-probe: missing required flag --${k.replace(/[A-Z]/g, (c) => '-' + c.toLowerCase())}\n`);
      return null;
    }
  }
  out.windowMs = Number(out.windowMs);
  out.quietMs = Number(out.quietMs);
  if (!Number.isFinite(out.windowMs) || out.windowMs <= 0) {
    process.stderr.write('app-server-probe: --window-ms must be a positive number\n');
    return null;
  }
  if (!Number.isFinite(out.quietMs) || out.quietMs <= 0) {
    process.stderr.write('app-server-probe: --quiet-ms must be a positive number\n');
    return null;
  }
  if (out.workspaceDir == null && out.rawOut) {
    // Default: the scratch directory holding the raw capture (guarded to be
    // outside the repo below).
    out.workspaceDir = dirname(path.resolve(out.rawOut));
  }
  return out;
}

function usage() {
  process.stderr.write(
    'usage: node experiments/app-server-probe.mjs --electron <bundle-root>/zcode \\\n' +
      '  --cjs <bundle-root>/resources/glm/zcode.cjs --builtin <builtin-config-file> \\\n' +
      '  --flow <probe|workspacekey> --raw-out <scratch-path-outside-repo> \\\n' +
      '  --window-ms 12000 [--quiet-ms 2500] [--workspace-dir <dir>]\n',
  );
}

const flags = parseArgs(process.argv.slice(2));
if (!flags) {
  usage();
  process.exit(1);
}

// --- scrub-first guard: raw captures are born OUTSIDE the repo ----------------
const rawOut = path.resolve(flags.rawOut);
if (rawOut === REPO_ROOT || rawOut.startsWith(REPO_ROOT + path.sep)) {
  process.stderr.write(
    'app-server-probe: refusing --raw-out inside the repository — raw captures are\n' +
      'born outside the repo (D-02); point it at a scratch directory and scrub into\n' +
      'experiments/captures/ with experiments/scrub-capture.mjs.\n',
  );
  process.exit(1);
}

if (!Object.hasOwn(FLOWS, flags.flow)) {
  const pending = Object.keys(CAPTURE_IDS).filter((k) => !Object.hasOwn(FLOWS, k));
  process.stderr.write(
    `app-server-probe: unknown flow "${flags.flow}" (known: ${Object.keys(FLOWS).join(', ')}` +
      `${pending.length ? `; not implemented yet: ${pending.join(', ')}` : ''})\n`,
  );
  process.exit(1);
}

// --- capture header (builtinRevision read AT CAPTURE TIME, Pitfall 5) ---------
let builtinRevision = 'unknown';
try {
  const parsed = JSON.parse(readFileSync(flags.builtin, 'utf8'));
  if (parsed && typeof parsed.revision !== 'undefined') builtinRevision = String(parsed.revision);
} catch {
  // recorded as "unknown" — the header still dates and anchors the capture
}

const captureId = CAPTURE_IDS[flags.flow];
writeFileSync(
  rawOut,
  JSON.stringify({
    capture: captureId,
    date: new Date().toISOString(),
    desktopVersion: '3.18.2',
    builtinRevision,
    host: os.release(),
  }) + '\n',
);

// --- spawn (own process group so the backstop can kill the whole tree) --------
const child = spawn(flags.electron, [flags.cjs, 'app-server', '--stdio'], {
  env: {
    ...process.env,
    ELECTRON_RUN_AS_NODE: '1',
    ZCODE_BUILTIN_PROVIDER_CONFIG_FILE: flags.builtin,
  },
  stdio: ['pipe', 'pipe', 'pipe'],
  detached: true,
});

child.stdin.on('error', () => {}); // EPIPE after the child exits is expected

let capturedLines = 0;
let stdoutBuf = '';
child.stdout.setEncoding('utf8');
child.stdout.on('data', (chunk) => {
  stdoutBuf += chunk;
  let idx;
  while ((idx = stdoutBuf.indexOf('\n')) >= 0) {
    const line = stdoutBuf.slice(0, idx);
    stdoutBuf = stdoutBuf.slice(idx + 1);
    appendFileSync(rawOut, JSON.stringify({ ts: new Date().toISOString(), line }) + '\n');
    capturedLines++;
  }
});

let stderrCapped = 0;
child.stderr.setEncoding('utf8');
child.stderr.on('data', (chunk) => {
  if (stderrCapped < STDERR_CAP_BYTES) {
    stderrCapped += chunk.length;
    process.stderr.write(chunk.slice(0, Math.max(0, STDERR_CAP_BYTES - (stderrCapped - chunk.length))));
  }
});

const exited = new Promise((resolve) => {
  child.on('exit', (code, sig) => resolve({ code, sig }));
  child.on('error', (err) => resolve({ spawnError: err.message }));
});

function send(obj) {
  child.stdin.write(JSON.stringify(obj) + '\n');
}

function killGroup() {
  try {
    process.kill(-child.pid, 'SIGKILL');
  } catch {
    // already gone — fine
  }
}

// --- run the flow --------------------------------------------------------------
const steps = FLOWS[flags.flow]({
  workspaceDir: flags.workspaceDir,
  windowMs: flags.windowMs,
  quietMs: flags.quietMs,
});
const timers = steps.map((s) => setTimeout(() => send(s.msg), s.atMs));

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
let exitInfo = await Promise.race([exited, sleep(flags.windowMs).then(() => null)]);

let killed = false;
if (!exitInfo) {
  // collection window over: clean shutdown first, SIGKILL group as backstop
  child.stdin.end();
  exitInfo = await Promise.race([exited, sleep(GRACE_MS).then(() => null)]);
  if (!exitInfo) {
    killGroup();
    killed = true;
    exitInfo = await exited;
  }
} else {
  child.stdin.end();
}
for (const t of timers) clearTimeout(t);

if (exitInfo.spawnError) {
  process.stderr.write(`app-server-probe: spawn failed: ${exitInfo.spawnError}\n`);
  process.exit(1);
}

const earlyZero = exitInfo.code === 0 && capturedLines === 0;
process.stderr.write(
  `app-server-probe: flow=${flags.flow} capture=${captureId} exit=${exitInfo.code}` +
    ` sig=${exitInfo.sig || '-'} lines=${capturedLines} killed=${killed} raw=${rawOut}\n`,
);
if (earlyZero) {
  process.stderr.write(
    'app-server-probe: WARNING — exit 0 with zero captured lines is the stdin-EOF\n' +
      'pitfall signature (or a bad --electron/--cjs path); the run produced no wire truth.\n',
  );
  process.exit(1);
}
process.exit(0);
