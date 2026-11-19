package zcode

import (
	"context"
	"encoding/json"
	"errors"
)

// errSkeleton marks codec stages that land with the GREEN commits (plan
// Tasks 2 and 3). The tracer RED must fail on this exact error — a planned
// assertion failure, never a build failure.
var errSkeleton = errors.New("skeleton: not implemented")

// Call sends method with params on the ordered queue lane and waits for the
// matching reply, correlated by type-tagged canonical id (dialect §1: ids
// are string-or-int and replies echo the request id verbatim). A wire error
// reply surfaces as a *CallError; ctx cancellation deregisters the pending
// entry.
//
// Skeleton stage: the pending-call table lands with the codec leaf commit
// (plan Task 2); until then every call reports errSkeleton.
func (*Conn) Call(_ context.Context, _ string, _ any) (json.RawMessage, error) {
	return nil, errSkeleton
}
