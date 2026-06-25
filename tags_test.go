package env

import (
	"errors"
	"testing"
	"time"
)

// TestRequired checks the inline required flag.
func TestRequired(t *testing.T) {
	type cfg struct {
		Port int `env:"PORT,required"`
	}

	var missing cfg
	if err := UnmarshalMap(map[string]string{}, &missing); !errors.Is(err, ErrRequired) {
		t.Errorf("expected ErrRequired, got %v", err)
	}

	var present cfg
	if err := UnmarshalMap(map[string]string{"PORT": "8080"}, &present); err != nil {
		t.Fatal(err)
	}
	if present.Port != 8080 {
		t.Errorf("Port = %d, want 8080", present.Port)
	}
}

// TestRequiredWithDefault checks that a default satisfies a required field.
func TestRequiredWithDefault(t *testing.T) {
	type cfg struct {
		Port int `env:"PORT,required" def:"80"`
	}

	var c cfg
	if err := UnmarshalMap(map[string]string{}, &c); err != nil {
		t.Errorf("required+def should not error: %v", err)
	}
	if c.Port != 80 {
		t.Errorf("Port = %d, want 80", c.Port)
	}
}

// TestIgnoreField checks that env:"-" skips a field on both decode and encode.
func TestIgnoreField(t *testing.T) {
	type cfg struct {
		Visible string `env:"VISIBLE"`
		Hidden  string `env:"-"`
	}

	c := cfg{Hidden: "keep"}
	if err := UnmarshalMap(map[string]string{"VISIBLE": "v", "-": "x"}, &c); err != nil {
		t.Fatal(err)
	}
	if c.Visible != "v" {
		t.Errorf("Visible = %q", c.Visible)
	}
	if c.Hidden != "keep" {
		t.Errorf("ignored field must be untouched, got %q", c.Hidden)
	}

	m, err := MarshalMap(cfg{Visible: "v", Hidden: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := m["-"]; ok || len(m) != 1 {
		t.Errorf("ignored field must not be marshaled: %v", m)
	}
}

// TestDuration checks time.Duration decode/encode round-trip.
func TestDuration(t *testing.T) {
	type cfg struct {
		Timeout time.Duration `env:"TIMEOUT"`
	}

	var c cfg
	if err := UnmarshalMap(map[string]string{"TIMEOUT": "1h30m"}, &c); err != nil {
		t.Fatal(err)
	}
	if c.Timeout != 90*time.Minute {
		t.Errorf("Timeout = %v, want 1h30m", c.Timeout)
	}

	m, err := MarshalMap(cfg{Timeout: 90 * time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	if m["TIMEOUT"] != "1h30m0s" {
		t.Errorf("marshaled TIMEOUT = %q", m["TIMEOUT"])
	}
}

// TestTimeRFC3339 checks time.Time with the default layout.
func TestTimeRFC3339(t *testing.T) {
	type cfg struct {
		At time.Time `env:"AT"`
	}

	var c cfg
	if err := UnmarshalMap(map[string]string{"AT": "2026-06-25T10:00:00Z"}, &c); err != nil {
		t.Fatal(err)
	}
	if c.At.Year() != 2026 || c.At.Month() != time.June || c.At.Day() != 25 {
		t.Errorf("At = %v", c.At)
	}

	m, err := MarshalMap(cfg{At: c.At})
	if err != nil {
		t.Fatal(err)
	}
	if m["AT"] != "2026-06-25T10:00:00Z" {
		t.Errorf("marshaled AT = %q", m["AT"])
	}
}

// TestTimeCustomLayout checks the layout tag (literal and named constant).
func TestTimeCustomLayout(t *testing.T) {
	type literal struct {
		Day time.Time `env:"DAY" layout:"2006-01-02"`
	}
	type named struct {
		Day time.Time `env:"DAY" layout:"DateOnly"`
	}

	for _, name := range []string{"literal", "named"} {
		var day time.Time
		var err error
		if name == "literal" {
			var c literal
			err = UnmarshalMap(map[string]string{"DAY": "2026-06-25"}, &c)
			day = c.Day
		} else {
			var c named
			err = UnmarshalMap(map[string]string{"DAY": "2026-06-25"}, &c)
			day = c.Day
		}
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if day.Day() != 25 || day.Month() != time.June {
			t.Errorf("%s: Day = %v", name, day)
		}
	}
}

// TestWithTimeLayout checks the call-level default layout option.
func TestWithTimeLayout(t *testing.T) {
	type cfg struct {
		Day time.Time `env:"DAY"`
	}

	var c cfg
	err := UnmarshalMap(map[string]string{"DAY": "2026-06-25"}, &c, WithTimeLayout("DateOnly"))
	if err != nil {
		t.Fatal(err)
	}
	if c.Day.Day() != 25 {
		t.Errorf("Day = %v", c.Day)
	}
}
