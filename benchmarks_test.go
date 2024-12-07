package env

import (
	"fmt"
	"net/url"
	"os"
	"testing"
)

// Test structures
type testNestedConfig struct {
	Label string `env:"LABEL"`
	Value int    `env:"VALUE"`
}

type testConfig struct {
	Host         string           `env:"HOST" def:"localhost"`
	Port         int              `env:"PORT" def:"8080"`
	IPs          []string         `env:"ALLOWED_IPS" sep:","`
	IsProduction bool             `env:"IS_PROD"`
	API          url.URL          `env:"API_URL"`
	Nested       testNestedConfig `env:"NESTED"`
}

// Benchmark environment variable operations
func BenchmarkSet(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Set("TEST_KEY", "test_value")
	}
}

func BenchmarkGet(b *testing.B) {
	Set("BENCH_KEY", "bench_value")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Get("BENCH_KEY")
	}
}

func BenchmarkLookup(b *testing.B) {
	Set("LOOKUP_KEY", "lookup_value")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Lookup("LOOKUP_KEY")
	}
}

// Benchmark string parsing
func BenchmarkSplitN(b *testing.B) {
	str := "value1,value2,value3,value4,value5"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		splitN(str, ",", -1)
	}
}

// Benchmark struct operations
func BenchmarkMarshalSimple(b *testing.B) {
	config := testConfig{
		Host: "localhost",
		Port: 8080,
		IPs:  []string{"127.0.0.1", "192.168.1.1"},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Marshal("TEST_", config)
	}
}

func BenchmarkUnmarshalSimple(b *testing.B) {
	Set("TEST_HOST", "localhost")
	Set("TEST_PORT", "8080")
	Set("TEST_ALLOWED_IPS", "127.0.0.1,192.168.1.1")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var config testConfig
		Unmarshal("TEST_", &config)
	}
}

// Benchmark file operations
func BenchmarkLoadEnvFile(b *testing.B) {
	// Create temporary .env file for testing
	content := `
HOST=localhost
PORT=8080
ALLOWED_IPS=127.0.0.1,192.168.1.1
IS_PROD=true
API_URL=https://api.example.com
NESTED_LABEL=Test
NESTED_VALUE=42
`
	tmpfile := b.TempDir() + "/.env"
	if err := os.WriteFile(tmpfile, []byte(content), 0644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Load(tmpfile)
	}
}

// Benchmark parallel operations
func BenchmarkParallelUnmarshal(b *testing.B) {
	Set("TEST_HOST", "localhost")
	Set("TEST_PORT", "8080")
	Set("TEST_ALLOWED_IPS", "127.0.0.1,192.168.1.1")

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var config testConfig
			Unmarshal("TEST_", &config)
		}
	})
}

// Benchmark with different parallel tasks settings
func BenchmarkParallelTasks(b *testing.B) {
	tests := []int{2, 4, 8, 16}
	content := `
HOST=localhost
PORT=8080
ALLOWED_IPS=127.0.0.1,192.168.1.1
IS_PROD=true
API_URL=https://api.example.com
`
	tmpfile := b.TempDir() + "/.env"
	if err := os.WriteFile(tmpfile, []byte(content), 0644); err != nil {
		b.Fatal(err)
	}

	for _, tasks := range tests {
		b.Run(fmt.Sprintf("ParallelTasks-%d", tasks), func(b *testing.B) {
			ParallelTasks(tasks)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				Load(tmpfile)
			}
		})
	}
}

// Benchmark URL parsing
func BenchmarkURLParsing(b *testing.B) {
	Set("API_URL", "https://api.example.com/v1")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var config testConfig
		Unmarshal("", &config)
	}
}

// Benchmark type conversion
func BenchmarkTypeConversion(b *testing.B) {
	tests := map[string]string{
		"INT":    "12345",
		"FLOAT":  "123.45",
		"BOOL":   "true",
		"STRING": "test_value",
	}

	for typ, val := range tests {
		b.Run(typ, func(b *testing.B) {
			Set("TEST_"+typ, val)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				switch typ {
				case "INT":
					var i int
					unmarshalEnv("TEST_", &i)
				case "FLOAT":
					var f float64
					unmarshalEnv("TEST_", &f)
				case "BOOL":
					var bo bool
					unmarshalEnv("TEST_", &bo)
				case "STRING":
					var s string
					unmarshalEnv("TEST_", &s)
				}
			}
		})
	}
}
