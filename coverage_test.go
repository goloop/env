package env_test

import (
	"bytes"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/goloop/env/v2"
)

// failRW fails on both Read and Write, to exercise reader/writer error paths.
type failRW struct{}

func (failRW) Read([]byte) (int, error)  { return 0, errors.New("read failed") }
func (failRW) Write([]byte) (int, error) { return 0, errors.New("write failed") }

// TestReadRaw checks ReadRaw parses a file without expansion.
func TestReadRaw(t *testing.T) {
	m, err := env.ReadRaw("./fixtures/simple.env")
	if err != nil {
		t.Fatal(err)
	}
	if m["KEY_1"] != "value_1" {
		t.Errorf("ReadRaw KEY_1 = %q, want value_1", m["KEY_1"])
	}
}

// TestNumericOverflow checks that out-of-range numbers are rejected per kind.
func TestNumericOverflow(t *testing.T) {
	cases := []struct {
		name string
		obj  any
		val  string
	}{
		{"int8", &struct {
			V int8 `env:"V"`
		}{}, "200"},
		{"int16", &struct {
			V int16 `env:"V"`
		}{}, "40000"},
		{"uint8", &struct {
			V uint8 `env:"V"`
		}{}, "300"},
		{"uint8-negative", &struct {
			V uint8 `env:"V"`
		}{}, "-1"},
		{"float32", &struct {
			V float32 `env:"V"`
		}{}, "1e40"},
	}
	for _, c := range cases {
		if err := env.UnmarshalMap(map[string]string{"V": c.val}, c.obj); err == nil {
			t.Errorf("%s=%s: expected an out-of-range error", c.name, c.val)
		}
	}
}

// TestErrorPaths exercises the reader/writer/file error branches.
func TestErrorPaths(t *testing.T) {
	type cfg struct {
		V string `env:"V"`
	}
	var c cfg

	if err := env.UnmarshalFile("/no/such/file.env", &c); err == nil {
		t.Error("UnmarshalFile missing file: expected error")
	}
	if err := env.UnmarshalReader(failRW{}, &c); err == nil {
		t.Error("UnmarshalReader failing reader: expected error")
	}
	if err := env.MarshalFile("/no/such/dir/x.env", cfg{V: "x"}); err == nil {
		t.Error("MarshalFile bad path: expected error")
	}
	if err := env.MarshalWriter(failRW{}, cfg{V: "x"}); err == nil {
		t.Error("MarshalWriter failing writer: expected error")
	}
	if err := env.LoadReader(failRW{}); err == nil {
		t.Error("LoadReader failing reader: expected error")
	}
}

// TestTimeLayoutNames checks that the standard layout names resolve and parse.
func TestTimeLayoutNames(t *testing.T) {
	type cfg struct {
		T time.Time `env:"T"`
	}
	now := time.Date(2026, 6, 27, 15, 4, 5, 0, time.UTC)
	names := map[string]string{
		"DateOnly":    time.DateOnly,
		"DateTime":    time.DateTime,
		"TimeOnly":    time.TimeOnly,
		"RFC1123":     time.RFC1123,
		"RFC1123Z":    time.RFC1123Z,
		"RFC822":      time.RFC822,
		"RFC822Z":     time.RFC822Z,
		"RFC850":      time.RFC850,
		"RFC3339Nano": time.RFC3339Nano,
		"ANSIC":       time.ANSIC,
		"UnixDate":    time.UnixDate,
		"Kitchen":     time.Kitchen,
		"Stamp":       time.Stamp,
	}
	for name, layout := range names {
		s := now.Format(layout)
		var c cfg
		err := env.UnmarshalMap(map[string]string{"T": s}, &c, env.WithTimeLayout(name))
		if err != nil {
			t.Errorf("layout %s (%q): %v", name, s, err)
		}
	}
}

// TestAllMissingFile checks the iterator yields nothing (and does not panic)
// for a missing file.
func TestAllMissingFile(t *testing.T) {
	n := 0
	for range env.All("/no/such/file.env") {
		n++
	}
	if n != 0 {
		t.Errorf("All(missing) yielded %d pairs, want 0", n)
	}
}

// TestMarshalUnsupportedType checks the encode-error path of Marshal* for a
// field type that cannot be serialized.
func TestMarshalUnsupportedType(t *testing.T) {
	type bad struct {
		C chan int `env:"C"`
	}
	b := bad{C: make(chan int)}

	if _, err := env.MarshalMap(b); err == nil {
		t.Error("MarshalMap chan: expected error")
	}
	if err := env.MarshalFile(t.TempDir()+"/x.env", b); err == nil {
		t.Error("MarshalFile chan: expected error")
	}
	var buf bytes.Buffer
	if err := env.MarshalWriter(&buf, b); err == nil {
		t.Error("MarshalWriter chan: expected error")
	}
}

// TestSetValueTypes exercises the time.Duration, *time.Time and url.URL
// branches of setValue, including their error paths.
func TestSetValueTypes(t *testing.T) {
	var d struct {
		D time.Duration `env:"D"`
	}
	if err := env.UnmarshalMap(map[string]string{"D": "30s"}, &d); err != nil || d.D != 30*time.Second {
		t.Errorf("duration: got %v err=%v", d.D, err)
	}
	if err := env.UnmarshalMap(map[string]string{"D": "nope"}, &d); err == nil {
		t.Error("invalid duration: expected error")
	}

	var pt struct {
		T *time.Time `env:"T" layout:"2006-01-02"`
	}
	if err := env.UnmarshalMap(map[string]string{"T": "2026-06-27"}, &pt); err != nil || pt.T == nil {
		t.Errorf("*time.Time: got %v err=%v", pt.T, err)
	}

	var u struct {
		U url.URL `env:"U"`
	}
	if err := env.UnmarshalMap(map[string]string{"U": "://bad"}, &u); err == nil {
		t.Error("invalid url: expected error")
	}
}

// TestReadSeq checks the error-aware iterator over a file's pairs.
func TestReadSeq(t *testing.T) {
	seq, err := env.ReadSeq("./fixtures/simple.env")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for k, v := range seq {
		got[k] = v
	}
	if got["KEY_1"] != "value_1" {
		t.Errorf("ReadSeq KEY_1 = %q, want value_1", got["KEY_1"])
	}

	if _, err := env.ReadSeq("/no/such/file.env"); err == nil {
		t.Error("ReadSeq missing file: expected error")
	}
}

// TestWithRequiredAll checks that every leaf field becomes required, that def
// satisfies it, and that nested structs are excluded while their sub-fields are
// required.
func TestWithRequiredAll(t *testing.T) {
	type cfg struct {
		Host string `env:"HOST"`
		Port int    `env:"PORT"`
	}
	var a cfg
	if err := env.UnmarshalMap(map[string]string{"HOST": "x"}, &a, env.WithRequiredAll()); err == nil {
		t.Error("missing field: expected error")
	}
	var b cfg
	if err := env.UnmarshalMap(map[string]string{"HOST": "x", "PORT": "1"}, &b, env.WithRequiredAll()); err != nil {
		t.Errorf("all present: %v", err)
	}

	type cfgDef struct {
		Host string `env:"HOST"`
		Port int    `env:"PORT" def:"8080"`
	}
	var d cfgDef
	if err := env.UnmarshalMap(map[string]string{"HOST": "x"}, &d, env.WithRequiredAll()); err != nil {
		t.Errorf("def should satisfy required: %v", err)
	}

	type inner struct {
		A int `env:"A"`
	}
	type cfgNested struct {
		Host string `env:"HOST"`
		In   inner  `env:"IN"`
	}
	var n cfgNested
	if err := env.UnmarshalMap(map[string]string{"HOST": "x"}, &n, env.WithRequiredAll()); err == nil {
		t.Error("nested sub-field missing: expected error")
	}
	var n2 cfgNested
	if err := env.UnmarshalMap(map[string]string{"HOST": "x", "IN_A": "5"}, &n2, env.WithRequiredAll()); err != nil {
		t.Errorf("nested all present: %v", err)
	}
}

// TestKeyParsing exercises the manual key parser: an export prefix with leading
// whitespace, "export" as a literal key, and a line without '='.
func TestKeyParsing(t *testing.T) {
	type cfg struct {
		Foo string `env:"FOO"`
	}
	var a cfg
	if err := env.UnmarshalString("  export FOO=bar\n", &a); err != nil || a.Foo != "bar" {
		t.Errorf("export+whitespace: got %q err=%v", a.Foo, err)
	}

	m, err := env.Parse(strings.NewReader("export=x\n"))
	if err != nil || m["export"] != "x" {
		t.Errorf(`export= literal key: got %v err=%v`, m, err)
	}

	if _, err := env.Parse(strings.NewReader("NOEQUALS\n")); err == nil {
		t.Error("line without '=': expected error")
	}
}

// shortWriter writes at most 3 bytes and reports no error, to trigger the
// short-write path.
type shortWriter struct{}

func (shortWriter) Write(p []byte) (int, error) {
	if len(p) > 3 {
		return 3, nil
	}
	return len(p), nil
}

// TestMarshalWriterShortWrite checks that a short write surfaces an error
// instead of being silently truncated (BUG-01).
func TestMarshalWriterShortWrite(t *testing.T) {
	type cfg struct {
		Host string `env:"HOST"`
		Port int    `env:"PORT"`
	}
	if err := env.MarshalWriter(shortWriter{}, cfg{Host: "localhost", Port: 8080}); err == nil {
		t.Error("short write: expected an error, got nil")
	}
}

// TestEmptyElementRoundTrip checks that a single empty element is distinct from
// an empty slice after a round-trip (BUG-04).
func TestEmptyElementRoundTrip(t *testing.T) {
	type cfg struct {
		V []string `env:"V" sep:","`
	}
	for _, want := range [][]string{{}, {""}, {"", ""}, {"a", ""}} {
		m, err := env.MarshalMap(cfg{V: want})
		if err != nil {
			t.Fatal(err)
		}
		var back cfg
		if err := env.UnmarshalMap(m, &back); err != nil {
			t.Fatal(err)
		}
		if len(back.V) != len(want) {
			t.Errorf("%#v -> wire %q -> %#v (len mismatch)", want, m["V"], back.V)
		}
	}
}

// TestRawCodec checks that the Raw variants round-trip values verbatim,
// including the $+quote+backtick combination that the non-raw path cannot
// represent (BUG-02).
func TestRawCodec(t *testing.T) {
	t.Setenv("HOME", "/home/real")
	type cfg struct {
		A string `env:"A"`
		B string `env:"B"`
		C string `env:"C"`
	}
	in := cfg{A: "it's `$5`", B: "$HOME/x", C: "a$b'c`d"}

	// String.
	s, err := env.MarshalStringRaw(in)
	if err != nil {
		t.Fatal(err)
	}
	var sOut cfg
	if err := env.UnmarshalStringRaw(s, &sOut); err != nil {
		t.Fatal(err)
	}
	if sOut != in {
		t.Errorf("string raw: got %+v, want %+v (wire %q)", sOut, in, s)
	}

	// File.
	path := t.TempDir() + "/raw.env"
	if err := env.MarshalFileRaw(path, in); err != nil {
		t.Fatal(err)
	}
	var fOut cfg
	if err := env.UnmarshalFileRaw(path, &fOut); err != nil {
		t.Fatal(err)
	}
	if fOut != in {
		t.Errorf("file raw: got %+v, want %+v", fOut, in)
	}

	// Writer/Reader.
	var buf bytes.Buffer
	if err := env.MarshalWriterRaw(&buf, in); err != nil {
		t.Fatal(err)
	}
	var wOut cfg
	if err := env.UnmarshalReaderRaw(&buf, &wOut); err != nil {
		t.Fatal(err)
	}
	if wOut != in {
		t.Errorf("writer/reader raw: got %+v, want %+v", wOut, in)
	}
}
