package env_test

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/goloop/env/v2"
)

// money is a third-party-like type that implements neither TextMarshaler nor
// TextUnmarshaler, so it can only be handled via WithParser/WithEncoder.
type money struct{ cents int }

func parseMoney(s string) (money, error) {
	f, err := strconv.ParseFloat(strings.TrimPrefix(s, "$"), 64)
	if err != nil {
		return money{}, err
	}
	return money{int(f * 100)}, nil
}

func encodeMoney(m money) (string, error) {
	return fmt.Sprintf("$%.2f", float64(m.cents)/100), nil
}

// TestWithParserEncoder checks that a registered parser/encoder handles a type
// (and slices and pointers of it) on both decode and encode, with a round-trip.
func TestWithParserEncoder(t *testing.T) {
	type cfg struct {
		Price money   `env:"PRICE"`
		Tiers []money `env:"TIERS" sep:","`
		Opt   *money  `env:"OPT"`
	}
	opts := []env.Option{env.WithParser(parseMoney), env.WithEncoder(encodeMoney)}

	var c cfg
	err := env.UnmarshalMap(map[string]string{
		"PRICE": "$1.50",
		"TIERS": "$1.00,$2.50",
		"OPT":   "$9.99",
	}, &c, opts...)
	if err != nil {
		t.Fatal(err)
	}
	if c.Price.cents != 150 {
		t.Errorf("Price = %v, want 150", c.Price)
	}
	if len(c.Tiers) != 2 || c.Tiers[0].cents != 100 || c.Tiers[1].cents != 250 {
		t.Errorf("Tiers = %v", c.Tiers)
	}
	if c.Opt == nil || c.Opt.cents != 999 {
		t.Errorf("Opt = %v", c.Opt)
	}

	// Encode with the registered encoder, then round-trip.
	m, err := env.MarshalMap(c, opts...)
	if err != nil {
		t.Fatal(err)
	}
	if m["PRICE"] != "$1.50" || m["TIERS"] != "$1.00,$2.50" || m["OPT"] != "$9.99" {
		t.Errorf("encode: %v", m)
	}

	var back cfg
	if err := env.UnmarshalMap(m, &back, opts...); err != nil {
		t.Fatal(err)
	}
	if back.Price != c.Price || len(back.Tiers) != 2 || *back.Opt != *c.Opt {
		t.Errorf("round-trip mismatch: %+v vs %+v", back, c)
	}
}

// TestWithParserAbsentPointer checks that an absent pointer field with a
// registered parser stays nil (the parser is not called with an empty value).
func TestWithParserAbsentPointer(t *testing.T) {
	type cfg struct {
		Opt *money `env:"OPT"`
	}
	var c cfg
	if err := env.UnmarshalMap(map[string]string{}, &c, env.WithParser(parseMoney)); err != nil {
		t.Fatal(err)
	}
	if c.Opt != nil {
		t.Errorf("absent *money = %v, want nil", c.Opt)
	}
}
