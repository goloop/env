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

// TestFts tests fts function.
func TestFts(t *testing.T) {
	var data = struct {
		KeyA uint8
		KeyB int64
		KeyC []string
	}{
		KeyA: 10,
		KeyB: -20,
		KeyC: []string{"One", "Two", "Three"},
	}

	if v := fts(data, "KEY_A", ""); v != "10" {
		t.Errorf("expected 10 but %v", v)
	}

	if v := fts(data, "KEY_B", ""); v != "-20" {
		t.Errorf("expected -20 but %v", v)
	}

	if v := fts(data, "KEY_C", ","); v != "One,Two,Three" {
		t.Errorf("expected One,Two,Three but %v", v)
	}
}
