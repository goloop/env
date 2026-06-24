package env

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// TestSts tests convert slice to string.
func TestSts(t *testing.T) {
	tests := []struct {
		value  interface{}
		result string
		sep    string
	}{
		{[]int{1, 2, 3}, "1,2,3", ","},
		{[]int{1, 2, 3}, "1:2:3", ":"},
		{[]int{1, 2, 3}, "1 2 3", " "},
		{[]string{"1", "2", "3"}, "1,2,3", ","},
		{[]string{"a,b,c", "d,e,f", "g,h,i"}, "a,b,c@d,e,f@g,h,i", "@"},
		{[]string{"a b c", "d e f", "g h i"}, "a b c@d e f@g h i", "@"},
	}

	for i, s := range tests {
		if r, _ := sts(s.value, s.sep); r != s.result {
			t.Errorf("test %d is failed, expected %v but %v", i, s.result, r)
		}
	}
}

// TestFts tests fts function.
func TestFts(t *testing.T) {
	data := struct {
		KeyA uint8
		KeyB int64
		KeyC []string
	}{
		KeyA: 10,
		KeyB: -20,
		KeyC: []string{"One", "Two", "Three"},
	}

	expected := fmt.Sprintf("%d", data.KeyA)
	if v := fts(data, "KEY_A", ""); v != expected {
		t.Errorf("expected %s but %s", expected, v)
	}

	expected = fmt.Sprintf("%d", data.KeyB)
	if v := fts(data, "KEY_B", ""); v != expected {
		t.Errorf("expected %s but %s", expected, v)
	}

	expected = fmt.Sprintf("%s", strings.Join(data.KeyC, ","))
	if v := fts(data, "KEY_C", ","); v != expected {
		t.Errorf("expected %s but %s", expected, v)
	}
}

// TestReadParseStoreOpen tests function to open a nonexistent file.
func TestLoadReadParseStoreOpen(t *testing.T) {
	err := readParseStore("./fixtures/nonexist.env", false, false, false)
	if err == nil {
		t.Error("should be an error for open a nonexistent file")
	}
}

// TestReadParseStoreExported checks the parsing of the
// env-file with the `export` command.
func TestReadParseStoreExported(t *testing.T) {
	tests := map[string]string{
		"KEY_0": "value 0",
		"KEY_1": "value 1",
		"KEY_2": "value_2",
		"KEY_3": "value_0:value_1:value_2:value_3",
	}

	// Load env-file.
	os.Clearenv()
	err := readParseStore("./fixtures/exported.env", false, false, false)
	if err != nil {
		t.Error(err)
	}

	// Compare with sample.
	for key, value := range tests {
		if v := os.Getenv(key); value != v {
			t.Errorf("incorrect value for `%s` key: `%s`!=`%s`", key, value, v)
		}
	}
}

// TestReadParseStoreComments checks the parsing of the
// env-file with the comments and empty strings.
func TestReadParseStoreComments(t *testing.T) {
	tests := map[string]string{
		"KEY_0": "value 0",
		"KEY_1": "value 1",
		"KEY_2": "value_2",
		"KEY_3": "value_3",
		"KEY_4": "value_4:value_4:value_4",
		"KEY_5": `some text with # sharp sign and "escaped quotation" mark`,
	}

	// Load env-file.
	os.Clearenv()
	err := readParseStore("./fixtures/comments.env", false, false, false)
	if err != nil {
		t.Error(err)
	}

	// Compare with sample.
	for key, value := range tests {
		if v := os.Getenv(key); value != v {
			t.Errorf("incorrect value for `%s` key: `%s`!=`%s`", key, value, v)
		}
	}
}

// TestReadParseStoreWorngEqualKey tests problem with
// spaces before the equal sign.
func TestReadParseStoreWorngEqualKey(t *testing.T) {
	err := readParseStore("./fixtures/wrongequalkey.env", false, false, false)
	if err == nil {
		t.Error("should be an error")
	}
}

// TestReadParseStoreSpaceAfterEqual checks that a space after the equal
// sign is valid (the value is trimmed), matching the dotenv specification.
func TestReadParseStoreSpaceAfterEqual(t *testing.T) {
	tests := map[string]string{
		"KEY_0": "value_0",
		"KEY_1": "value_1",
		"KEY_2": "value_2", // `KEY_2= "value_2"` - space after `=`.
		"KEY_3": "value_3",
	}

	os.Clearenv()
	err := readParseStore("./fixtures/wrongequalvalue.env", false, true, false)
	if err != nil {
		t.Error(err)
	}

	for key, value := range tests {
		if v := os.Getenv(key); value != v {
			t.Errorf("incorrect value for `%s` key: `%s`!=`%s`", key, value, v)
		}
	}
}

// TestReadParseStoreEmptyValue checks that empty values (`KEY=`, `KEY=""`)
// are valid and resolve to "" with Lookup reporting them as set.
func TestReadParseStoreEmptyValue(t *testing.T) {
	tests := map[string]string{
		"EMPTY":  "",
		"QUOTED": "",
		"AFTER":  "value",
	}

	os.Clearenv()
	err := readParseStore("./fixtures/empty.env", false, false, false)
	if err != nil {
		t.Error(err)
	}

	for key, value := range tests {
		v, ok := os.LookupEnv(key)
		if !ok {
			t.Errorf("key `%s` should be set (present) in the environment", key)
		}
		if v != value {
			t.Errorf("incorrect value for `%s` key: `%s`!=`%s`", key, value, v)
		}
	}
}

// TestReadParseStoreIgnoreWorngEntry tests to force loading with
// the incorrect lines.
func TestReadParseStoreIgnoreWorngEntry(t *testing.T) {
	forced := true
	tests := map[string]string{
		"KEY_0": "value_0",
		"KEY_1": "value_1",
		"KEY_4": "value_4",
		"KEY_5": "value",
		"KEY_6": "777",
		"KEY_7": "${KEY_1}",
	}

	// Load env-file.
	os.Clearenv()
	err := readParseStore("./fixtures/wrongentries.env", false, false, forced)
	if err != nil {
		t.Error(err.Error())
	}

	// Compare with sample.
	for key, value := range tests {
		if v := os.Getenv(key); value != v {
			t.Errorf("incorrect value for `%s` key: `%s`!=`%s`", key, value, v)
		}
	}
}

// TestReadParseStoreVariables tests replacing variables on real values.
func TestReadParseStoreVariables(t *testing.T) {
	expand := true
	tests := map[string]string{
		"KEY_0": "value_0",
		"KEY_1": "value_1",
		"KEY_2": "value_001",
		"KEY_3": "value_001->correct value",
		"KEY_4": "value_0value_001",
	}

	// Load env-file.
	os.Clearenv()
	err := readParseStore("./fixtures/variables.env", expand, false, false)
	if err != nil {
		t.Error(err.Error())
	}

	// Compare with sample.
	for key, value := range tests {
		if v := os.Getenv(key); value != v {
			t.Errorf("incorrect value for `%s` key: `%s`!=`%s`", key, value, v)
		}
	}
}

// TestReadParseStoreQuoteExpansion checks that variable expansion applies
// to double-quoted and unquoted values only; single quotes and backticks
// are literal per the dotenv specification.
func TestReadParseStoreQuoteExpansion(t *testing.T) {
	expand := true
	tests := map[string]string{
		"BASE":     "world",
		"DOUBLE":   "hello world", // double quotes expand.
		"SINGLE":   "hello $BASE", // single quotes are literal.
		"BACKTICK": "hello $BASE", // backticks are literal.
		"UNQUOTED": "hello-world", // unquoted expands.
	}

	os.Clearenv()
	err := readParseStore("./fixtures/expandquotes.env", expand, false, false)
	if err != nil {
		t.Error(err)
	}

	for key, value := range tests {
		if v := os.Getenv(key); value != v {
			t.Errorf("incorrect value for `%s` key: `%s`!=`%s`", key, value, v)
		}
	}
}

// TestReadParseStoreEscapes checks that escape sequences are interpreted
// inside double-quoted values (\n, \t, \\, \") and left literal inside
// single quotes, matching the dotenv specification.
func TestReadParseStoreEscapes(t *testing.T) {
	tests := map[string]string{
		"DOUBLE_NL":    "line1\nline2", // \n -> newline.
		"DOUBLE_TAB":   "a\tb",         // \t -> tab.
		"DOUBLE_BS":    `a\b`,          // \\ -> single backslash.
		"DOUBLE_QUOTE": `say "hi"`,     // \" -> quote.
		"SINGLE_NL":    `line1\nline2`, // single quotes: literal \n.
	}

	os.Clearenv()
	err := readParseStore("./fixtures/escapes.env", false, false, false)
	if err != nil {
		t.Error(err)
	}

	for key, value := range tests {
		if v := os.Getenv(key); value != v {
			t.Errorf("incorrect value for `%s` key: %q != %q", key, value, v)
		}
	}
}

// TestReadParseStoreMultiline checks that values spanning several physical
// lines inside quotes are joined with newlines, that expansion applies to
// double quotes only, and that parsing resumes after the closing quote.
func TestReadParseStoreMultiline(t *testing.T) {
	expand := true
	tests := map[string]string{
		"BASE":         "xyz",
		"MULTI_DOUBLE": "line1\nline2\nline3", // joined with newlines.
		"MULTI_SINGLE": "one\ntwo",            // literal, single quotes.
		"MULTI_EXPAND": "a\nxyz\nb",           // ${BASE} expanded.
		"TAIL":         "done",                // parsing resumes after value.
	}

	os.Clearenv()
	err := readParseStore("./fixtures/multiline.env", expand, false, false)
	if err != nil {
		t.Error(err)
	}

	for key, value := range tests {
		if v := os.Getenv(key); value != v {
			t.Errorf("incorrect value for `%s` key: %q != %q", key, value, v)
		}
	}
}

// TestReadParseStoreNotUpdate tests variable update protection.
func TestReadParseStoreNotUpdate(t *testing.T) {
	var (
		update = false
		err    error
	)

	// Set empty string
	os.Clearenv()
	if err = os.Setenv("KEY_0", ""); err != nil {
		t.Error(err)
	}

	// Read simple env-file with KEY_0.
	err = readParseStore("./fixtures/simple.env", false, update, false)
	if err != nil {
		t.Error(err.Error())
	}

	// Tests.
	if v := Get("KEY_0"); v != "" {
		t.Error("the value has been updated")
	}
}

// TestReadParseStoreUpdate tests variable update.
func TestReadParseStoreUpdate(t *testing.T) {
	update := true

	// Set empty string
	os.Clearenv()
	if err := os.Setenv("KEY_0", ""); err != nil {
		t.Error(err)
	}

	// Read simple env-file with KEY_0.
	err := readParseStore("./fixtures/simple.env", false, update, false)
	if err != nil {
		t.Error(err.Error())
	}

	// Tests.
	if v := Get("KEY_0"); v != "value 0" {
		t.Error("error: variable won't update")
	}
}

// TestSplitN tests splits the string at the specified rune-marker ignoring
// the position of the marker inside of the group: `...`, '...', "..."
// and (...), {...}, [...].
func TestSplitN(t *testing.T) {
	tests := []struct {
		n      int
		value  string
		result []string
	}{
		{-1, "a,b,c", []string{"a", "b", "c"}},
		{-1, "a,,c", []string{"a", "", "c"}},
		{-1, "a,,", []string{"a", "", ""}},
		{-1, "a,(b,c),d", []string{"a", "(b,c)", "d"}},
		{-1, "a,\"b,c\",d", []string{"a", "\"b,c\"", "d"}},
		{-1, "a,'b,c',d", []string{"a", "'b,c'", "d"}},
		{-1, "a,`b,c`,d", []string{"a", "`b,c`", "d"}},
		{-1, "a,b,c,d", []string{"a", "b", "c", "d"}},
		{0, "a,b,c,d", []string{}},
		{1, "a,b,c,d", []string{"a,b,c,d"}},
		{2, "a,b,c,d", []string{"a", "b,c,d"}},
		{3, "a,b,c,d", []string{"a", "b", "c,d"}},
		{4, "a,b,c,d", []string{"a", "b", "c", "d"}},
		{5, "a,b,c,d", []string{"a", "b", "c", "d"}},
		{3, "a_b_c,, ,", []string{"a_b_c", "", " ,"}},
		{-1, "a_b_c,, ,", []string{"a_b_c", "", " ", ""}},
	}

	for i, s := range tests {
		tmp := splitN(s.value, ",", s.n)
		r1, _ := sts(tmp, ":")
		r2, _ := sts(s.result, ":")
		if r1 != r2 {
			t.Errorf("test %d is failed, expected %v but %v", i, s.result, r1)
		}
	}
}

// TestSplitNUnicode tests that splitN keeps multi-byte (non-ASCII) values
// and separators intact. Indexing the string by byte used to corrupt
// Cyrillic, accented Latin and emoji and to cut the result short.
func TestSplitNUnicode(t *testing.T) {
	tests := []struct {
		value  string
		sep    string
		n      int
		result []string
	}{
		// Cyrillic values, ASCII separator.
		{"ключ,значення", ",", -1, []string{"ключ", "значення"}},
		// Accented Latin values, ASCII separator.
		{"café,naïve,über", ",", -1, []string{"café", "naïve", "über"}},
		// Emoji values, ASCII separator.
		{"🔑,🌍,✓", ",", -1, []string{"🔑", "🌍", "✓"}},
		// Non-ASCII (multi-byte) separator.
		{"a→b→c", "→", -1, []string{"a", "b", "c"}},
		// Multi-rune non-ASCII separator.
		{"один::два::три", "::", -1, []string{"один", "два", "три"}},
		// Grouping must still hold with multi-byte content.
		{"a,(б,в),г", ",", -1, []string{"a", "(б,в)", "г"}},
		// n>0 remainder must keep the rest of a multi-byte string intact.
		{"ключ,значення,решта", ",", 2, []string{"ключ", "значення,решта"}},
	}

	for i, s := range tests {
		tmp := splitN(s.value, s.sep, s.n)
		r1, _ := sts(tmp, "|")
		r2, _ := sts(s.result, "|")
		if r1 != r2 {
			t.Errorf("test %d failed, expected %v but got %v", i, s.result, tmp)
		}
	}
}
