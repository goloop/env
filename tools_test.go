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

// TestReadParseStoreWorngEqualValue tests problem with
// space after the equal sign.
func TestReadParseStoreWorngEqualValue(t *testing.T) {
	err := readParseStore("./fixtures/wrongequalvalue.env", false, true, false)
	if err == nil {
		t.Error("should be an error")
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
