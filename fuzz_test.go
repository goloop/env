package env

import (
	"strings"
	"testing"
)

// FuzzSplitN checks that splitN never panics on arbitrary input and that for
// a simple (group-free) string joining the parts back yields the original.
func FuzzSplitN(f *testing.F) {
	f.Add("a,b,c", ",")
	f.Add("ключ,значення", ",")
	f.Add("a,(b,c),d", ",")
	f.Add("café,naïve", ",")
	f.Add("один::два", "::")

	f.Fuzz(func(t *testing.T, s, sep string) {
		if sep == "" {
			t.Skip()
		}
		parts := splitN(s, sep, -1)

		// For a string without grouping characters, the round-trip holds.
		if !strings.ContainsAny(s, "\"'`([{}])") {
			if got := strings.Join(parts, sep); got != s && s != "" {
				t.Errorf("round-trip: splitN(%q, %q) joined = %q", s, sep, got)
			}
		}
	})
}

// FuzzParseExpression checks that parseExpression never panics; malformed
// input must return an error, not crash.
func FuzzParseExpression(f *testing.F) {
	f.Add("KEY=value")
	f.Add(`KEY="a # b"`)
	f.Add("export K='it\\'s'")
	f.Add("=")
	f.Add(`A="`)

	f.Fuzz(func(t *testing.T, line string) {
		_, _, _, _ = parseExpression(line)
	})
}

// FuzzParse checks that Parse never panics on arbitrary .env data.
func FuzzParse(f *testing.F) {
	f.Add("KEY=value\n")
	f.Add("export A=\"x\"\n# comment\nB='y'\n")
	f.Add("MULTI=\"line1\nline2\"\n")
	f.Add("EMPTY=\nSPACED= value\n")

	f.Fuzz(func(t *testing.T, data string) {
		_, _ = Parse(strings.NewReader(data))
	})
}
