package env

import (
	"os"
	"strings"
	"testing"
)

// TestExpandKeepsDollarText is the reason this package does not use the
// shell's substitution language. A .env file has no positional
// parameters, but it does have prices, one-liners and passwords, and
// under os.Expand every one of these lost part of itself without a word.
func TestExpandKeepsDollarText(t *testing.T) {
	src := strings.Join([]string{
		`PRICE=cost: $100`,
		`DISCOUNT="save $5 today"`,
		`AWK=$1 == "x"`,
		`SED=s/a/b/$2`,
		`PASSWORD=pa$$word`,
		`TRAILING=100$`,
		`ALONE=$`,
		`BRACED_DIGITS=${1}`,
	}, "\n")

	m, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct{ key, want string }{
		{"PRICE", "cost: $100"},
		{"DISCOUNT", "save $5 today"},
		{"AWK", `$1 == "x"`},
		{"SED", "s/a/b/$2"},
		{"PASSWORD", "pa$$word"},
		{"TRAILING", "100$"},
		{"ALONE", "$"},
		{"BRACED_DIGITS", "${1}"},
	} {
		if got := m[tc.key]; got != tc.want {
			t.Errorf("%s = %q, want %q", tc.key, got, tc.want)
		}
	}
}

// TestExpandStillExpandsNames pins the behaviour that must not change:
// a reference that names a variable is still a reference.
func TestExpandStillExpandsNames(t *testing.T) {
	t.Setenv("KEY_0", "value0")
	t.Setenv("_LEADING", "under")

	src := strings.Join([]string{
		`BARE=$KEY_0`,
		`BRACED=${KEY_0}`,
		`INSIDE=a-${KEY_0}-b`,
		`UNDERSCORE=$_LEADING`,
		`JOINED=$KEY_0$KEY_0`,
		`MISSING=${NOT_SET_ANYWHERE}/tail`,
	}, "\n")

	m, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct{ key, want string }{
		{"BARE", "value0"},
		{"BRACED", "value0"},
		{"INSIDE", "a-value0-b"},
		{"UNDERSCORE", "under"},
		{"JOINED", "value0value0"},
		{"MISSING", "/tail"},
	} {
		if got := m[tc.key]; got != tc.want {
			t.Errorf("%s = %q, want %q", tc.key, got, tc.want)
		}
	}
}

func TestExpandChainsWithinTheFile(t *testing.T) {
	src := "HOST=example.com\nURL=https://${HOST}/api\n"
	m, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	if m["URL"] != "https://example.com/api" {
		t.Errorf("URL = %q", m["URL"])
	}
}

func TestExpandSingleQuotesStayLiteral(t *testing.T) {
	t.Setenv("KEY_0", "value0")
	src := "LITERAL='keep $100 and $KEY_0'\nBACKTICK=`also $KEY_0`\n"
	m, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	if m["LITERAL"] != "keep $100 and $KEY_0" {
		t.Errorf("LITERAL = %q", m["LITERAL"])
	}
	if m["BACKTICK"] != "also $KEY_0" {
		t.Errorf("BACKTICK = %q", m["BACKTICK"])
	}
}

// TestExpandUnterminatedBraceIsText records the other half of the fix:
// os.Expand ate an unterminated "${" and everything after it.
func TestExpandUnterminatedBraceIsText(t *testing.T) {
	src := "A=${NOT_CLOSED\nB=${}\n"
	m, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	if m["A"] != "${NOT_CLOSED" {
		t.Errorf("A = %q", m["A"])
	}
	if m["B"] != "${}" {
		t.Errorf("B = %q", m["B"])
	}
}

func TestExpandRawSkipsSubstitution(t *testing.T) {
	t.Setenv("KEY_0", "value0")
	m, err := ParseRaw(strings.NewReader("A=$KEY_0\n"))
	if err != nil {
		t.Fatal(err)
	}
	if m["A"] != "$KEY_0" {
		t.Errorf("A = %q", m["A"])
	}
}

// TestLoadUsesTheSameRules covers the other call site: Load writes into
// the process environment and expanded on its own path.
func TestLoadUsesTheSameRules(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/.env"
	body := "KEY_0=value0\nPRICE=cost: $100\nURL=https://$KEY_0/api\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, k := range []string{"KEY_0", "PRICE", "URL"} {
		os.Unsetenv(k)
		t.Cleanup(func() { os.Unsetenv(k) })
	}

	if err := Load(path); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("PRICE"); got != "cost: $100" {
		t.Errorf("PRICE = %q", got)
	}
	if got := os.Getenv("URL"); got != "https://value0/api" {
		t.Errorf("URL = %q", got)
	}
}

func TestExpandFunction(t *testing.T) {
	t.Setenv("KEY_0", "value0")
	for _, tc := range []struct{ in, want string }{
		{"$KEY_0", "value0"},
		{"${KEY_0}", "value0"},
		{"cost: $100", "cost: $100"},
		{"pa$$word", "pa$$word"},
		{"$NOT_SET_ANYWHERE", ""},
	} {
		if got := Expand(tc.in); got != tc.want {
			t.Errorf("Expand(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
