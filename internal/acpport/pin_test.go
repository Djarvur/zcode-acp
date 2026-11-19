package acpport_test

import (
	_ "github.com/coder/acp-go-sdk"
)

// The blank import keeps the D-01 SDK require tidy-stable before plan 04-02
// lands the adapter: go mod tidy retains requirements imported by any test
// in the module, so the pinned version survives the tidy-diff gate while no
// production code references the SDK yet (T-4-SC pin discipline — plan
// 04-01). The import stays inside the seam package: no non-test file outside
// internal/acpport may import coder/acp-go-sdk (TestSDKImportBoundary,
// plan 04-02, exempts _test.go files).
