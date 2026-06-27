package env_test

import (
	"bytes"
	"errors"
	"net/url"
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
