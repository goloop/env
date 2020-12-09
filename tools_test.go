package env

import "testing"

// TestSts tests convert slice to string.
func TestSts(t *testing.T) {
	type test struct {
		value  interface{}
		result string
		sep    string
	}

	var tests = []test{
		{[]int{1, 2, 3}, "1,2,3", ","},
		{[]int{1, 2, 3}, "1:2:3", ":"},
		{[]int{1, 2, 3}, "1 2 3", " "},
		{[]string{"1", "2", "3"}, "1,2,3", ","},
		{[]string{"a,b,c", "d,e,f", "g,h,i"}, "a,b,c@d,e,f@g,h,i", "@"},
		{[]string{"a b c", "d e f", "g h i"}, "a b c@d e f@g h i", "@"},
	}

	for i, s := range tests {
		if r := sts(s.value, s.sep); r != s.result {
			t.Errorf("test %d is failed, expected %v but %v", i, s.result, r)
		}
	}
}

// TestSplit tests splits the string at the specified rune-marker ignoring
// the position of the marker inside of the group: `...`, '...', "..."
// and (...), {...}, [...].
func TestSplit(t *testing.T) {
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
	}

	for i, s := range tests {
		if r := splitN(s.value, ",", s.n); sts(r, ":") != sts(s.result, ":") {
			t.Errorf("test %d is failed, expected %v but %v", i, s.result, r)
		}
	}
}
