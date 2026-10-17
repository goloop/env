package env_test

import (
	"github.com/goloop/env/v2"
	"testing"
	"time"
)

type emptyText string

func (v *emptyText) UnmarshalText(b []byte) error { *v = emptyText(b); return nil }

func TestExplicitEmptyResetsTimeAndCallsTextDecoder(t *testing.T) {
	cfg := struct {
		Duration time.Duration
		Time     time.Time
		Text     emptyText
	}{time.Hour, time.Now(), "old"}
	if err := env.UnmarshalMap(map[string]string{"Duration": "", "Time": "", "Text": ""}, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Duration != 0 || !cfg.Time.IsZero() || cfg.Text != "" {
		t.Fatalf("fields not cleared: %+v", cfg)
	}
}

func TestParserCanReturnNilInterface(t *testing.T) {
	cfg := struct{ Value any }{"old"}
	if err := env.UnmarshalMap(map[string]string{"Value": "clear"}, &cfg, env.WithParser(func(string) (any, error) { return nil, nil })); err != nil {
		t.Fatal(err)
	}
	if cfg.Value != nil {
		t.Fatalf("got %#v", cfg.Value)
	}
}
