package env

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"testing"
)

// TestGet tests Get function.
func TestGet(t *testing.T) {
	tests := []struct {
		key   string
		value string
	}{
		{"KEY_0", "Abc"},
		{"KEY_1", "Def"},
	}

	// Call methods from os module to set test
	// variables to the environment.
	os.Clearenv()
	for _, test := range tests {
		os.Setenv(test.key, test.value)
	}

	// Test the method.
	for _, test := range tests {
		if v := Get(test.key); v != test.value {
			t.Errorf("expected `%s` but `%s`", test.value, v)
		}
	}
}

// TestSet tests Set function.
func TestSet(t *testing.T) {
	tests := []struct {
		key   string
		value string
	}{
		{"KEY_0", "Abc"},
		{"KEY_1", "Def"},
	}

	// Test the method.
	os.Clearenv()
	for _, test := range tests {
		if err := Set(test.key, test.value); err != nil {
			t.Error(err)
		}
	}

	// Call methods from os module to get test
	// variables from the environment.
	for _, test := range tests {
		if v := os.Getenv(test.key); v != test.value {
			t.Errorf("expected `%s` but `%s`", test.value, v)
		}
	}
}

// TestUnset tests Unset function.
func TestUnset(t *testing.T) {
	tests := []struct {
		key   string
		value string
	}{
		{"KEY_0", "Abc"},
		{"KEY_1", "Def"},
	}

	// Set test data.
	os.Clearenv()
	for _, test := range tests {
		if err := os.Setenv(test.key, test.value); err != nil {
			t.Error(err)
		}

		if v := os.Getenv(test.key); v != test.value {
			t.Errorf("expected `%s` but `%s`", test.value, v)
		}
	}

	// Erase the data and check the function.
	for _, test := range tests {
		if err := Unset(test.key); err != nil {
			t.Error(err)
		}

		if v := os.Getenv(test.key); v != "" {
			t.Errorf("must be cleaned but `%s`", v)
		}
	}
}

// TestClear tests Clear function.
func TestClear(t *testing.T) {
	tests := []struct {
		key   string
		value string
	}{
		{"KEY_0", "Abc"},
		{"KEY_1", "Def"},
	}

	// Set test data.
	os.Clearenv()
	for _, test := range tests {
		if err := os.Setenv(test.key, test.value); err != nil {
			t.Error(err)
		}

		if v := os.Getenv(test.key); v != test.value {
			t.Errorf("expected `%s` but `%s`", test.value, v)
		}
	}

	// Erase the data and check the function.
	Clear()
	for _, test := range tests {
		if v := os.Getenv(test.key); v != "" {
			t.Errorf("must be cleaned but `%s`", v)
		}
	}
}

// TestEnviron tests Environ function.
func TestEnviron(t *testing.T) {
	tests := map[string]string{
		"KEY_0": "Abc",
		"KEY_1": "Def",
	}

	// Set test data.
	os.Clearenv()
	for key, value := range tests {
		if err := os.Setenv(key, value); err != nil {
			t.Error(err)
		}
	}

	// Test function.
	for i, str := range Environ() {
		tmp := strings.Split(str, "=")
		key, value := tmp[0], tmp[1]
		if v, ok := tests[key]; v != value || !ok {
			if !ok {
				t.Errorf("test %v. extra key`%v`", i, key)
			} else {
				t.Errorf("test %v. expected `%v` but `%v`", i, v, value)
			}
		}
	}
}

// TestExpand tests Expand function.
func TestExpand(t *testing.T) {
	tests := map[string]string{
		"KEY_0": "7",
		"KEY_1": "5",
		"KEY_2": "3",
	}

	// Set test data.
	os.Clearenv()
	for key, value := range tests {
		if err := os.Setenv(key, value); err != nil {
			t.Error(err)
		}
	}

	// Test the replacement of keys with their data.
	for i := 0; i < 10; i++ {
		var tpl string

		keyA := fmt.Sprintf("KEY_%d", rand.Intn(len(tests)))
		keyB := fmt.Sprintf("KEY_%d", rand.Intn(len(tests)))
		keyC := fmt.Sprintf("KEY_%d", rand.Intn(len(tests)))

		exp := fmt.Sprintf("%s%s%s", tests[keyA], tests[keyB], tests[keyC])
		for _, key := range []string{keyA, keyB, keyC} {
			if rand.Intn(2) != 0 {
				tpl += fmt.Sprintf("${%s}", key)
			} else {
				tpl += fmt.Sprintf("$%s", key)
			}
		}

		if v := Expand(tpl); v != exp {
			t.Errorf("for keys `%s`. expected `%s` but `%s`", tpl, exp, v)
		}
	}
}

// TestLookup tests Lookup function.
func TestLookup(t *testing.T) {
	tests := []struct {
		key   string
		value string
	}{
		{"KEY_0", "Abc"},
		{"KEY_1", "Def"},
	}

	// Call methods from os module to set test
	// variables to the environment.
	os.Clearenv()
	for _, test := range tests {
		os.Setenv(test.key, test.value)
	}

	// Test the method.
	for _, test := range tests {
		if v, ok := Lookup(test.key); v != test.value || !ok {
			if !ok {
				t.Errorf("expected `%s` but the key is not set", test.value)
			} else {
				t.Errorf("expected `%s` but `%s`", test.value, v)
			}
		}
	}

	for _, key := range []string{"KEY_A", "KEY_B", "KEY_C"} {
		if _, ok := Lookup(key); ok {
			t.Errorf("the `%s` key must be empty", key)
		}
	}
}
