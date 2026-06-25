package env

import (
	"os"
	"strings"
	"testing"
)

// TestParse checks that Parse reads a reader into a map, expanding double
// quoted/unquoted values while leaving single quotes literal.
func TestParse(t *testing.T) {
	os.Clearenv()
	Set("BASE", "world")

	src := "A=hello\nB=\"x ${BASE}\"\nC='$BASE'\n"
	m, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]string{"A": "hello", "B": "x world", "C": "$BASE"}
	for k, v := range want {
		if m[k] != v {
			t.Errorf("Parse[%s] = %q, want %q", k, m[k], v)
		}
	}
}

// TestParseRaw checks that ParseRaw does not expand variables.
func TestParseRaw(t *testing.T) {
	os.Clearenv()
	Set("BASE", "world")

	m, err := ParseRaw(strings.NewReader("B=$BASE\n"))
	if err != nil {
		t.Fatal(err)
	}
	if m["B"] != "$BASE" {
		t.Errorf("ParseRaw[B] = %q, want %q", m["B"], "$BASE")
	}
}

// TestRead checks that Read returns the file as a map without touching the
// process environment.
func TestRead(t *testing.T) {
	os.Clearenv()
	m, err := Read("./fixtures/simple.env")
	if err != nil {
		t.Fatal(err)
	}
	if m["KEY_0"] != "value 0" || m["KEY_1"] != "value_1" {
		t.Errorf("unexpected map: %v", m)
	}
	if _, ok := os.LookupEnv("KEY_0"); ok {
		t.Error("Read must not write to the environment")
	}
}

// TestLoadReader checks that LoadReader loads into the environment and does
// not override existing keys.
func TestLoadReader(t *testing.T) {
	os.Clearenv()
	if err := LoadReader(strings.NewReader("X=1\nY=2\n")); err != nil {
		t.Fatal(err)
	}
	if Get("X") != "1" || Get("Y") != "2" {
		t.Errorf("X=%q Y=%q", Get("X"), Get("Y"))
	}

	Set("X", "keep")
	if err := LoadReader(strings.NewReader("X=changed\n")); err != nil {
		t.Fatal(err)
	}
	if Get("X") != "keep" {
		t.Errorf("LoadReader must not override existing keys, X=%q", Get("X"))
	}
}

// TestUnmarshalFile checks that UnmarshalFile decodes a file into a struct
// without touching the process environment.
func TestUnmarshalFile(t *testing.T) {
	type cfg struct {
		Key0 string `env:"KEY_0"`
		Key1 string `env:"KEY_1"`
	}

	os.Clearenv()
	var c cfg
	if err := UnmarshalFile("./fixtures/simple.env", &c); err != nil {
		t.Fatal(err)
	}
	if c.Key0 != "value 0" || c.Key1 != "value_1" {
		t.Errorf("unexpected struct: %+v", c)
	}
	if _, ok := os.LookupEnv("KEY_0"); ok {
		t.Error("UnmarshalFile must not write to the environment")
	}
}
