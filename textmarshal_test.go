package env_test

import (
	"fmt"
	"net"
	"reflect"
	"testing"

	"github.com/goloop/env/v2"
)

// level is a custom enum that implements encoding.TextMarshaler and
// encoding.TextUnmarshaler.
type level int

const (
	levelInfo level = iota
	levelWarn
	levelError
)

func (l *level) UnmarshalText(b []byte) error {
	switch string(b) {
	case "info":
		*l = levelInfo
	case "warn":
		*l = levelWarn
	case "error":
		*l = levelError
	default:
		return fmt.Errorf("invalid level %q", b)
	}
	return nil
}

func (l level) MarshalText() ([]byte, error) {
	return []byte([...]string{"info", "warn", "error"}[l]), nil
}

// TestTextUnmarshaler checks that types implementing TextUnmarshaler decode,
// including a stdlib type whose kind is a slice (net.IP), a custom enum,
// slices of them and pointers to them.
func TestTextUnmarshaler(t *testing.T) {
	type cfg struct {
		IP  net.IP   `env:"IP"`
		Lvl level    `env:"LVL"`
		IPs []net.IP `env:"IPS" sep:","`
		PIP *net.IP  `env:"PIP"`
	}

	var c cfg
	err := env.UnmarshalMap(map[string]string{
		"IP":  "10.0.0.1",
		"LVL": "warn",
		"IPS": "1.1.1.1,8.8.8.8",
		"PIP": "127.0.0.1",
	}, &c)
	if err != nil {
		t.Fatal(err)
	}

	if c.IP.String() != "10.0.0.1" {
		t.Errorf("IP = %v", c.IP)
	}
	if c.Lvl != levelWarn {
		t.Errorf("Lvl = %v, want warn", c.Lvl)
	}
	if len(c.IPs) != 2 || c.IPs[0].String() != "1.1.1.1" || c.IPs[1].String() != "8.8.8.8" {
		t.Errorf("IPs = %v", c.IPs)
	}
	if c.PIP == nil || c.PIP.String() != "127.0.0.1" {
		t.Errorf("PIP = %v", c.PIP)
	}

	// An invalid value surfaces the type's error.
	var bad cfg
	if err := env.UnmarshalMap(map[string]string{"LVL": "nope"}, &bad); err == nil {
		t.Error("expected an error for an invalid enum value")
	}
}

// TestTextMarshalerRoundTrip checks the encode side and a full round-trip.
func TestTextMarshalerRoundTrip(t *testing.T) {
	type cfg struct {
		IP  net.IP   `env:"IP"`
		Lvl level    `env:"LVL"`
		IPs []net.IP `env:"IPS" sep:","`
	}
	in := cfg{
		IP:  net.ParseIP("10.0.0.1"),
		Lvl: levelError,
		IPs: []net.IP{net.ParseIP("1.1.1.1"), net.ParseIP("8.8.8.8")},
	}

	m, err := env.MarshalMap(in)
	if err != nil {
		t.Fatal(err)
	}
	if m["IP"] != "10.0.0.1" || m["LVL"] != "error" || m["IPS"] != "1.1.1.1,8.8.8.8" {
		t.Errorf("marshal: %v", m)
	}

	var out cfg
	if err := env.UnmarshalMap(m, &out); err != nil {
		t.Fatal(err)
	}
	if out.Lvl != in.Lvl || out.IP.String() != in.IP.String() || !reflect.DeepEqual(toStrs(out.IPs), toStrs(in.IPs)) {
		t.Errorf("round-trip: got %+v, want %+v", out, in)
	}
}

func toStrs(ips []net.IP) []string {
	s := make([]string, len(ips))
	for i, ip := range ips {
		s[i] = ip.String()
	}
	return s
}

// ptrLevel implements TextMarshaler and TextUnmarshaler with POINTER receivers,
// a common idiom; both must work on encode and decode.
type ptrLevel int

func (l *ptrLevel) UnmarshalText(b []byte) error {
	switch string(b) {
	case "lo":
		*l = 0
	case "hi":
		*l = 1
	default:
		return fmt.Errorf("bad ptrLevel %q", b)
	}
	return nil
}

func (l *ptrLevel) MarshalText() ([]byte, error) {
	return []byte([...]string{"lo", "hi"}[*l]), nil
}

// TestPointerReceiverTextMarshaler checks that a pointer-receiver MarshalText is
// honoured on encode (it used to be ignored, encoding the kind default).
func TestPointerReceiverTextMarshaler(t *testing.T) {
	type cfg struct {
		L ptrLevel `env:"L"`
	}

	// Encoding a value (not a pointer) must still call the pointer-receiver.
	m, err := env.MarshalMap(cfg{L: 1})
	if err != nil {
		t.Fatal(err)
	}
	if m["L"] != "hi" {
		t.Errorf("pointer-receiver MarshalText: got %q, want %q", m["L"], "hi")
	}

	var back cfg
	if err := env.UnmarshalMap(m, &back); err != nil {
		t.Fatal(err)
	}
	if back.L != 1 {
		t.Errorf("round-trip: got %d, want 1", back.L)
	}
}
