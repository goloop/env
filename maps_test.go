package env

import (
	"os"
	"testing"
)

// TestMarshalMapRoundTrip checks MarshalMap -> UnmarshalMap round-trips and
// that MarshalMap does not touch the process environment.
func TestMarshalMapRoundTrip(t *testing.T) {
	type cfg struct {
		Host  string   `env:"HOST"`
		Port  int      `env:"PORT"`
		Hosts []string `env:"HOSTS" sep:":"`
	}
	in := cfg{Host: "localhost", Port: 8080, Hosts: []string{"a", "b"}}

	os.Clearenv()
	m, err := MarshalMap(in)
	if err != nil {
		t.Fatal(err)
	}

	// MarshalMap must not change the environment.
	if _, ok := os.LookupEnv("HOST"); ok {
		t.Error("MarshalMap must not write to the environment")
	}
	if m["HOST"] != "localhost" || m["PORT"] != "8080" || m["HOSTS"] != "a:b" {
		t.Errorf("unexpected map: %v", m)
	}

	var out cfg
	if err := UnmarshalMap(m, &out); err != nil {
		t.Fatal(err)
	}
	if out.Host != in.Host || out.Port != in.Port || len(out.Hosts) != 2 {
		t.Errorf("round-trip mismatch: %+v != %+v", out, in)
	}
}

// TestWithPrefixNormalization checks that WithPrefix names a namespace level
// joined with "_": both "APP" and "APP_" resolve APP_PORT, and an empty prefix
// adds no leading underscore.
func TestWithPrefixNormalization(t *testing.T) {
	type cfg struct {
		Port int `env:"PORT"`
	}
	m := map[string]string{"APP_PORT": "8080"}

	for _, prefix := range []string{"APP", "APP_"} {
		var c cfg
		if err := UnmarshalMap(m, &c, WithPrefix(prefix)); err != nil {
			t.Fatal(err)
		}
		if c.Port != 8080 {
			t.Errorf("WithPrefix(%q): expected 8080, got %d", prefix, c.Port)
		}
	}

	var c cfg
	if err := UnmarshalMap(map[string]string{"PORT": "80"}, &c); err != nil {
		t.Fatal(err)
	}
	if c.Port != 80 {
		t.Errorf("no prefix: expected 80, got %d", c.Port)
	}
}

// TestWithSeparator checks that WithSeparator sets the default list separator
// for fields without a sep tag.
func TestWithSeparator(t *testing.T) {
	type cfg struct {
		Hosts []string `env:"HOSTS"`
	}
	var c cfg
	m := map[string]string{"HOSTS": "a,b,c"}
	if err := UnmarshalMap(m, &c, WithSeparator(",")); err != nil {
		t.Fatal(err)
	}
	if len(c.Hosts) != 3 {
		t.Errorf("expected 3 hosts, got %v", c.Hosts)
	}
}

// mapReceiver implements the Unmarshaler interface and records the data map.
type mapReceiver struct {
	got map[string]string
}

func (r *mapReceiver) UnmarshalEnv(data map[string]string) error {
	r.got = data
	return nil
}

// TestUnmarshalerReceivesMap checks that a custom Unmarshaler is handed the
// resolved source map.
func TestUnmarshalerReceivesMap(t *testing.T) {
	src := map[string]string{"A": "1", "B": "2"}
	var r mapReceiver
	if err := UnmarshalMap(src, &r); err != nil {
		t.Fatal(err)
	}
	if r.got["A"] != "1" || r.got["B"] != "2" {
		t.Errorf("custom Unmarshaler did not receive the source map: %v", r.got)
	}
}
