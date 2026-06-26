package env_test

import (
	"reflect"
	"testing"

	"github.com/goloop/env/v2"
)

// TestSliceReplace checks that decoding replaces a slice (like encoding/json)
// instead of appending, so Unmarshal is idempotent and overrides defaults.
func TestSliceReplace(t *testing.T) {
	type cfg struct {
		A []int `env:"A" sep:","`
	}

	// Repeated decode is idempotent.
	var s cfg
	env.UnmarshalMap(map[string]string{"A": "1,2,3"}, &s)
	env.UnmarshalMap(map[string]string{"A": "1,2,3"}, &s)
	if !reflect.DeepEqual(s.A, []int{1, 2, 3}) {
		t.Errorf("repeated decode: got %v, want [1 2 3]", s.A)
	}

	// An in-code default is replaced, not appended to.
	d := cfg{A: []int{9, 9}}
	env.UnmarshalMap(map[string]string{"A": "1,2"}, &d)
	if !reflect.DeepEqual(d.A, []int{1, 2}) {
		t.Errorf("default override: got %v, want [1 2]", d.A)
	}
}

// TestPointerSliceDecode checks that []*scalar decodes (a nil element for an
// empty value) and round-trips.
func TestPointerSliceDecode(t *testing.T) {
	type cfg struct {
		A []*int `env:"A" sep:","`
	}

	var s cfg
	if err := env.UnmarshalMap(map[string]string{"A": "1,,3"}, &s); err != nil {
		t.Fatal(err)
	}
	if len(s.A) != 3 || *s.A[0] != 1 || s.A[1] != nil || *s.A[2] != 3 {
		t.Fatalf("decode []*int: got %v", s.A)
	}

	// Round-trip: marshal -> "1,,3" -> decode back to the same shape.
	m, err := env.MarshalMap(s)
	if err != nil {
		t.Fatal(err)
	}
	if m["A"] != "1,,3" {
		t.Errorf("marshal []*int: got %q, want %q", m["A"], "1,,3")
	}
	var back cfg
	env.UnmarshalMap(m, &back)
	if len(back.A) != 3 || *back.A[0] != 1 || back.A[1] != nil || *back.A[2] != 3 {
		t.Errorf("round-trip []*int: got %v", back.A)
	}
}

// TestPresenceSemantics checks the encoding/json-style presence rules: an
// absent key leaves the field untouched (preserving in-code defaults), while a
// present but empty value sets the zero value.
func TestPresenceSemantics(t *testing.T) {
	type cfg struct {
		Port int    `env:"PORT"`
		Arr  [3]int `env:"ARR" sep:","`
		Sl   []int  `env:"SL" sep:","`
	}

	// Absent keys: every field keeps its in-code default.
	def := cfg{Port: 8080, Arr: [3]int{9, 9, 9}, Sl: []int{7}}
	if err := env.UnmarshalMap(map[string]string{}, &def); err != nil {
		t.Fatal(err)
	}
	if def.Port != 8080 || def.Arr != [3]int{9, 9, 9} || !reflect.DeepEqual(def.Sl, []int{7}) {
		t.Errorf("absent keys must leave fields untouched, got %+v", def)
	}

	// Present but empty: fields are zeroed/cleared.
	empty := cfg{Port: 8080, Arr: [3]int{9, 9, 9}, Sl: []int{7}}
	err := env.UnmarshalMap(map[string]string{"PORT": "", "ARR": "", "SL": ""}, &empty)
	if err != nil {
		t.Fatal(err)
	}
	if empty.Port != 0 || empty.Arr != [3]int{0, 0, 0} || len(empty.Sl) != 0 {
		t.Errorf("present-empty must clear fields, got %+v", empty)
	}
}

// TestNestedStructDefaultsPreserved checks that decoding a nested struct in
// place keeps sub-fields absent from the source at their existing values.
func TestNestedStructDefaultsPreserved(t *testing.T) {
	type Inner struct {
		A int `env:"A"`
		B int `env:"B"`
	}
	type cfg struct {
		In Inner `env:"IN"`
	}

	c := cfg{In: Inner{A: 1, B: 2}}
	if err := env.UnmarshalMap(map[string]string{"IN_A": "10"}, &c); err != nil {
		t.Fatal(err)
	}
	if c.In.A != 10 || c.In.B != 2 {
		t.Errorf("nested defaults: got %+v, want {A:10 B:2}", c.In)
	}
}

// TestFieldCacheRespectsOptions checks that the per-type field cache does not
// leak call-level settings: the same struct type decoded with different
// prefixes and separators must behave correctly each time.
func TestFieldCacheRespectsOptions(t *testing.T) {
	type cfg struct {
		Items []string `env:"ITEMS"`
		Port  int      `env:"PORT"`
	}

	var a cfg
	err := env.UnmarshalMap(
		map[string]string{"A_ITEMS": "x:y", "A_PORT": "1"},
		&a, env.WithPrefix("A"), env.WithSeparator(":"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Items) != 2 || a.Items[0] != "x" || a.Port != 1 {
		t.Errorf("call A (prefix A, sep ':'): got %+v", a)
	}

	var b cfg
	err = env.UnmarshalMap(
		map[string]string{"B_ITEMS": "p,q,r", "B_PORT": "2"},
		&b, env.WithPrefix("B"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Items) != 3 || b.Items[0] != "p" || b.Port != 2 {
		t.Errorf("call B (prefix B, default sep): got %+v", b)
	}
}
