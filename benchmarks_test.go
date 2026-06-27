package env

import (
	"net/url"
	"os"
	"strings"
	"testing"
)

// benchNested is the nested block of benchConfig.
type benchNested struct {
	Label string `env:"LABEL"`
	Value int    `env:"VALUE"`
}

// benchConfig is a small but representative configuration: scalars, a list, a
// url.URL, a nested struct and defaults.
type benchConfig struct {
	Host   string      `env:"HOST" def:"localhost"`
	Port   int         `env:"PORT" def:"8080"`
	Debug  bool        `env:"DEBUG"`
	Hosts  []string    `env:"HOSTS" sep:","`
	API    url.URL     `env:"API_URL"`
	Nested benchNested `env:"NESTED"`
}

// benchEnvFile is the .env payload used by the parse/load benchmarks.
const benchEnvFile = `HOST=localhost
PORT=8080
DEBUG=true
HOSTS=127.0.0.1,192.168.1.1,10.0.0.1
API_URL=https://api.example.com/v1
NESTED_LABEL=worker
NESTED_VALUE=42
`

// benchSource mirrors benchEnvFile as a decoded map.
func benchSource() map[string]string {
	return map[string]string{
		"HOST":         "localhost",
		"PORT":         "8080",
		"DEBUG":        "true",
		"HOSTS":        "127.0.0.1,192.168.1.1,10.0.0.1",
		"API_URL":      "https://api.example.com/v1",
		"NESTED_LABEL": "worker",
		"NESTED_VALUE": "42",
	}
}

func benchValue() benchConfig {
	return benchConfig{
		Host:   "localhost",
		Port:   8080,
		Debug:  true,
		Hosts:  []string{"127.0.0.1", "192.168.1.1", "10.0.0.1"},
		Nested: benchNested{Label: "worker", Value: 42},
	}
}

// ─── Environment helpers ─────────────────────────────────────────────────────

func BenchmarkSet(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		Set("BENCH_KEY", "value")
	}
}

func BenchmarkGet(b *testing.B) {
	Set("BENCH_KEY", "value")
	b.ReportAllocs()
	for b.Loop() {
		Get("BENCH_KEY")
	}
}

func BenchmarkLookup(b *testing.B) {
	Set("BENCH_KEY", "value")
	b.ReportAllocs()
	for b.Loop() {
		Lookup("BENCH_KEY")
	}
}

// ─── Parsing ─────────────────────────────────────────────────────────────────

func BenchmarkSplitN(b *testing.B) {
	const s = "value1,value2,value3,value4,value5"
	b.ReportAllocs()
	for b.Loop() {
		splitN(s, ",", -1)
	}
}

func BenchmarkParse(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		Parse(strings.NewReader(benchEnvFile))
	}
}

func BenchmarkLoadFile(b *testing.B) {
	path := b.TempDir() + "/.env"
	if err := os.WriteFile(path, []byte(benchEnvFile), 0o644); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		Load(path)
	}
}

// ─── Decoding (map → struct) ─────────────────────────────────────────────────

func BenchmarkUnmarshal(b *testing.B) {
	m := benchSource()
	b.ReportAllocs()
	for b.Loop() {
		var c benchConfig
		UnmarshalMap(m, &c)
	}
}

func BenchmarkUnmarshalConcurrent(b *testing.B) {
	m := benchSource()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var c benchConfig
			UnmarshalMap(m, &c)
		}
	})
}

func BenchmarkDecodeScalar(b *testing.B) {
	cases := []struct{ name, value string }{
		{"int", "12345"},
		{"float", "123.45"},
		{"bool", "true"},
		{"string", "value"},
	}
	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			m := map[string]string{"V": c.value}
			b.ReportAllocs()
			for b.Loop() {
				switch c.name {
				case "int":
					var v struct {
						V int `env:"V"`
					}
					UnmarshalMap(m, &v)
				case "float":
					var v struct {
						V float64 `env:"V"`
					}
					UnmarshalMap(m, &v)
				case "bool":
					var v struct {
						V bool `env:"V"`
					}
					UnmarshalMap(m, &v)
				case "string":
					var v struct {
						V string `env:"V"`
					}
					UnmarshalMap(m, &v)
				}
			}
		})
	}
}

// ─── Encoding (struct → map / string) ────────────────────────────────────────

func BenchmarkMarshal(b *testing.B) {
	c := benchValue()
	b.ReportAllocs()
	for b.Loop() {
		MarshalMap(c)
	}
}

func BenchmarkMarshalString(b *testing.B) {
	c := benchValue()
	b.ReportAllocs()
	for b.Loop() {
		MarshalString(c)
	}
}

// ─── Round-trip (struct → string → struct) ───────────────────────────────────

func BenchmarkRoundTrip(b *testing.B) {
	c := benchValue()
	b.ReportAllocs()
	for b.Loop() {
		s, _ := MarshalString(c)
		var back benchConfig
		UnmarshalString(s, &back)
	}
}
