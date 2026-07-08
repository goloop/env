package env_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goloop/env/v2"
)

// TestLoadReadDuplicateKeyAgree guards BUG-02: a key repeated within one file
// resolves to the last occurrence for both Load and Read.
func TestLoadReadDuplicateKeyAgree(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "dup.env")
	if err := os.WriteFile(file, []byte("DUP=first\nDUP=second\n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	m, err := env.Read(file)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if m["DUP"] != "second" {
		t.Errorf("Read DUP = %q, want %q", m["DUP"], "second")
	}

	os.Unsetenv("DUP")
	t.Cleanup(func() { os.Unsetenv("DUP") })
	if err := env.Load(file); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := os.Getenv("DUP"); got != "second" {
		t.Errorf("Load DUP = %q, want %q (must agree with Read)", got, "second")
	}
}

// TestQuoteInInlineCommentDoesNotEatLines guards BUG-01: a quote inside an
// inline comment on a closed, single-line value must not open a multiline
// value and swallow the following lines.
func TestQuoteInInlineCommentDoesNotEatLines(t *testing.T) {
	cases := []struct {
		name string
		body string
		key  string
		val  string
		next string
	}{
		{"single quote apostrophe", "KEY='abc' # don't worry\nPORT=8080\n", "KEY", "abc", "8080"},
		{"double quote in comment", "A=\"abc\" # say \"hello\nB=1\n", "A", "abc", "1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m, err := env.Parse(strings.NewReader(c.body))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if m[c.key] != c.val {
				t.Errorf("%s = %q, want %q", c.key, m[c.key], c.val)
			}
			nextKey := "PORT"
			if c.name == "double quote in comment" {
				nextKey = "B"
			}
			if m[nextKey] != c.next {
				t.Errorf("%s = %q, want %q (line swallowed?)", nextKey, m[nextKey], c.next)
			}
		})
	}
}

// TestSplitNestedSameBracket guards BUG-03: splitN must count nested brackets
// of the same kind so an inner close does not end the group early.
func TestSplitNestedSameBracket(t *testing.T) {
	var obj struct {
		Items []string `env:"ITEMS" sep:","`
	}
	err := env.UnmarshalMap(map[string]string{
		"ITEMS": `{"a":{"b":1},"c":2},x`,
	}, &obj)
	if err != nil {
		t.Fatalf("UnmarshalMap: %v", err)
	}
	want := []string{`{"a":{"b":1},"c":2}`, "x"}
	if len(obj.Items) != len(want) {
		t.Fatalf("Items = %#v, want %#v", obj.Items, want)
	}
	for i := range want {
		if obj.Items[i] != want[i] {
			t.Errorf("Items[%d] = %q, want %q", i, obj.Items[i], want[i])
		}
	}
}

// TestTextAfterClosingQuoteIsError guards BUG-06: unexpected text after a
// closing quote is a parse error rather than being silently dropped.
func TestTextAfterClosingQuoteIsError(t *testing.T) {
	if _, err := env.Parse(strings.NewReader(`J="abc"def` + "\n")); err == nil {
		t.Error("Parse of J=\"abc\"def returned nil error, want error")
	}
	// A trailing inline comment after the closing quote stays valid.
	m, err := env.Parse(strings.NewReader(`J="abc" # ok` + "\n"))
	if err != nil {
		t.Fatalf("Parse with trailing comment: %v", err)
	}
	if m["J"] != "abc" {
		t.Errorf("J = %q, want %q", m["J"], "abc")
	}
}

// TestEmptyNestedStructDecodes guards BUG-04: an empty nested struct must not
// break Unmarshal (Marshal already tolerates it).
func TestEmptyNestedStructDecodes(t *testing.T) {
	var obj struct {
		Name string `env:"NAME"`
		Meta struct{}
		Ptr  *struct{}
	}
	if err := env.UnmarshalMap(map[string]string{"NAME": "x"}, &obj); err != nil {
		t.Fatalf("UnmarshalMap: %v", err)
	}
	if obj.Name != "x" {
		t.Errorf("Name = %q, want %q", obj.Name, "x")
	}
}

// TestParserOnPointerType guards BUG-05: a parser registered for a pointer
// type must be honoured for a pointer field instead of losing the value.
// (money is declared in customparser_test.go.)
func TestParserOnPointerType(t *testing.T) {
	var obj struct {
		M *money `env:"M"`
	}
	err := env.UnmarshalMap(map[string]string{"M": "500"}, &obj,
		env.WithParser(func(s string) (*money, error) {
			return &money{cents: len(s)}, nil
		}),
	)
	if err != nil {
		t.Fatalf("UnmarshalMap: %v", err)
	}
	if obj.M == nil {
		t.Fatal("M is nil, parser for *money was ignored")
	}
	if obj.M.cents != 3 {
		t.Errorf("M.cents = %d, want 3", obj.M.cents)
	}
}
