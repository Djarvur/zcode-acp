//go:build windows

package acpfront

import "time"

// autoAuthBudget bounds the silent authenticate at session/new. Windows
// pays a cold-start cost the Linux budget never sees: the backend is an
// Electron process whose first session/create can take well over 7 s
// (Defender scanning included), so the probe create must be allowed the
// time to answer. 30 s covers the cold start while staying under the
// client's 60 s kill.
const autoAuthBudget = 30 * time.Second
