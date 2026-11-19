// Command zcode-acp exposes ZCode Desktop as a standard ACP (Agent Client
// Protocol) v1 server over stdio. This is the Phase-2 module skeleton: it
// compiles and answers --version; the real CLI and agent implementation
// arrive with the later phases.
package main

import (
	"flag"
	"fmt"
	"os"
)

// version is reported by --version until the release flow stamps a real
// value (Phase 7).
var version = "0.0.0-dev"

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		_, _ = fmt.Fprintln(os.Stdout, "zcode-acp "+version)

		return
	}
}
