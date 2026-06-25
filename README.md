[![Go Report Card](https://goreportcard.com/badge/github.com/goloop/env)](https://goreportcard.com/report/github.com/goloop/env) [![License](https://img.shields.io/badge/godoc-A+-brightgreen)](https://godoc.org/github.com/goloop/env) [![License](https://img.shields.io/badge/license-MIT-brightgreen)](https://github.com/goloop/env/blob/master/LICENSE) [![Stay with Ukraine](https://img.shields.io/static/v1?label=Stay%20with&message=Ukraine%20♥&color=ffD700&labelColor=0057B8&style=flat)](https://u24.gov.ua/)

# env

`env` is a configuration package for Go that bridges `.env` files, the process
environment and your Go structures.

It does three things, and does them well:

1. **Loads `.env` files into the process environment** — a familiar,
   `godotenv`-style API (`Load`, `Overload`, …).
2. **Maps the environment to and from Go structs** — a familiar,
   `encoding/json`-style API (`Unmarshal`, `Marshal`, …) with struct tags,
   defaults, validation and rich type support.
3. **Parses `.env` data into plain maps** without side effects (`Read`,
   `Parse`), so you can work with configuration from files, `embed.FS`, the
   network or a string.

The parser follows the de-facto `.env` specification (as used by Ruby
`dotenv`, `motdotla/dotenv` and `godotenv`): single/double/backtick quotes,
escape sequences, multi-line values, inline comments, variable expansion and
the `export` prefix.

## Features

- **Familiar API** — `Load`/`Overload` for files (like `godotenv`),
  `Marshal`/`Unmarshal` for structs (like `encoding/json`).
- **Three destinations** — load into the environment, a `map[string]string`,
  or a typed struct; serialize a struct back to the environment, a map or a
  file.
- **Rich types** — all integer and float sizes, `string`, `bool`, `url.URL`,
  `time.Duration`, `time.Time`, nested structs, pointers, slices and arrays.
- **Struct tags** — custom key names, default values, list separators, time
  layouts, an `-` to ignore a field and an inline `required` flag.
- **Spec-compliant parsing** — quotes, escape sequences, multi-line values,
  inline comments, `${VAR}`/`$VAR` expansion and `export`.
- **Side-effect-free variants** — `Read`/`Parse`/`MarshalMap`/`UnmarshalMap`
  never touch the global environment, which keeps tests and multi-tenant code
  clean.
- **Typed errors** — `errors.Is`-friendly sentinels (`ErrRequired`,
  `ErrNotPointer`, …).

## Installation

```sh
go get github.com/goloop/env
```

```go
import "github.com/goloop/env"
```

Requires Go 1.24 or newer. The package has no third-party dependencies.

## Quick start

Given a `.env` file:

```ini
# Server configuration.
HOST=0.0.0.0
PORT=8080
ALLOWED_HOSTS=localhost:127.0.0.1
REQUEST_TIMEOUT=30s
```

Describe it with a struct and load it:

```go
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/goloop/env"
)

type Config struct {
	Host         string        `env:"HOST"`
	Port         int           `env:"PORT" def:"80"`
	AllowedHosts []string      `env:"ALLOWED_HOSTS" sep:":"`
	Timeout      time.Duration `env:"REQUEST_TIMEOUT" def:"15s"`
}

func main() {
	// Load the .env file into the process environment.
	if err := env.Load(".env"); err != nil {
		log.Fatal(err)
	}

	// Map the environment into the struct.
	var cfg Config
	if err := env.Unmarshal(&cfg); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%+v\n", cfg)
	// {Host:0.0.0.0 Port:8080 AllowedHosts:[localhost 127.0.0.1] Timeout:30s}
}
```

Prefer to skip the global environment entirely (great for tests)? Read the
file into a struct directly:

```go
var cfg Config
if err := env.UnmarshalFile(".env", &cfg); err != nil {
	log.Fatal(err)
}
```

## A real-world example

A typical service reads several namespaced subsystems from one file. Reuse a
single component struct with `WithPrefix`:

```ini
APP_HOST=0.0.0.0
APP_PORT=8080

DB_HOST=db.internal
DB_PORT=5432
DB_DSN=postgres://user:pass@db.internal:5432/app

REDIS_HOST=cache.internal
REDIS_PORT=6379
```

```go
type Endpoint struct {
	Host string `env:"HOST"`
	Port int    `env:"PORT"`
}

type Config struct {
	App   Endpoint `env:"APP"`   // reads APP_HOST, APP_PORT
	DB    Endpoint `env:"DB"`    // reads DB_HOST, DB_PORT
	Redis Endpoint `env:"REDIS"` // reads REDIS_HOST, REDIS_PORT
	DSN   string   `env:"DB_DSN,required"`
}

func main() {
	if err := env.Load(".env"); err != nil {
		log.Fatal(err)
	}

	var cfg Config
	if err := env.Unmarshal(&cfg); err != nil {
		log.Fatal(err) // e.g. "env: required key is not set: DB_DSN"
	}
}
```

Nested structs are namespaced automatically (the tag becomes a prefix joined
with `_`). The same `Endpoint` type is reused three times, and the `required`
flag turns a forgotten `DB_DSN` into a clear error instead of a silent zero
value.

## The API at a glance

**Files / readers → process environment** (variadic, defaults to `.env`):

| Function | Expands `${VAR}` | Overwrites existing keys |
|----------|:----------------:|:------------------------:|
| `Load(files...)`         | yes | no  |
| `Overload(files...)`     | yes | yes |
| `LoadRaw(files...)`      | no  | no  |
| `OverloadRaw(files...)`  | no  | yes |
| `LoadReader(r)`          | yes | no  |

**Files / readers → map** (no side effects):

| Function | Expands `${VAR}` |
|----------|:----------------:|
| `Read(files...) (map, error)`  | yes |
| `ReadRaw(files...) (map, error)` | no |
| `Parse(r) (map, error)`        | yes |
| `ParseRaw(r) (map, error)`     | no  |

**Struct mapping:**

| Decode (→ struct) | Encode (struct →) |
|-------------------|-------------------|
| `Unmarshal(v, opts...)`        | `Marshal(v, opts...)` → environment |
| `UnmarshalMap(m, v, opts...)`  | `MarshalMap(v, opts...)` → map |
| `UnmarshalFile(name, v, opts...)` | `MarshalFile(name, v, opts...)` → file |

**Options:** `WithPrefix(p)`, `WithSeparator(sep)`, `WithTimeLayout(layout)`.

**Environment helpers** (thin wrappers over `os`): `Get`, `Set`, `Unset`,
`Clear`, `Environ`, `Expand`, `Lookup`, `Exists`.

> `Load` populates the real process environment (so child processes and
> libraries that read `os.Getenv` see the values). `UnmarshalFile` fills a
> struct without touching the environment. They complement each other — pick
> the one that matches your goal.

## Struct tags

```go
type Config struct {
	Host    string        `env:"HOST"`                       // key name
	Port    int           `env:"PORT" def:"8080"`            // default value
	Hosts   []string      `env:"HOSTS" sep:","`              // list separator
	Started time.Time      `env:"STARTED_AT" layout:"DateOnly"` // time layout
	Token   string        `env:"TOKEN,required"`             // must be present
	Ignored string        `env:"-"`                          // never mapped
}
```

| Tag | Purpose | Default |
|-----|---------|---------|
| `env` | key name; `-` ignores the field; `,required` marks it mandatory | field name |
| `def` | default value when the key is absent | zero value |
| `sep` | separator for slices/arrays | a single space |
| `layout` | layout for `time.Time` (Go layout or a constant name like `DateOnly`) | RFC3339 |

## Supported types

`string`, `bool`, every sized `int`/`uint`, `float32`/`float64`, `url.URL`,
`time.Duration` (`30s`, `1h30m`), `time.Time`, nested structs, pointers to any
of these, and slices/arrays of any of these.

## The `.env` format

```ini
# A full-line comment.
export HOST=localhost          # the optional `export` prefix is allowed
PORT=8080                      # an inline comment

EMPTY=                         # an empty value is valid -> ""
SPACED= trimmed                # surrounding spaces of unquoted values are trimmed

DOUBLE="expands ${HOST}"       # double quotes expand variables and escapes
SINGLE='literal ${HOST}'       # single quotes are literal
BACKTICK=`literal ${HOST}`     # backticks are literal
TABBED="a\tb"                  # \n \t \r \\ are interpreted in double quotes

MULTILINE="line one
line two"                      # quoted values may span several lines

LIST=a:b:c                     # split with a sep tag, e.g. sep:":"
```

Variable expansion (`${VAR}` / `$VAR`) is resolved against earlier keys in the
file and the existing environment. Single quotes and backticks are literal.

See **[DOC.md](DOC.md)** for the full reference, every function, more examples
and tips (Ukrainian: **[DOC.UK.md](DOC.UK.md)**).

## Documentation

- Full reference and recipes: [DOC.md](DOC.md) · [DOC.UK.md](DOC.UK.md)
- Package API: [pkg.go.dev/github.com/goloop/env](https://pkg.go.dev/github.com/goloop/env)

## Contributing

Bug reports and pull requests are welcome. Please run `go test ./...`,
`go vet ./...` and `gofmt -l .` before submitting.

## License

`env` is released under the MIT License. See [LICENSE](LICENSE).
