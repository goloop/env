package env

import "testing"

// TestGetTagValues tests getTagValues function.
func TestGetTagValues(t *testing.T) {
	type test struct {
		value  string
		result [3]string
	}

	var tests = []test{
		{"a,b,c", [3]string{"a", "b", "c"}},
		{"a,,c", [3]string{"a", "", "c"}},
		{"a,,", [3]string{"a", "", ""}},
		{"a,(b,c),d,e,f", [3]string{"a", "(b,c)", "d,e,f"}},
	}

	for i, s := range tests {
		if r := getTagValues(s.value); r != s.result {
			t.Errorf("test %d is failed, expected %v but %v", i, s.result, r)
		}
	}
}

// TestGetTagArgs tests getTagArgs function.
func TestGetTagArgs(t *testing.T) {
	type test struct {
		value string
		args  *tagArgs
	}

	var tests = []test{
		{"a,b,c", &tagArgs{"a", "b", "c"}},
		{",b,c", &tagArgs{"default", "b", "c"}},
		{"-,b,c", &tagArgs{"-", "b", "c"}},
		{"a", &tagArgs{"a", "", ":"}},
		{"a,\"a, b, c\"", &tagArgs{"a", "a, b, c", ":"}},
		{"11", &tagArgs{"11", "", ":"}},
		{"a,b, ", &tagArgs{"a", "b", " "}}, // <-- set space for separator
		{"a,b ", &tagArgs{"a", "b ", ":"}},
	}

	for i, s := range tests {
		if args := getTagArgs(s.value, "default"); *args != *s.args {
			t.Errorf("test %d is failed, expected %v but %v",
				i, s.args, args)
		}
	}
}

// TestGetArgumentsMethods tests methods of the tagArgs struct.
func TestGetTagArgsMethods(t *testing.T) {
	type test struct {
		value     string
		isValid   bool
		isIgnored bool
	}

	var tests = []test{
		{"a,b,c", true, false},
		{",b,c", true, false},
		{"-,b,c", false, true},
		{"a", true, false},
		{"a,\"a, b, c\"", true, false},
		{"11", false, true},
	}

	for i, s := range tests {
		args := getTagArgs(s.value, "default")
		if args.IsValid() != s.isValid {
			t.Errorf("[isValid] test %d is failed, expected %v but %v",
				i, s.isValid, args.IsValid())
		}

		if args.IsIgnored() != s.isIgnored {
			t.Errorf("[isIgnored] test %d is failed, expected %v but %v",
				i, s.isIgnored, args.IsIgnored())
		}
	}
}
