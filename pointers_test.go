package env_test

import (
	"errors"
	"testing"

	"github.com/goloop/env/v2"
)

type ptrInner struct {
	X int `env:"X"`
}

// TestNoPanicPublicAPI is the safety net: no public call may panic on input a
// user can realistically pass (nil objects, nil pointer fields, nil slice
// elements, unexported fields).
func TestNoPanicPublicAPI(t *testing.T) {
	cases := []struct {
		name string
		fn   func()
	}{
		{"MarshalMap(nil)", func() { env.MarshalMap(nil) }},
		{"Marshal(nil)", func() { env.Marshal(nil) }},
		{"Unmarshal(nil)", func() { env.Unmarshal(nil) }},
		{"MarshalMap nil scalar ptr field", func() {
			env.MarshalMap(struct {
				P *int `env:"P"`
			}{})
		}},
		{"MarshalMap nil struct ptr field", func() {
			env.MarshalMap(struct {
				I *ptrInner `env:"I"`
			}{})
		}},
		{"MarshalMap nil slice element", func() {
			env.MarshalMap(struct {
				A []*int `env:"A" sep:","`
			}{A: []*int{nil}})
		}},
		{"Unmarshal nil scalar ptr field", func() {
			var s struct {
				P *int `env:"P"`
			}
			env.UnmarshalMap(map[string]string{"P": "5"}, &s)
		}},
		{"Unmarshal unexported field", func() {
			var s struct {
				Port   int `env:"PORT"`
				secret string
			}
			_ = s.secret
			env.UnmarshalMap(map[string]string{"PORT": "9090"}, &s)
		}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("panicked: %v", r)
				}
			}()
			c.fn()
		})
	}
}

// TestPointerContract verifies the nil-pointer contract: allocate only when
// there is a value to assign; otherwise leave nil; marshal omits nil so the
// value round-trips.
func TestPointerContract(t *testing.T) {
	type cfg struct {
		P *int `env:"P"`
	}

	// Present -> allocate and set.
	var present cfg
	if err := env.UnmarshalMap(map[string]string{"P": "5"}, &present); err != nil {
		t.Fatal(err)
	}
	if present.P == nil || *present.P != 5 {
		t.Errorf("present: expected *5, got %v", present.P)
	}

	// Absent -> stay nil.
	var absent cfg
	if err := env.UnmarshalMap(map[string]string{}, &absent); err != nil {
		t.Fatal(err)
	}
	if absent.P != nil {
		t.Errorf("absent: expected nil, got %v", *absent.P)
	}

	// Marshal omits nil; round-trip keeps it nil.
	m, err := env.MarshalMap(cfg{})
	if err != nil {
		t.Fatal(err)
	}
	if len(m) != 0 {
		t.Errorf("nil pointer must be omitted, got %v", m)
	}
	var back cfg
	env.UnmarshalMap(m, &back)
	if back.P != nil {
		t.Error("round-trip must keep nil")
	}
}

// TestNestedStructPointer checks that an optional nested-struct pointer is
// allocated only when the source has keys under its prefix.
func TestNestedStructPointer(t *testing.T) {
	type cfg struct {
		I *ptrInner `env:"I"`
	}

	var withSub cfg
	if err := env.UnmarshalMap(map[string]string{"I_X": "7"}, &withSub); err != nil {
		t.Fatal(err)
	}
	if withSub.I == nil || withSub.I.X != 7 {
		t.Errorf("expected I.X=7, got %v", withSub.I)
	}

	var noSub cfg
	env.UnmarshalMap(map[string]string{}, &noSub)
	if noSub.I != nil {
		t.Error("nested pointer must stay nil when no subkeys are present")
	}
}

// TestUnexportedFieldSkipped checks that unexported fields are ignored on both
// decode and encode (no panic), like encoding/json.
func TestUnexportedFieldSkipped(t *testing.T) {
	type cfg struct {
		Port   int `env:"PORT"`
		secret string
	}

	var c cfg
	if err := env.UnmarshalMap(map[string]string{"PORT": "9090"}, &c); err != nil {
		t.Fatal(err)
	}
	if c.Port != 9090 {
		t.Errorf("Port=%d, want 9090", c.Port)
	}

	c.secret = "x"
	m, err := env.MarshalMap(c)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := m["secret"]; ok {
		t.Errorf("unexported field must not be marshaled: %v", m)
	}
}

// TestMarshalNilObject checks that marshaling nil returns ErrNilObject.
func TestMarshalNilObject(t *testing.T) {
	if _, err := env.MarshalMap(nil); !errors.Is(err, env.ErrNilObject) {
		t.Errorf("expected ErrNilObject, got %v", err)
	}
	if err := env.Marshal(nil); !errors.Is(err, env.ErrNilObject) {
		t.Errorf("expected ErrNilObject, got %v", err)
	}
}
