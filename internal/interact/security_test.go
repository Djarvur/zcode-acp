//go:build darwin

package interact_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Djarvur/zcode-acp/internal/interact"
)

const fixtureHomeEnv = "HOME=/fixture-home"

var errSecurityExit = errors.New("exit status 1")

// securityFixture writes a credentials.json and returns its path and the
// Keychain account identity.
func securityFixture(t *testing.T) (path, account string) {
	t.Helper()

	path = filepath.Join(t.TempDir(), "credentials.json")
	if err := os.WriteFile(path, []byte(`{"fixture":"not-a-credential"}`), 0o600); err != nil {
		t.Fatalf("write credentials: %v", err)
	}

	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatalf("resolve credentials: %v", err)
	}
	sum := sha256.Sum256([]byte(resolved))

	return path, hex.EncodeToString(sum[:])
}

func TestSecurityKeyLoaderLookupCommand(t *testing.T) {
	path, account := securityFixture(t)
	env := []string{fixtureHomeEnv}

	argv, gotEnv, err := interact.NewSecurityKeyLoader(interact.DefaultSecurityBin, path, env).LookupCommand()
	if err != nil {
		t.Fatalf("LookupCommand: %v", err)
	}

	want := []string{
		interact.DefaultSecurityBin, "find-generic-password",
		"-s", "ZCode Credential Encryption",
		"-a", account,
		"-w",
	}
	if strings.Join(argv, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("argv = %v, want %v", argv, want)
	}
	if strings.Join(gotEnv, "\x00") != strings.Join(env, "\x00") {
		t.Fatalf("env = %v, want %v", gotEnv, env)
	}
	if strings.Contains(strings.Join(argv, " "), "secret") {
		t.Fatalf("argv names key material: %v", argv)
	}
}

func TestSecurityKeyLoaderLoadKey(t *testing.T) {
	path, _ := securityFixture(t)
	key := bytes.Repeat([]byte{0x5a}, 32)
	encoded := base64.RawURLEncoding.EncodeToString(key)

	t.Run("decodes the keychain answer", func(t *testing.T) {
		var gotArgv []string

		loader := interact.NewSecurityKeyLoaderWithExec(interact.DefaultSecurityBin, path, nil,
			func(_ context.Context, argv, _ []string) ([]byte, error) {
				gotArgv = argv

				return []byte(encoded + "\n"), nil
			})

		got, err := loader.LoadKey(context.Background())
		if err != nil {
			t.Fatalf("LoadKey: %v", err)
		}
		if !bytes.Equal(got, key) {
			t.Fatalf("decoded key = %x, want %x", got, key)
		}
		if len(gotArgv) == 0 || gotArgv[0] != interact.DefaultSecurityBin {
			t.Fatalf("loader argv = %v", gotArgv)
		}
	})

	t.Run("rejects malformed key material", func(t *testing.T) {
		loader := interact.NewSecurityKeyLoaderWithExec(interact.DefaultSecurityBin, path, nil,
			func(context.Context, []string, []string) ([]byte, error) {
				return []byte("not-base64!"), nil
			})

		if _, err := loader.LoadKey(context.Background()); err == nil {
			t.Fatal("malformed key material accepted")
		}
	})

	t.Run("wraps lookup failure", func(t *testing.T) {
		loader := interact.NewSecurityKeyLoaderWithExec(interact.DefaultSecurityBin, path, nil,
			func(context.Context, []string, []string) ([]byte, error) {
				return nil, errSecurityExit
			})

		_, err := loader.LoadKey(context.Background())
		if err == nil || !strings.Contains(err.Error(), "Keychain lookup failed") {
			t.Fatalf("LoadKey error = %v, want a wrapped lookup failure", err)
		}
	})
}

//nolint:cyclop,funlen // each Keychain outcome is a separate acceptance branch
func TestKeychainProber(t *testing.T) {
	credPath, account := securityFixture(t)
	env := []string{fixtureHomeEnv}

	t.Run("success", func(t *testing.T) {
		probe := interact.NewKeychainProber(interact.DefaultSecurityBin, credPath, env,
			interact.WithKeyringProbeLookPath(securityLookPath),
			interact.WithKeyringProbeExec(func(context.Context, []string, []string) ([]byte, []byte, error) {
				return []byte("synthetic-key-material"), nil, nil
			}))

		if err := probe(context.Background()); err != nil {
			t.Fatalf("keychain probe success = %v", err)
		}
	})

	t.Run("missing security binary", func(t *testing.T) {
		executed := false
		probe := interact.NewKeychainProber(interact.DefaultSecurityBin, credPath, env,
			interact.WithKeyringProbeLookPath(func(string) (string, error) {
				return "", errSecurityExit
			}),
			interact.WithKeyringProbeExec(func(context.Context, []string, []string) ([]byte, []byte, error) {
				executed = true

				return nil, nil, nil
			}))

		err := probe(context.Background())
		if !errors.Is(err, interact.ErrKeychainBinaryMissing) {
			t.Fatalf("error = %v, want ErrKeychainBinaryMissing", err)
		}
		if !strings.Contains(err.Error(), "security") {
			t.Fatalf("error %q must name the security tool", err)
		}
		if executed {
			t.Fatal("missing binary must not spawn a subprocess")
		}
	})

	t.Run("missing keychain item", func(t *testing.T) {
		probe := interact.NewKeychainProber(interact.DefaultSecurityBin, credPath, env,
			interact.WithKeyringProbeLookPath(securityLookPath),
			interact.WithKeyringProbeExec(func(context.Context, []string, []string) ([]byte, []byte, error) {
				return nil, []byte("The specified item could not be found in the keychain."), errSecurityExit
			}))

		err := probe(context.Background())
		if !errors.Is(err, interact.ErrKeychainNoEntry) {
			t.Fatalf("error = %v, want ErrKeychainNoEntry", err)
		}
		if !strings.Contains(err.Error(), "Keychain") {
			t.Fatalf("error %q must name the Keychain", err)
		}
	})

	t.Run("locked keychain", func(t *testing.T) {
		probe := interact.NewKeychainProber(interact.DefaultSecurityBin, credPath, env,
			interact.WithKeyringProbeLookPath(securityLookPath),
			interact.WithKeyringProbeExec(func(context.Context, []string, []string) ([]byte, []byte, error) {
				return nil, []byte("security: User interaction is not allowed."), errSecurityExit
			}))

		err := probe(context.Background())
		if !errors.Is(err, interact.ErrKeychainLocked) {
			t.Fatalf("error = %v, want ErrKeychainLocked", err)
		}
		if !strings.Contains(err.Error(), "unlock the login keychain") {
			t.Fatalf("error %q must carry the unlock hint", err)
		}
		if strings.Contains(err.Error(), "User interaction") {
			t.Fatalf("error %q leaked stderr verbatim", err)
		}
	})

	t.Run("argv and bounded context", func(t *testing.T) {
		var (
			gotArgv     []string
			hasDeadline bool
		)

		probe := interact.NewKeychainProber(interact.DefaultSecurityBin, credPath, env,
			interact.WithKeyringProbeLookPath(securityLookPath),
			interact.WithKeyringProbeExec(func(ctx context.Context, argv, _ []string) ([]byte, []byte, error) {
				gotArgv = argv
				_, hasDeadline = ctx.Deadline()

				return []byte("synthetic-key-material"), nil, nil
			}))

		if err := probe(context.Background()); err != nil {
			t.Fatalf("probe: %v", err)
		}
		if !hasDeadline {
			t.Fatal("keychain lookup must run under a bounded context")
		}

		want := []string{
			interact.DefaultSecurityBin, "find-generic-password",
			"-s", "ZCode Credential Encryption",
			"-a", account,
			"-w",
		}
		if strings.Join(gotArgv, "\x00") != strings.Join(want, "\x00") {
			t.Fatalf("argv = %v, want %v", gotArgv, want)
		}
	})

	t.Run("unresolvable credentials path", func(t *testing.T) {
		executed := false
		probe := interact.NewKeychainProber(interact.DefaultSecurityBin,
			filepath.Join(t.TempDir(), "missing.json"), env,
			interact.WithKeyringProbeLookPath(securityLookPath),
			interact.WithKeyringProbeExec(func(context.Context, []string, []string) ([]byte, []byte, error) {
				executed = true

				return nil, nil, nil
			}))

		err := probe(context.Background())
		if !errors.Is(err, interact.ErrKeychainNoEntry) {
			t.Fatalf("error = %v, want ErrKeychainNoEntry", err)
		}
		if executed {
			t.Fatal("an unresolvable credentials path must not spawn a subprocess")
		}
	})

	t.Run("timeout", func(t *testing.T) {
		probe := interact.NewKeychainProber(interact.DefaultSecurityBin, credPath, env,
			interact.WithKeyringProbeLookPath(securityLookPath),
			interact.WithKeyringProbeTimeout(20*time.Millisecond),
			interact.WithKeyringProbeExec(func(ctx context.Context, _ []string, _ []string) ([]byte, []byte, error) {
				<-ctx.Done()

				return nil, nil, ctx.Err()
			}))

		err := probe(context.Background())
		if !errors.Is(err, interact.ErrKeychainLookupTimeout) {
			t.Fatalf("error = %v, want ErrKeychainLookupTimeout", err)
		}
	})
}

func securityLookPath(string) (string, error) {
	return interact.DefaultSecurityBin, nil
}
