// Command fake-app-server serves a scripted ZCode app-server scenario over
// stdio for exec-plumbing tests (D-01 mode B); the real desktop bundle is
// never spawned in CI. --transcript optionally writes the served transcript
// to a file after the clean EOF-drain exit (plan 04-05): the binary E2E's
// window into what the standalone binary actually served.
package main

import (
	"flag"
	"log/slog"
	"os"

	"github.com/Djarvur/zcode-acp/internal/zcode/fake"
)

func main() {
	scenarioPath := flag.String("scenario", "", "path to the scenario JSON to serve over stdio (required)")
	transcriptPath := flag.String("transcript", "",
		"optional path to write the served transcript to after the run (golden dir\\tline format)")
	flag.Parse()

	if *scenarioPath == "" {
		slog.Error("fake-app-server: --scenario is required: pass the path to a scenario JSON file")

		os.Exit(fake.ExitUsage)
	}

	os.Exit(fake.MainTo(*scenarioPath, *transcriptPath))
}
