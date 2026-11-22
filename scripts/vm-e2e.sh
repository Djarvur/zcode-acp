#!/usr/bin/env bash
# scripts/vm-e2e.sh — the clean-Ubuntu-GNOME VM scenario (QA-02, D-05):
# install -> configure (both auth paths) -> run -> the no-keyring error path
# asserted. One script that walks the whole v1 scenario so it is reproducible
# from the repo on a fresh machine, without guessing any step.
#
# Purpose:
#   The script encodes the field lessons of phases 4-6:
#   - the desktop-discovery probe order the wrapper itself drives
#     (cmd/zcode-acp discoverInstall: env/flag override -> desktop entry ->
#     AppImage paths -> package paths; docs/protocol/install-discovery.md);
#   - the runtime-only dependency posture (Pitfall 6: a static binary needs
#     NO dev toolchain on the VM — the script asserts runtime deps only);
#   - the four coded keyring sentinels as the client-facing no-keyring
#     contract (internal/interact/keyringprobe.go, asserted VERBATIM);
#   - the paired provider-config env fact: the wrapper's child env passes
#     ZCODE_BUILTIN_PROVIDER_CONFIG_FILE AND ZCODE_PERSONAL_PROVIDER_CONFIG_FILE
#     together (internal/supervisor/spawn.go ChildEnv) — the coding-plan stage
#     asserts the FACT BY READ-BACK (the session/new reply must LIST
#     account-provider models), never by trusting any ack.
#
# Usage:
#   scripts/vm-e2e.sh [--stage NAME] [--binary PATH] [--require-auth]
#
#     --stage NAME     one of: discovery | run | no-keyring | auth-apikey |
#                      auth-codingplan | all (default: all)
#     --binary PATH    path to the zcode-acp binary (or env ZCODE_ACP_BIN);
#                      required by the wire stages (run, no-keyring,
#                      auth-apikey, auth-codingplan)
#     --require-auth   make the auth stages' SKIP (missing key / missing
#                      keyring facility) a hard FAIL instead of a skip
#
#   The binary is built-or-fetched OFF the VM (Pitfall 6): build it on a dev
#   machine (`go build -o zcode-acp ./cmd/zcode-acp`) or fetch a release
#   artifact and verify its checksum against the publisher's sidecar (the
#   sha512 precedent of docs/protocol/install-discovery.md). This script
#   NEVER downloads anything, NEVER pipes a network response into a shell,
#   and NEVER installs a toolchain or package on the VM — dependency asserts
#   are presence checks with actionable hints only.
#
# Discovery probe order (documented; the wrapper's own discovery is
# authoritative — this stage mirrors it read-only for diagnostics):
#   1. env/flag override (ZCODE_ACP_APP_SERVER)
#   2. desktop entry (xdg-mime query default x-scheme-handler/zcode -> Exec)
#   3. AppImage paths (~/Applications/ZCode*.AppImage, ~/Applications/zcode-*/squashfs-root)
#   4. package paths (/opt/ZCode/resources/glm/zcode.cjs, dpkg-query zcode)
#
# Runtime dependencies (asserted, never installed):
#   - bash 4+ (this script)
#   - the zcode-acp binary (see --binary above)
#   - the ZCode desktop install — required by the run/auth-codingplan stages
#     (the wrapper discovers it; the run stage proves discovery by working)
#   - secret-tool (libsecret-tools) — required ONLY by the coding-plan stage;
#     its absence is exactly the binary-missing sentinel arm the no-keyring
#     stage asserts
#
# Credential discipline (T-07-20): env-var NAMES only in output and logs —
# ZCODE_ACP_API_KEY is referenced by name and read from the environment; its
# VALUE is never echoed, logged, or captured. Run experiments/scan-secrets.sh
# after any evidence capture.
#
# Safety: no destructive step exists; re-running after an interruption is
# safe (temp dirs are created under mktemp and removed on exit; the ZCode
# desktop and its data are never touched).
#
# Output contract:
#   PASS <check-name>[ — <detail>]   the named check passed
#   FAIL <check-name> — <detail>     the named check failed (first FAIL is
#                                    named again by the summary line)
#   SKIP <check-name> — <reason>     the check is skipped-not-failed (stage
#                                    precondition absent; see --require-auth)
#   INFO <message>                   diagnostic detail, never affects exit
#   final summary line: every check's outcome, then
#     vm-e2e: PASS (<n> checks, <m> skipped)   — exit 0
#     vm-e2e: FAIL (first failed check: <name>) — exit 1
#
# Exit codes:
#   0  every check passed or skipped
#   1  at least one check failed (the summary names the first)
#   2  usage error (unknown stage/flag, bad arguments)

set -euo pipefail
export LC_ALL=C
umask 077

SELF=$(readlink -f "$0")

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

STAGE="all"
BINARY="${ZCODE_ACP_BIN:-}"
REQUIRE_AUTH=0
INTERNAL_ARM=0

# Per-leg read bounds (seconds). A hung wrapper is caught by the bound and
# surfaces as a FAIL — a stage never hangs on an unanswerable child.
TIMEOUT_INIT=60       # initialize reply (the wrapper serves ACP immediately)
TIMEOUT_SESSION=180   # session/new (first leg may spawn the desktop child)
TIMEOUT_PROMPT=180    # one minimal real turn
TIMEOUT_AUTH=120      # authenticate (coding-plan runs the probe + mint chain)
TIMEOUT_ARM=180       # the bounded no-keyring attempt (timeout(1) wrapper)

usage_error() {
	echo "USAGE: $0 [--stage discovery|run|no-keyring|auth-apikey|auth-codingplan|all]" >&2
	echo "              [--binary PATH] [--require-auth]" >&2
	exit 2
}

while [ $# -gt 0 ]; do
	case "$1" in
	--stage)
		[ $# -ge 2 ] || usage_error
		STAGE=$2
		shift 2
		;;
	--binary)
		[ $# -ge 2 ] || usage_error
		BINARY=$2
		shift 2
		;;
	--require-auth)
		REQUIRE_AUTH=1
		shift
		;;
	--internal-arm) # hidden: the bounded no-keyring attempt's self-invocation
		INTERNAL_ARM=1
		shift
		;;
	--help | -h)
		sed -n '2,60p' "$SELF" | sed 's/^# \{0,1\}//'
		exit 0
		;;
	*)
		echo "vm-e2e: unknown argument: $1" >&2
		usage_error
		;;
	esac
done

case "$STAGE" in
discovery | run | no-keyring | auth-apikey | auth-codingplan | all | nokeyring-arm) ;;
*)
	echo "vm-e2e: unknown stage: $STAGE" >&2
	usage_error
	;;
esac

# ---------------------------------------------------------------------------
# Named-check machinery
# ---------------------------------------------------------------------------

CHECK_ORDER=()
declare -A CHECK_STATUS=() # name -> pass|fail|skip
declare -A CHECK_DETAIL=() # name -> detail line
FIRST_FAIL=""

check() { # check <name> <pass|fail|skip> [detail]
	local name=$1 status=$2 detail=${3:-}
	CHECK_ORDER+=("$name")
	CHECK_STATUS["$name"]=$status
	CHECK_DETAIL["$name"]=$detail
	case "$status" in
	pass) echo "PASS $name${detail:+ — $detail}" ;;
	fail)
		echo "FAIL $name${detail:+ — $detail}"
		[ -z "$FIRST_FAIL" ] && FIRST_FAIL=$name
		;;
	skip) echo "SKIP $name — $detail" ;;
	esac
}

info() { echo "INFO $*"; }

# auth_stage_skip decides a missing auth-stage precondition: skipped-not-failed
# by default, a hard FAIL under --require-auth (the stage caller cannot verify
# the scenario it was asked to verify).
auth_stage_skip() {
	local name=$1 reason=$2
	if [ "$REQUIRE_AUTH" -eq 1 ]; then
		check "$name" fail "$reason (--require-auth makes this precondition mandatory)"
	else
		check "$name" skip "$reason"
	fi
}

summary() {
	local pass=0 fail=0 skip=0 name
	echo "---- vm-e2e summary (stage: $STAGE) ----"
	for name in "${CHECK_ORDER[@]}"; do
		case "${CHECK_STATUS[$name]}" in
		pass) pass=$((pass + 1)) ;;
		fail) fail=$((fail + 1)) ;;
		skip) skip=$((skip + 1)) ;;
		esac
		local upper
		case "${CHECK_STATUS[$name]}" in
		pass) upper=PASS ;;
		fail) upper=FAIL ;;
		skip) upper=SKIP ;;
		esac
		printf '%s %s%s\n' \
			"$upper" "$name" \
			"${CHECK_DETAIL[$name]:+ — ${CHECK_DETAIL[$name]}}"
	done
	if [ -n "$FIRST_FAIL" ]; then
		echo "vm-e2e: FAIL (first failed check: $FIRST_FAIL)"
		return 1
	fi
	echo "vm-e2e: PASS ($pass checks, $skip skipped)"
}

# Cleanup: the script's own temp state only — never the desktop, never the
# install, never anything it did not create.
STRIP_DIR=""
cleanup() {
	if [ -n "$RUN_PID" ]; then
		kill "$RUN_PID" 2>/dev/null || true
	fi
	if [ -n "$STRIP_DIR" ] && [ -d "$STRIP_DIR" ]; then
		rm -rf "$STRIP_DIR"
	fi
}
trap cleanup EXIT

# ---------------------------------------------------------------------------
# Scripted stdio ACP client (the arc driver)
#
# The agent runs as a bash coprocess; every blocking read is bounded by
# read -t so a hung wrapper surfaces as a FAIL, never a hang. Frames are
# newline-delimited JSON on the agent's stdout; the reply to request id N is
# recognized as a line carrying "id":N, and NO "method" member (an
# agent->client request carries both; a notification carries neither result
# nor error with our ids).
#
# The pid is captured into RUN_PID the moment the coprocess spawns: bash
# UNSETS the AGENT_PID variable once the coprocess is reaped, so the EXIT
# trap must never read the managed variable (observed: set -u turns the
# reaping-then-trap sequence into an unbound-variable exit 1).
# ---------------------------------------------------------------------------

AGENT_FD=""
RUN_PID=""
ARC_STDERR=""

agent_start() { # agent_start [VAR=VALUE ...] — extra env applies to the child only
	local errfile
	errfile=$(mktemp "${TMPDIR:-/tmp}/vm-e2e-agent.XXXXXX")
	ARC_STDERR=$errfile

	if [ $# -gt 0 ]; then
		coproc AGENT { env "$@" "$BINARY" 2>"$errfile"; }
	else
		coproc AGENT { "$BINARY" 2>"$errfile"; }
	fi
	AGENT_FD="${AGENT[0]}"
	RUN_PID="$AGENT_PID"
}

agent_send() { # agent_send <json-line>
	printf '%s\n' "$1" >&"${AGENT[1]}"
}

agent_expect() { # agent_expect <request-id> <timeout-seconds> — sets REPLY_LINE
	local want_id=$1 bound=$2 line
	REPLY_LINE=""
	AGENT_DEAD=0

	while IFS= read -r -u "$AGENT_FD" -t "$bound" line; do
		case "$line" in
		*"\"id\":$want_id,"*)
			case "$line" in
			*'"method":'*) continue ;; # an agent->client request, not our reply
			esac
			REPLY_LINE=$line
			return 0
			;;
		esac
	done

	# read timed out (rc>128) or the agent's stdout closed early (EOF, rc>1)
	kill -0 "$RUN_PID" 2>/dev/null || AGENT_DEAD=1
	return 1
}

agent_stderr_tail() { # last stderr lines — the wrapper's actionable diagnostics
	tail -4 "$ARC_STDERR" 2>/dev/null | tr '\n' ' ' || true
}

agent_stop() {
	if [ -n "$RUN_PID" ]; then
		kill "$RUN_PID" 2>/dev/null || true
		RUN_PID=""
	fi
}

json_field() { # json_field <json-line> <member> — first string value of member
	printf '%s' "$1" | sed -n 's/.*"'"$2"'":"\([^"]*\)".*/\1/p'
}

# ---------------------------------------------------------------------------
# Stage: discovery — mirror the wrapper's probe order read-only
# ---------------------------------------------------------------------------

stage_discovery() {
	info "documented probe order: env/flag override -> desktop entry -> AppImage paths -> package paths (the wrapper's discoverInstall drives the same order; the run stage proves discovery by working)"

	local found=0 row_detail=() out exec_target

	# Row 1: env/flag override (the wrapper's ZCODE_ACP_APP_SERVER / --app-server)
	if [ -n "${ZCODE_ACP_APP_SERVER:-}" ]; then
		row_detail+=("override: ZCODE_ACP_APP_SERVER is set (NAME only reported; the wrapper honors it first)")
		found=$((found + 1))
	else
		row_detail+=("override: not set (ok — on-disk discovery proceeds)")
	fi

	# Row 2: desktop entry
	out=$(xdg-mime query default x-scheme-handler/zcode 2>/dev/null || true)
	if [ -n "$out" ]; then
		exec_target=""
		local f
		for f in "${XDG_DATA_HOME:-$HOME/.local/share}/applications/$out" \
			/usr/share/applications/"$out" /usr/local/share/applications/"$out"; do
			if [ -f "$f" ]; then
				exec_target=$(sed -n 's/^Exec=//p' "$f" | head -1)
				break
			fi
		done
		row_detail+=("desktop entry: $out${exec_target:+ (Exec: $exec_target)}")
		found=$((found + 1))
	else
		row_detail+=("desktop entry: no x-scheme-handler/zcode default (hint: install the ZCode desktop; xdg-utils provides xdg-mime)")
	fi

	# Row 3: AppImage paths
	local appimg=""
	if ls "$HOME"/Applications/ZCode*.AppImage >/dev/null 2>&1; then
		appimg=$(ls "$HOME"/Applications/ZCode*.AppImage 2>/dev/null | head -1)
	fi
	local unpacked=""
	if compgen -G "$HOME/Applications/zcode-*/squashfs-root" >/dev/null 2>&1; then
		unpacked=$(ls -d "$HOME"/Applications/zcode-*/squashfs-root 2>/dev/null | head -1)
	fi
	if [ -n "$appimg" ] || [ -n "$unpacked" ]; then
		row_detail+=("AppImage paths: ${appimg:-}${appimg:+ + }${unpacked:-}${unpacked:+ (unpacked)}")
		found=$((found + 1))
	else
		row_detail+=("AppImage paths: no ~/Applications/ZCode*.AppImage, no ~/Applications/zcode-*/squashfs-root")
	fi

	# Row 4: package paths
	local pkg=""
	if [ -f /opt/ZCode/resources/glm/zcode.cjs ]; then
		pkg="/opt/ZCode/resources/glm/zcode.cjs"
	elif command -v dpkg-query >/dev/null 2>&1 && dpkg-query -W -f='${Status}' zcode 2>/dev/null | grep -q 'install ok installed'; then
		pkg="dpkg: zcode installed (deb layout)"
	elif [ -x /usr/bin/zcode ]; then
		pkg="/usr/bin/zcode"
	fi
	if [ -n "$pkg" ]; then
		row_detail+=("package paths: $pkg")
		found=$((found + 1))
	else
		row_detail+=("package paths: no /opt/ZCode/resources/glm/zcode.cjs, no installed zcode package")
	fi

	for row in "${row_detail[@]}"; do
		info "$row"
	done

	if [ "$found" -gt 0 ]; then
		check "discovery-install-found" pass "probe rows: $found hit(s)"
	else
		# The wrapper's nothing-found diagnostics name every probe row; this
		# script's mirror names them too — every row's output is in the detail.
		check "discovery-install-found" fail \
			"no probe row found an install — ${row_detail[*]} (install the ZCode desktop, or set ZCODE_ACP_APP_SERVER)"
	fi
}

# ---------------------------------------------------------------------------
# Stage: run — version probe + the scripted ACP arc (initialize -> session/new
# -> one minimal prompt) against the real binary
# ---------------------------------------------------------------------------

version_floor_ok() { # version_floor_ok <X.Y.Z> — the 07-04 floor (>= 18.1.9)
	local v=$1
	awk -v v="$v" 'BEGIN {
		split("18.1.9", f, ".")
		split(v, a, ".")
		exit !((a[1] > f[1]) || (a[1] == f[1] && (a[2] > f[2] || (a[2] == f[2] && a[3] >= f[3]))))
	}'
}

check_version_floor() {
	local raw want_floor="18.1.9"
	raw=$("$BINARY" omp --version 2>/dev/null || true)
	local semver
	semver=$(printf '%s' "$raw" | sed -n 's/^zcode-acp \([0-9]*\.[0-9]*\.[0-9]*\).*/\1/p')
	if [ -z "$semver" ]; then
		check "binary-version-floor" fail \
			"'$BINARY omp --version' printed no 'zcode-acp <semver>' line (got: ${raw:-<empty>})"
	elif version_floor_ok "$semver"; then
		check "binary-version-floor" pass "$semver >= $want_floor"
	else
		check "binary-version-floor" fail \
			"version $semver is below the card's floor $want_floor (07-04)"
	fi
}

stage_run() {
	if [ -z "$BINARY" ] || [ ! -x "$BINARY" ]; then
		check "binary-present" fail "no executable binary — pass --binary PATH or set ZCODE_ACP_BIN (build-or-fetch off the VM; see the header)"
		return 0
	fi
	check "binary-present" pass "$BINARY"

	check_version_floor

	agent_start
	local line sid stop reason err

	# Leg 1: initialize
	agent_send '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":1,"clientCapabilities":{"fs":{"readTextFile":false,"writeTextFile":false}}}}'
	if agent_expect 1 "$TIMEOUT_INIT"; then
		if [[ "$REPLY_LINE" == *'"protocolVersion":1'* ]]; then
			check "acp-initialize-protocolversion" pass "protocolVersion 1, agentInfo $(json_field "$REPLY_LINE" version)"
		else
			check "acp-initialize-protocolversion" fail "initialize reply carries no protocolVersion 1: $REPLY_LINE"
		fi
	else
		if [ "$AGENT_DEAD" -eq 1 ]; then
			check "acp-initialize-protocolversion" fail \
				"the wrapper exited before initialize — discovery likely found no desktop; wrapper diagnostics: $(agent_stderr_tail)"
		else
			check "acp-initialize-protocolversion" fail "no initialize reply within ${TIMEOUT_INIT}s (never hangs; wrapper diagnostics: $(agent_stderr_tail))"
		fi
		agent_stop
		return 0
	fi

	# Leg 2: session/new
	agent_send '{"jsonrpc":"2.0","id":2,"method":"session/new","params":{"cwd":"/tmp","mcpServers":[]}}'
	if agent_expect 2 "$TIMEOUT_SESSION"; then
		sid=$(json_field "$REPLY_LINE" sessionId)
		if [ -n "$sid" ]; then
			check "acp-session-new" pass "sessionId ${sid%%-*}... (configOptions listed: $(printf '%s' "$REPLY_LINE" | grep -c 'configOptions' || true))"
		else
			check "acp-session-new" fail "session/new reply carries no sessionId: $REPLY_LINE"
		fi
	else
		check "acp-session-new" fail "no session/new reply within ${TIMEOUT_SESSION}s (wrapper diagnostics: $(agent_stderr_tail))"
		agent_stop
		return 0
	fi

	# Leg 3: one minimal prompt — the arm is RECORDED, never assumed: on an
	# authenticated desktop one real turn completes; unauthenticated the turn
	# fails with the backend's honest auth error (no hang, no crash; the
	# unauthenticated arm spends no quota).
	agent_send "{\"jsonrpc\":\"2.0\",\"id\":3,\"method\":\"session/prompt\",\"params\":{\"sessionId\":\"$sid\",\"prompt\":[{\"type\":\"text\",\"text\":\"Reply with the single word: pong\"}]}}"
	if agent_expect 3 "$TIMEOUT_PROMPT"; then
		case "$REPLY_LINE" in
		*'"error"'*)
			reason=$(json_field "$REPLY_LINE" message)
			check "acp-prompt-turn" pass "arm=auth-error — the backend's honest error arrived cleanly (bounded), no hang, no crash: ${reason:0:160}"
			;;
		*)
			stop=$(json_field "$REPLY_LINE" stopReason)
			check "acp-prompt-turn" pass "arm=completed — one minimal real turn resolved (stopReason: ${stop:-unknown})"
			;;
		esac
	else
		check "acp-prompt-turn" fail \
			"the prompt leg neither resolved nor errored within ${TIMEOUT_PROMPT}s — a hang class (arm=unknown; wrapper diagnostics: $(agent_stderr_tail))"
	fi

	agent_stop
}

# ---------------------------------------------------------------------------
# Stage: no-keyring — the four coded sentinels (task 2 lands this stage)
# ---------------------------------------------------------------------------

stage_no_keyring() {
	check "no-keyring-stage" skip "stage not scripted yet (07-07 task 2: the four coded keyring sentinels asserted verbatim)"
}

# ---------------------------------------------------------------------------
# Stage: auth-apikey (task 3 lands this stage)
# ---------------------------------------------------------------------------

stage_auth_apikey() {
	check "auth-apikey-stage" skip "stage not scripted yet (07-07 task 3: the api-key path asserted by read-back)"
}

# ---------------------------------------------------------------------------
# Stage: auth-codingplan (task 3 lands this stage)
# ---------------------------------------------------------------------------

stage_auth_codingplan() {
	check "auth-codingplan-stage" skip "stage not scripted yet (07-07 task 3: the paired-env fact asserted by read-back)"
}

# ---------------------------------------------------------------------------
# The bounded no-keyring attempt (internal arm): the no-keyring stage
# re-invokes this script under timeout(1); this arm builds the stripped PATH,
# runs initialize -> authenticate{zai-individual-coding-plan} against the
# binary, and prints EXACTLY ONE tagged line for the caller to classify:
#   ARM-REPLY <json-line>   the authenticate reply arrived (result or error)
#   ARM-EOF                 the wrapper exited before replying
#   ARM-TIMEOUT             no reply within the read bound
# The caller's environment is never mutated: the stripped PATH applies to the
# child process only.
# ---------------------------------------------------------------------------

build_stripped_path() {
	STRIP_DIR=$(mktemp -d "${TMPDIR:-/tmp}/vm-e2e-strip.XXXXXX")
	local d f b
	local IFS=':'
	read -ra _dirs <<< "$PATH"
	for d in "${_dirs[@]}"; do
		[ -d "$d" ] || continue
		for f in "$d"/*; do
			[ -e "$f" ] || [ -L "$f" ] || continue
			b=${f##*/}
			case "$b" in
			secret-tool*) continue ;;
			esac
			ln -s "$f" "$STRIP_DIR/$b" 2>/dev/null || true
		done
	done
}

stage_nokeyring_arm() {
	build_stripped_path
	agent_start "PATH=$STRIP_DIR"

	agent_send '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":1,"clientCapabilities":{"fs":{"readTextFile":false,"writeTextFile":false}}}}'
	if ! agent_expect 1 "$TIMEOUT_INIT"; then
		echo "ARM-EOF"
		return 0
	fi

	agent_send '{"jsonrpc":"2.0","id":2,"method":"authenticate","params":{"methodId":"zai-individual-coding-plan"}}'
	if agent_expect 2 "$TIMEOUT_AUTH"; then
		echo "ARM-REPLY $REPLY_LINE"
	else
		echo "ARM-TIMEOUT"
	fi
}

# ---------------------------------------------------------------------------
# Dispatch
# ---------------------------------------------------------------------------

run_stages() {
	case "$STAGE" in
	all)
		stage_discovery
		stage_run
		stage_no_keyring
		stage_auth_apikey
		stage_auth_codingplan
		;;
	discovery) stage_discovery ;;
	run) stage_run ;;
	no-keyring) stage_no_keyring ;;
	auth-apikey) stage_auth_apikey ;;
	auth-codingplan) stage_auth_codingplan ;;
	nokeyring-arm) stage_nokeyring_arm ;; # internal: bounded attempt, no checks
	esac
}

if [ "$INTERNAL_ARM" -eq 1 ]; then
	# Internal arm: no checks, no summary — exactly one tagged line on stdout.
	[ -n "$BINARY" ] || usage_error
	run_stages
	exit 0
fi

run_stages
summary
