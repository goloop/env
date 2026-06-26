package env_test

import (
	"strings"
	"testing"

	"github.com/goloop/env/v2"
)

// TestInlineComment checks that an unquoted '#' starts a comment only when it
// is preceded by whitespace, so values containing '#' are preserved.
func TestInlineComment(t *testing.T) {
	cases := map[string]string{
		"PASSWORD=abc#123":            "abc#123",     // no space: literal
		"TITLE=hello world # comment": "hello world", // space before #: comment
		"FRAG=http://x/#section":      "http://x/#section",
		"COLOR=#fff":                  "#fff", // hex colour, leading #
		"NAME=a # b # c":              "a",    // first whitespace #
	}

	for line, want := range cases {
		m, err := env.Parse(strings.NewReader(line + "\n"))
		if err != nil {
			t.Fatalf("%q: %v", line, err)
		}
		key := strings.SplitN(line, "=", 2)[0]
		if m[key] != want {
			t.Errorf("%q: got %q, want %q", line, m[key], want)
		}
	}
}

// TestBoolSynonyms checks the accepted boolean spellings (strconv literals plus
// yes/no/on/off, case-insensitive) and that anything else is an error.
func TestBoolSynonyms(t *testing.T) {
	type cfg struct {
		B bool `env:"B"`
	}

	truthy := []string{"true", "1", "t", "yes", "on", "YES", "On"}
	falsy := []string{"false", "0", "f", "no", "off", "NO", "Off"}
	invalid := []string{"ok", "0xff", "true/false", "2"}

	for _, v := range truthy {
		var c cfg
		if err := env.UnmarshalMap(map[string]string{"B": v}, &c); err != nil || !c.B {
			t.Errorf("%q: got %v err=%v, want true", v, c.B, err)
		}
	}
	for _, v := range falsy {
		var c cfg
		if err := env.UnmarshalMap(map[string]string{"B": v}, &c); err != nil || c.B {
			t.Errorf("%q: got %v err=%v, want false", v, c.B, err)
		}
	}
	for _, v := range invalid {
		var c cfg
		if err := env.UnmarshalMap(map[string]string{"B": v}, &c); err == nil {
			t.Errorf("%q: expected an error", v)
		}
	}
}

// TestDefaultSeparatorComma checks that the default list separator is a comma,
// so spaces inside elements are preserved.
func TestDefaultSeparatorComma(t *testing.T) {
	type cfg struct {
		A []string `env:"A"` // no sep tag -> default
	}

	var c cfg
	if err := env.UnmarshalMap(map[string]string{"A": "a b,c"}, &c); err != nil {
		t.Fatal(err)
	}
	if len(c.A) != 2 || c.A[0] != "a b" || c.A[1] != "c" {
		t.Errorf("got %v, want [\"a b\" \"c\"]", c.A)
	}
}

// TestErrorContext checks that a conversion error names the offending key.
func TestErrorContext(t *testing.T) {
	type cfg struct {
		Port int `env:"PORT"`
	}

	var c cfg
	err := env.UnmarshalMap(map[string]string{"PORT": "abc"}, &c)
	if err == nil || !strings.HasPrefix(err.Error(), "PORT:") {
		t.Errorf("expected an error prefixed with the key, got %v", err)
	}
}

// TestLargeValue checks that a value longer than bufio.Scanner's default token
// limit (~64 KiB) is parsed, e.g. a PEM chain, key or base64 blob.
func TestLargeValue(t *testing.T) {
	big := strings.Repeat("x", 70000)
	m, err := env.Parse(strings.NewReader("A=" + big + "\n"))
	if err != nil {
		t.Fatalf("Parse large value: %v", err)
	}
	if len(m["A"]) != len(big) {
		t.Errorf("got %d bytes, want %d", len(m["A"]), len(big))
	}
}
