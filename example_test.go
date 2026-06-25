package env_test

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/goloop/env/v2"
)

// Unmarshal reads values from the process environment into a struct. Field
// names are matched by the env tag (or the field name when the tag is empty).
func ExampleUnmarshal() {
	env.Clear()
	env.Set("HOST", "0.0.0.0")
	env.Set("PORT", "8080")
	env.Set("ALLOWED_HOSTS", "localhost:127.0.0.1")

	type Config struct {
		Host         string   `env:"HOST"`
		Port         int      `env:"PORT" def:"80"`
		AllowedHosts []string `env:"ALLOWED_HOSTS" sep:":"`
	}

	var c Config
	if err := env.Unmarshal(&c); err != nil {
		log.Fatal(err)
	}

	fmt.Println(c.Host)
	fmt.Println(c.Port)
	fmt.Println(c.AllowedHosts)
	// Output:
	// 0.0.0.0
	// 8080
	// [localhost 127.0.0.1]
}

// WithPrefix reads a namespaced subset of the environment. Levels are joined
// with "_", so WithPrefix("APP") maps the field PORT to APP_PORT. The same
// component struct can be reused for several prefixes.
func ExampleUnmarshal_withPrefix() {
	env.Clear()
	env.Set("APP_PORT", "8080")
	env.Set("DB_PORT", "5432")

	type Service struct {
		Port int `env:"PORT"`
	}

	var app, db Service
	env.Unmarshal(&app, env.WithPrefix("APP"))
	env.Unmarshal(&db, env.WithPrefix("DB"))

	fmt.Println(app.Port, db.Port)
	// Output: 8080 5432
}

// UnmarshalMap decodes a struct from a map without touching the process
// environment - handy for tests and multi-tenant configuration.
func ExampleUnmarshalMap() {
	type Config struct {
		Host string `env:"HOST"`
		Port int    `env:"PORT"`
	}

	data := map[string]string{"HOST": "localhost", "PORT": "9000"}

	var c Config
	if err := env.UnmarshalMap(data, &c); err != nil {
		log.Fatal(err)
	}

	fmt.Println(c.Host, c.Port)
	// Output: localhost 9000
}

// MarshalMap converts a struct into a map without changing the environment.
func ExampleMarshalMap() {
	type Config struct {
		Host         string   `env:"HOST"`
		Port         int      `env:"PORT"`
		AllowedHosts []string `env:"ALLOWED_HOSTS" sep:":"`
	}

	c := Config{Host: "localhost", Port: 8080, AllowedHosts: []string{"a", "b"}}

	m, err := env.MarshalMap(c)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(m["HOST"])
	fmt.Println(m["PORT"])
	fmt.Println(m["ALLOWED_HOSTS"])
	// Output:
	// localhost
	// 8080
	// a:b
}

// Marshal writes a struct into the process environment.
func ExampleMarshal() {
	env.Clear()

	type Config struct {
		Host string `env:"HOST"`
		Port int    `env:"PORT"`
	}

	if err := env.Marshal(Config{Host: "localhost", Port: 8080}); err != nil {
		log.Fatal(err)
	}

	fmt.Println(env.Get("HOST"), env.Get("PORT"))
	// Output: localhost 8080
}

// Parse reads .env data from any io.Reader into a map. Double-quoted and
// unquoted values expand ${VAR}/$VAR; single quotes are literal.
func ExampleParse() {
	env.Clear()
	env.Set("USER", "goloop")

	r := strings.NewReader(`
# a comment
HOST=localhost
GREETING="hello ${USER}"
LITERAL='hello ${USER}'
`)

	m, err := env.Parse(r)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(m["HOST"])
	fmt.Println(m["GREETING"])
	fmt.Println(m["LITERAL"])
	// Output:
	// localhost
	// hello goloop
	// hello ${USER}
}

// The required flag makes a field mandatory.
func ExampleUnmarshal_required() {
	type Config struct {
		Token string `env:"TOKEN,required"`
	}

	var c Config
	err := env.UnmarshalMap(map[string]string{}, &c)
	fmt.Println(err)
	// Output: env: required key is not set: TOKEN
}

// time.Duration values are parsed with time.ParseDuration; time.Time uses the
// RFC3339 layout by default or a per-field layout tag.
func ExampleUnmarshal_time() {
	type Config struct {
		Timeout time.Duration `env:"TIMEOUT"`
		StartAt time.Time     `env:"START_AT" layout:"2006-01-02"`
	}

	data := map[string]string{"TIMEOUT": "1h30m", "START_AT": "2026-06-25"}

	var c Config
	if err := env.UnmarshalMap(data, &c); err != nil {
		log.Fatal(err)
	}

	fmt.Println(c.Timeout)
	fmt.Println(c.StartAt.Format("2006-01-02"))
	// Output:
	// 1h30m0s
	// 2026-06-25
}
