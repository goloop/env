package env

import (
	"testing"
	"time"
)

// TestArrayOverflow checks that decoding more elements than a fixed array can
// hold is an error.
func TestArrayOverflow(t *testing.T) {
	type cfg struct {
		Nums [2]int `env:"NUMS" sep:","`
	}

	var c cfg
	if err := UnmarshalMap(map[string]string{"NUMS": "1,2,3"}, &c); err == nil {
		t.Error("expected overflow error for [2]int with 3 elements")
	}
}

// TestUnmarshalUnicodeSlice checks that multi-byte slice values are decoded
// intact (a regression guard for the splitN rune handling).
func TestUnmarshalUnicodeSlice(t *testing.T) {
	type cfg struct {
		Names []string `env:"NAMES" sep:","`
	}

	var c cfg
	if err := UnmarshalMap(map[string]string{"NAMES": "Привіт,світ,🌍"}, &c); err != nil {
		t.Fatal(err)
	}

	want := []string{"Привіт", "світ", "🌍"}
	if len(c.Names) != len(want) {
		t.Fatalf("got %v, want %v", c.Names, want)
	}
	for i := range want {
		if c.Names[i] != want[i] {
			t.Errorf("Names[%d] = %q, want %q", i, c.Names[i], want[i])
		}
	}
}

// TestResolveLayoutConstants checks the named-constant and literal layouts.
func TestResolveLayoutConstants(t *testing.T) {
	cases := map[string]string{
		"":            time.RFC3339,
		"RFC3339":     time.RFC3339,
		"RFC3339Nano": time.RFC3339Nano,
		"RFC1123":     time.RFC1123,
		"RFC822":      time.RFC822,
		"ANSIC":       time.ANSIC,
		"UnixDate":    time.UnixDate,
		"Kitchen":     time.Kitchen,
		"Stamp":       time.Stamp,
		"DateTime":    time.DateTime,
		"DateOnly":    time.DateOnly,
		"TimeOnly":    time.TimeOnly,
		"2006-01-02":  "2006-01-02", // literal pass-through
	}

	for name, want := range cases {
		if got := resolveLayout(name); got != want {
			t.Errorf("resolveLayout(%q) = %q, want %q", name, got, want)
		}
	}
}
