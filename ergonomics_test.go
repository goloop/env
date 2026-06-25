package env_test

import (
	"os"
	"testing"

	"github.com/goloop/env"
)

// TestMustLoad checks that MustLoad loads a valid file without panicking.
func TestMustLoad(t *testing.T) {
	env.Clear()
	env.MustLoad("./fixtures/simple.env")
	if env.Get("KEY_1") != "value_1" {
		t.Errorf("KEY_1 = %q, want value_1", env.Get("KEY_1"))
	}
}

// TestMustLoadPanics checks that MustLoad panics on a missing file.
func TestMustLoadPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected a panic for a missing file")
		}
	}()
	env.MustLoad("./fixtures/does-not-exist.env")
}

// TestWithFileMode checks that MarshalFile honours the requested permissions.
func TestWithFileMode(t *testing.T) {
	type cfg struct {
		Token string `env:"TOKEN"`
	}
	path := t.TempDir() + "/secret.env"

	if err := env.MarshalFile(path, cfg{Token: "s3cr3t"}, env.WithFileMode(0o600)); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("mode = %o, want 600", info.Mode().Perm())
	}
}

// TestAll checks the iterator: it yields the parsed pairs, does not touch the
// environment, and supports early break.
func TestAll(t *testing.T) {
	env.Clear()

	got := map[string]string{}
	for key, value := range env.All("./fixtures/simple.env") {
		got[key] = value
	}
	if got["KEY_0"] != "value 0" || got["KEY_1"] != "value_1" {
		t.Errorf("All = %v", got)
	}
	if _, ok := os.LookupEnv("KEY_0"); ok {
		t.Error("All must not write to the environment")
	}

	count := 0
	for range env.All("./fixtures/simple.env") {
		count++
		break
	}
	if count != 1 {
		t.Errorf("early break: visited %d, want 1", count)
	}
}
