package env

import (
	"os"
	"testing"
)

// A nested struct namespaces its fields, which is usually what you want and
// occasionally exactly what you cannot have: deployments name some variables
// once and for all, and DATABASE_URL does not become DB_DATABASE_URL because
// the Go type groups it.
func TestAbsoluteEscapesThePrefix(t *testing.T) {
	type DB struct {
		URL      string `env:"DATABASE_URL,absolute"`
		PoolSize int    `env:"POOL_SIZE"`
	}
	type Config struct {
		DB DB `env:"DB"`
	}

	t.Setenv("DATABASE_URL", "postgres://localhost/app")
	t.Setenv("DB_POOL_SIZE", "12")

	var c Config
	if err := Unmarshal(&c); err != nil {
		t.Fatal(err)
	}
	if c.DB.URL != "postgres://localhost/app" {
		t.Errorf("URL = %q, want the value of DATABASE_URL", c.DB.URL)
	}
	if c.DB.PoolSize != 12 {
		t.Errorf("PoolSize = %d, want the value of DB_POOL_SIZE - a sibling "+
			"without the flag must keep its prefix", c.DB.PoolSize)
	}
}

// The flag drops the whole chain, not one level of it: a name that is
// absolute is absolute.
func TestAbsoluteDropsTheWholeChain(t *testing.T) {
	type Inner struct {
		Key string `env:"SECRET_KEY,absolute"`
	}
	type Middle struct {
		Inner Inner `env:"INNER"`
	}
	type Config struct {
		Middle Middle `env:"OUTER"`
	}

	t.Setenv("SECRET_KEY", "s3cret")

	var c Config
	if err := Unmarshal(&c); err != nil {
		t.Fatal(err)
	}
	if c.Middle.Inner.Key != "s3cret" {
		t.Errorf("Key = %q, want the value of SECRET_KEY", c.Middle.Inner.Key)
	}
}

// The flag composes with required, and required still means required.
func TestAbsoluteWithRequired(t *testing.T) {
	type DB struct {
		URL string `env:"DATABASE_URL,absolute,required"`
	}
	type Config struct {
		DB DB `env:"DB"`
	}

	os.Unsetenv("DATABASE_URL")
	os.Unsetenv("DB_DATABASE_URL")

	var c Config
	if err := Unmarshal(&c); err == nil {
		t.Error("a missing absolute required key was accepted")
	}

	t.Setenv("DATABASE_URL", "postgres://localhost/app")
	if err := Unmarshal(&c); err != nil {
		t.Errorf("Unmarshal() = %v, want the absolute key to satisfy required", err)
	}
}

// A call-level prefix is a prefix too, and absolute means absolute.
func TestAbsoluteIgnoresWithPrefix(t *testing.T) {
	type Config struct {
		URL  string `env:"DATABASE_URL,absolute"`
		Port int    `env:"PORT"`
	}

	t.Setenv("DATABASE_URL", "postgres://localhost/app")
	t.Setenv("APP_PORT", "8080")

	var c Config
	if err := Unmarshal(&c, WithPrefix("APP")); err != nil {
		t.Fatal(err)
	}
	if c.URL != "postgres://localhost/app" {
		t.Errorf("URL = %q, want DATABASE_URL despite the call prefix", c.URL)
	}
	if c.Port != 8080 {
		t.Errorf("Port = %d, want APP_PORT", c.Port)
	}
}

// Marshal has to write the same names Unmarshal reads, or a config cannot
// round-trip through its own environment.
func TestAbsoluteRoundTrips(t *testing.T) {
	type DB struct {
		URL      string `env:"DATABASE_URL,absolute"`
		PoolSize int    `env:"POOL_SIZE"`
	}
	type Config struct {
		DB DB `env:"DB"`
	}

	for _, k := range []string{"DATABASE_URL", "DB_POOL_SIZE", "DB_DATABASE_URL"} {
		os.Unsetenv(k)
		t.Cleanup(func() { os.Unsetenv(k) })
	}

	if err := Marshal(Config{DB: DB{URL: "postgres://x", PoolSize: 3}}); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("DATABASE_URL"); got != "postgres://x" {
		t.Errorf("Marshal wrote %q under DATABASE_URL", got)
	}
	if got := os.Getenv("DB_POOL_SIZE"); got != "3" {
		t.Errorf("Marshal wrote %q under DB_POOL_SIZE", got)
	}
	if _, ok := os.LookupEnv("DB_DATABASE_URL"); ok {
		t.Error("Marshal wrote the prefixed name for an absolute key")
	}

	// And what Marshal wrote, Unmarshal reads back.
	var back Config
	if err := Unmarshal(&back); err != nil {
		t.Fatal(err)
	}
	if back.DB.URL != "postgres://x" || back.DB.PoolSize != 3 {
		t.Errorf("round trip gave %+v", back.DB)
	}
}

// The compatibility half: a tag without the flag behaves exactly as before.
func TestWithoutAbsoluteNothingChanges(t *testing.T) {
	type DB struct {
		URL string `env:"DATABASE_URL"`
	}
	type Config struct {
		DB DB `env:"DB"`
	}

	t.Setenv("DB_DATABASE_URL", "prefixed")
	t.Setenv("DATABASE_URL", "bare")

	var c Config
	if err := Unmarshal(&c); err != nil {
		t.Fatal(err)
	}
	if c.DB.URL != "prefixed" {
		t.Errorf("URL = %q, want the prefixed key", c.DB.URL)
	}
}
