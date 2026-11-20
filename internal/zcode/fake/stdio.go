package fake

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
)

// Process exit codes for the standalone binary hosting mode (D-01 mode B),
// named here so cmd/fake-app-server and the reexec helper tests agree on the
// contract by construction.
const (
	// ExitServeFailed reports a serve run that recorded an error — a mid-run
	// broken pipe or a failed scenario assertion. A clean EOF-drain exit is
	// exit 0 (dialect §1 EOF facts).
	ExitServeFailed = 1

	// ExitUsage reports the fail-loud usage contract: --scenario missing or
	// the scenario file failing to load — one clear stderr line naming the
	// problem, exit 2 (the scrub-capture IN-04 precedent: invalid input
	// exits non-zero with a message naming the contract).
	ExitUsage = 2
)

// Main is the standalone-binary hosting mode of the scripting core (D-01
// mode B): load the scenario at scenarioPath and serve it over the process's
// own stdio. stdin is the protocol's inbound stream and stdout carries
// protocol frames only — never diagnostics (stdout discipline T-3-01); every
// log line goes to stderr via slog. On stdin EOF the scenario's eof_drain_ms
// window applies and the run ends cleanly (dialect §1: the server drains
// ~100 ms, a request issued just before EOF is never answered, then exits).
// The return value is the process exit code: ExitUsage when the scenario
// fails to load, ExitServeFailed when the serve run records an error, 0 on
// the clean EOF-drain exit. Both cmd/fake-app-server and the reexec helper
// tests call this entry — one scripting core, two hosting modes.
func Main(scenarioPath string) int {
	return MainTo(scenarioPath, "")
}

// MainTo is Main with an optional transcript sink (plan 04-05): when
// transcriptPath is non-empty, the served transcript is written there after
// the clean EOF-drain exit — golden format, one dir\tline record per frame —
// so the exec tier can assert what the standalone binary ACTUALLY served
// (the binary E2E's setModel payload assertion rides it; the in-process
// Serve already hands the transcript back, but the binary hosting mode had
// no window into its own wire). A transcript write failure is a serve
// failure (ExitServeFailed): the run's evidence must never be silently lost.
func MainTo(scenarioPath, transcriptPath string) int {
	sc, err := LoadScenario(scenarioPath)
	if err != nil {
		slog.Error("fake-app-server: load scenario failed", "err", err)

		return ExitUsage
	}

	tr, err := Serve(newStdioRW(), sc)
	if err != nil {
		slog.Error("fake-app-server: serve failed", "err", err)

		return ExitServeFailed
	}

	if transcriptPath != "" {
		if werr := writeTranscriptFile(transcriptPath, tr); werr != nil {
			slog.Error("fake-app-server: write transcript failed", "path", transcriptPath, "err", werr)

			return ExitServeFailed
		}
	}

	slog.Info("fake-app-server: scenario served to EOF", "name", sc.Name, "frames", len(tr.Lines))

	return 0
}

// writeTranscriptFile writes the transcript in the golden fixture format —
// one "dir\tjson" record per line — so every tier reads the same shape.
func writeTranscriptFile(path string, tr Transcript) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("fake: create transcript file: %w", err)
	}

	w := bufio.NewWriter(f)

	for _, line := range tr.Lines {
		if _, werr := fmt.Fprintf(w, "%s\t%s\n", line.Dir, line.JSON); werr != nil {
			_ = f.Close()

			return fmt.Errorf("fake: write transcript record: %w", werr)
		}
	}

	if err := w.Flush(); err != nil {
		_ = f.Close()

		return fmt.Errorf("fake: flush transcript file: %w", err)
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("fake: close transcript file: %w", err)
	}

	return nil
}

// stdioRW adapts the raw stdio pair to the io.ReadWriteCloser Serve expects:
// reads come from os.Stdin and writes go to os.Stdout — os.Stdin/os.Stdout
// are the protocol. Close closes stdin only (ending the protocol stream) and
// never touches os.Stdout, whose ordering belongs to the process exit path.
type stdioRW struct {
	in  *os.File
	out *os.File
}

// newStdioRW wires the adapter over the process's own stdin/stdout pair.
func newStdioRW() *stdioRW {
	return &stdioRW{in: os.Stdin, out: os.Stdout}
}

// Read implements io.Reader over os.Stdin (the protocol's inbound stream).
func (s *stdioRW) Read(b []byte) (int, error) {
	n, err := s.in.Read(b)
	if err != nil {
		return n, fmt.Errorf("fake: read stdio stream: %w", err)
	}

	return n, nil
}

// Write implements io.Writer to os.Stdout: protocol frames only.
func (s *stdioRW) Write(b []byte) (int, error) {
	n, err := s.out.Write(b)
	if err != nil {
		return n, fmt.Errorf("fake: write stdio stream: %w", err)
	}

	return n, nil
}

// Close ends the protocol stream by closing stdin; stdout stays open —
// closing it would race the serve loop's final writes (the EOF-drain
// shutdown writes honest unanswered notes up to the last moment).
func (s *stdioRW) Close() error {
	if err := s.in.Close(); err != nil {
		return fmt.Errorf("fake: close stdio stream: %w", err)
	}

	return nil
}
