package env

import (
	"errors"
	"testing"
)

// TestSentinelErrors checks that validation failures are testable with
// errors.Is against the exported sentinel errors.
func TestSentinelErrors(t *testing.T) {
	var empty struct{}
	notStruct := 5

	cases := []struct {
		name string
		err  error
		want error
	}{
		{"nil object", Unmarshal(nil), ErrNilObject},
		{"not a pointer", Unmarshal(struct {
			X int `env:"X"`
		}{}), ErrNotPointer},
		{"not a struct", Unmarshal(&notStruct), ErrNotStruct},
		{"empty struct", Unmarshal(&empty), ErrEmptyStruct},
	}

	for _, c := range cases {
		if !errors.Is(c.err, c.want) {
			t.Errorf("%s: expected %v, got %v", c.name, c.want, c.err)
		}
	}

	// Marshal with a non-struct value.
	if err := Marshal(5); !errors.Is(err, ErrInvalidObject) {
		t.Errorf("marshal non-struct: expected %v, got %v", ErrInvalidObject, err)
	}
}
