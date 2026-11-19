package zcode_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/Djarvur/zcode-acp/internal/zcode"
)

// TestIDKeyTypeTagging pins the WR-02 correlation rule at the key level:
// "5" the string and 5 the int are different wire values (dialect §1), so
// their canonical pending-table keys must never collide.
func TestIDKeyTypeTagging(t *testing.T) {
	intID := zcode.ID{Int: 5, IsInt: true}
	strID := zcode.ID{Str: "5"}

	if intID.Key() == strID.Key() {
		t.Fatalf("canonical keys collide across id types: %q", intID.Key())
	}

	if intID.Key() != "i:5" {
		t.Fatalf("int id key = %q, want i:5", intID.Key())
	}

	if strID.Key() != "s:5" {
		t.Fatalf("string id key = %q, want s:5", strID.Key())
	}
}

// TestEncodeOmitsJSONRPC pins the structural framing-key defense (dialect
// §1: the child rejects a jsonrpc key with -32600 unrecognized_keys — the
// typed Envelope cannot emit one).
func TestEncodeOmitsJSONRPC(t *testing.T) {
	line, err := zcode.Encode(&zcode.Envelope{
		ID:     &zcode.ID{Int: 5, IsInt: true},
		Method: "session/create",
		Params: json.RawMessage(`{"workspace":{"workspacePath":"/w","workspaceKey":"/w"}}`),
	})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	if bytes.Contains(line, []byte("jsonrpc")) {
		t.Fatalf("JSON-RPC key leaked to wire: %s", line)
	}

	if line[len(line)-1] != '\n' {
		t.Fatalf("encoded line is not newline-terminated: %q", line)
	}
}

// TestDecodeEchoesIDTypes pins id-type fidelity on decode: replies echo the
// request id verbatim (dialect §1) — numeric client ids and string server
// ids must decode with their type recorded (live shapes from the captures).
func TestDecodeEchoesIDTypes(t *testing.T) {
	intEnv, err := zcode.Decode([]byte(`{"id":5,"result":{"accepted":true}}`))
	if err != nil {
		t.Fatalf("decode int id: %v", err)
	}

	if intEnv.ID == nil || !intEnv.ID.IsInt || intEnv.ID.Int != 5 {
		t.Fatalf("int id decoded wrong: %+v", intEnv.ID)
	}

	strEnv, err := zcode.Decode([]byte(`{"id":"server-1","method":"session/requestRuntimePreferences"}`))
	if err != nil {
		t.Fatalf("decode string id: %v", err)
	}

	if strEnv.ID == nil || strEnv.ID.IsInt || strEnv.ID.Str != "server-1" {
		t.Fatalf("string id decoded wrong: %+v", strEnv.ID)
	}

	if strEnv.Method != "session/requestRuntimePreferences" {
		t.Fatalf("method decoded wrong: %q", strEnv.Method)
	}
}

// TestDecodeRoundTrip pins byte-exact params preservation: params travel as
// json.RawMessage so goldens can replay wire bytes without normalization.
func TestDecodeRoundTrip(t *testing.T) {
	raw := []byte(`{"id":1,"method":"session/create","params":{"workspace":{"workspacePath":"/w"}}}`)

	env, err := zcode.Decode(raw)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	if string(env.Params) != `{"workspace":{"workspacePath":"/w"}}` {
		t.Fatalf("params not byte-exact: %s", env.Params)
	}
}
