package env_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/goloop/env/v2"
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

// TestMarshalFileEscaping checks that MarshalFile quotes values that would
// otherwise produce an invalid .env (newlines, edge spaces, inline-# risk), so
// the file round-trips through UnmarshalFile.
func TestMarshalFileEscaping(t *testing.T) {
	type cfg struct {
		Multi string `env:"MULTI"`
		Pad   string `env:"PAD"`
		Hash  string `env:"HASH"`
		Quote string `env:"QUOTE"`
		Lead  string `env:"LEAD"`
		Tab   string `env:"TAB"`
	}
	in := cfg{
		Multi: "line1\nline2",
		Pad:   "  spaced  ",
		Hash:  "a # b",
		Quote: `say "hi"`,
		Lead:  `"quoted"`,
		Tab:   "\ttabbed\t",
	}

	path := t.TempDir() + "/escaped.env"
	if err := env.MarshalFile(path, in); err != nil {
		t.Fatal(err)
	}

	var back cfg
	if err := env.UnmarshalFile(path, &back); err != nil {
		t.Fatalf("round-trip read: %v", err)
	}
	if back != in {
		t.Errorf("round-trip: got %+v, want %+v", back, in)
	}
}

// prefixMarshaler is a custom Marshaler used to check WithPrefix handling.
type prefixMarshaler struct{ V int }

func (p prefixMarshaler) MarshalEnv() (map[string]string, error) {
	return map[string]string{"KEY": "5"}, nil
}

// TestCustomMarshalerPrefix checks that WithPrefix is applied to the keys of a
// custom Marshaler, the same as for reflective structs.
func TestCustomMarshalerPrefix(t *testing.T) {
	m, err := env.MarshalMap(prefixMarshaler{5}, env.WithPrefix("APP"))
	if err != nil {
		t.Fatal(err)
	}
	if m["APP_KEY"] != "5" {
		t.Errorf("custom Marshaler + WithPrefix: got %v, want APP_KEY=5", m)
	}
}

// TestMarshalWriterUnmarshalReader checks the io.Writer/io.Reader symmetry and
// that the written text round-trips (including a value needing quoting).
func TestMarshalWriterUnmarshalReader(t *testing.T) {
	type cfg struct {
		Host string `env:"HOST"`
		Port int    `env:"PORT"`
		Note string `env:"NOTE"`
	}
	in := cfg{Host: "localhost", Port: 8080, Note: "a # b"}

	var buf bytes.Buffer
	if err := env.MarshalWriter(&buf, in); err != nil {
		t.Fatal(err)
	}

	var out cfg
	if err := env.UnmarshalReader(&buf, &out); err != nil {
		t.Fatal(err)
	}
	if out != in {
		t.Errorf("round-trip: got %+v, want %+v", out, in)
	}
}

// TestMarshalFileLiteralDollar checks that a literal $ survives a file
// round-trip (UnmarshalFile expands ${VAR}/$VAR, so MarshalFile must write
// $-values non-expandably).
func TestMarshalFileLiteralDollar(t *testing.T) {
	t.Setenv("HOME", "/home/real")
	type cfg struct {
		Tmpl string `env:"TMPL"`
		Cost string `env:"COST"`
		Apos string `env:"APOS"`
	}
	in := cfg{Tmpl: "path=$HOME/app", Cost: "$5.00", Apos: "it's $5"}

	path := t.TempDir() + "/dollar.env"
	if err := env.MarshalFile(path, in); err != nil {
		t.Fatal(err)
	}
	var back cfg
	if err := env.UnmarshalFile(path, &back); err != nil {
		t.Fatal(err)
	}
	if back != in {
		t.Errorf("literal $ round-trip: got %+v, want %+v", back, in)
	}
}
