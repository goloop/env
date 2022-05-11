package env

import "testing"

// TestSplitN tests splits the string at the specified rune-marker ignoring
// the position of the marker inside of the group: `...`, '...', "..."
// and (...), {...}, [...].
func TestSplitN(t *testing.T) {
	type test struct {
		n      int
		value  string
		result []string
	}

	var tests = []test{
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
		if r := splitN(s.value, ",", s.n); sts(r, ":") != sts(s.result, ":") {
			t.Errorf("test %d is failed, expected %v but %v", i, s.result, r)
		}
	}
}
