# env — reference

Complete reference for `github.com/goloop/env`. For a quick overview see the
[README](README.md). Ukrainian version: [DOC.UK.md](DOC.UK.md).

## Contents

- [Mental model](#mental-model)
- [Loading files into the environment](#loading-files-into-the-environment)
- [Parsing into a map](#parsing-into-a-map)
- [Decoding into a struct](#decoding-into-a-struct)
- [Encoding a struct](#encoding-a-struct)
- [Options](#options)
- [Struct tags](#struct-tags)
- [Supported types](#supported-types)
- [Custom marshaling](#custom-marshaling)
- [Environment helpers](#environment-helpers)
- [Errors](#errors)
- [The .env file format](#the-env-file-format)
- [Recipes and tips](#recipes-and-tips)

## Mental model

The package moves configuration between three places:

```
.env file / io.Reader  ──►  process environment (os.Environ)  ──►  Go struct
       │                              ▲                               ▲
       └──────────────►  map[string]string  ──────────────────────────┘
```

- **Loading** (`Load`, `Overload`, …) writes file data into the **process
  environment**. Values become strings visible to `os.Getenv`, child processes
  and other libraries.
- **Parsing** (`Read`, `Parse`) turns file/reader data into a plain
  **`map[string]string`** without any side effects.
- **Decoding** (`Unmarshal`, `UnmarshalMap`, `UnmarshalFile`) fills a **typed
  struct** from the environment, a map or a file.
- **Encoding** (`Marshal`, `MarshalMap`, `MarshalFile`) serializes a struct to
  the environment, a map or a file.

A key rule: the bare `Marshal`/`Unmarshal` operate on the global process
environment; the `*Map` and `*File` variants do not. Reach for the pure
variants in tests and concurrent or multi-tenant code.

## Loading files into the environment

These functions read one or more `.env` files (variadic; with no argument they
default to `.env`) and write the result into the process environment. When
several files are given, the **first value set for a key wins**.

| Function | Expansion | Existing keys |
|----------|-----------|---------------|
| `Load(filenames ...string) error`        | `${VAR}` expanded | kept |
| `Overload(filenames ...string) error`    | `${VAR}` expanded | overwritten |
| `LoadRaw(filenames ...string) error`     | literal | kept |
| `OverloadRaw(filenames ...string) error` | literal | overwritten |

```go
// Load .env, keeping anything already set in the environment.
if err := env.Load(".env"); err != nil {
	log.Fatal(err)
}

// Layer files: base first, then environment-specific overrides.
if err := env.Overload(".env", ".env.local"); err != nil {
	log.Fatal(err)
}
```

`Raw` variants skip `${VAR}`/`$VAR` expansion and store values verbatim. Use
them when your values legitimately contain `$` and must not be interpolated.

### LoadReader

```go
func LoadReader(r io.Reader) error
```

Loads `.env` data from any reader into the environment (expansion on, existing
keys kept). Handy for embedded files, the network or a string.

```go
//go:embed config.env
var configFS embed.FS

f, _ := configFS.Open("config.env")
defer f.Close()
if err := env.LoadReader(f); err != nil {
	log.Fatal(err)
}
```

## Parsing into a map

These functions return a `map[string]string` and never touch the environment.

| Function | Source | Expansion |
|----------|--------|-----------|
| `Read(filenames ...string) (map[string]string, error)`    | files  | yes |
| `ReadRaw(filenames ...string) (map[string]string, error)` | files  | no  |
| `Parse(r io.Reader) (map[string]string, error)`           | reader | yes |
| `ParseRaw(r io.Reader) (map[string]string, error)`        | reader | no  |

```go
m, err := env.Read(".env")
if err != nil {
	log.Fatal(err)
}
fmt.Println(m["HOST"])

// Parse from a string.
m, _ = env.Parse(strings.NewReader("HOST=localhost\nPORT=8080\n"))
```

Expansion in `Read`/`Parse` resolves `${VAR}` against earlier keys in the same
source and, as a fallback, the current process environment — without writing
anything back.

## Decoding into a struct

```go
func Unmarshal(v any, opts ...Option) error                          // from os.Environ
func UnmarshalMap(m map[string]string, v any, opts ...Option) error  // from a map
func UnmarshalFile(filename string, v any, opts ...Option) error     // from a file
```

`v` must be a non-nil pointer to a struct. Fields are matched by the `env` tag
(or the field name when the tag is empty). `UnmarshalMap` and `UnmarshalFile`
do not touch the environment.

```go
type Config struct {
	Host string `env:"HOST"`
	Port int    `env:"PORT" def:"80"`
}

// From the environment.
var c Config
if err := env.Unmarshal(&c); err != nil {
	log.Fatal(err)
}

// From a map (no environment involved).
_ = env.UnmarshalMap(map[string]string{"HOST": "localhost", "PORT": "9000"}, &c)

// From a file directly (parses the file, no environment involved).
_ = env.UnmarshalFile(".env", &c)
```

`Unmarshal(&cfg)` and `UnmarshalFile(".env", &cfg)` look similar but differ in
one important way: `Unmarshal` reads the **process environment** (so you must
`Load` the file first if you want the file's values there), while
`UnmarshalFile` reads the **file directly** and leaves the environment
untouched.

## Encoding a struct

```go
func Marshal(v any, opts ...Option) error                       // into os.Environ
func MarshalMap(v any, opts ...Option) (map[string]string, error) // into a map
func MarshalFile(filename string, v any, opts ...Option) error  // into a file
```

`Marshal` writes each field into the process environment (overwriting). The
`*Map` and `*File` variants do not change the environment. `MarshalMap` pairs
with `UnmarshalMap` for round-tripping.

```go
type Config struct {
	Host  string   `env:"HOST"`
	Port  int      `env:"PORT"`
	Hosts []string `env:"HOSTS" sep:":"`
}
cfg := Config{Host: "localhost", Port: 8080, Hosts: []string{"a", "b"}}

env.Marshal(cfg)                 // HOST/PORT/HOSTS set in the environment
m, _ := env.MarshalMap(cfg)      // m["HOSTS"] == "a:b"
_ = env.MarshalFile(".env", cfg) // writes KEY=value lines to .env
```

## Options

Options set **call-level defaults** that a per-field tag can override. The
precedence is always: **field tag > option > built-in default**.

```go
func WithPrefix(prefix string) Option
func WithSeparator(sep string) Option
func WithTimeLayout(layout string) Option
```

### WithPrefix

Names a key namespace. A prefix is a level; levels are joined with `_`. A
trailing `_` is added automatically when missing, and an empty prefix adds no
leading `_`. `WithPrefix("APP")` and `WithPrefix("APP_")` are equivalent.

```go
type Service struct {
	Port int `env:"PORT"`
}
var app, db Service
env.Unmarshal(&app, env.WithPrefix("APP")) // reads APP_PORT
env.Unmarshal(&db, env.WithPrefix("DB"))   // reads DB_PORT
```

> Tip: for fixed namespaces, prefer **nested structs** — they read the same
> values in a single call:
>
> ```go
> type Config struct {
>     App Service `env:"APP"` // reads APP_PORT
>     DB  Service `env:"DB"`  // reads DB_PORT
> }
> env.Unmarshal(&cfg)
> ```
>
> Reserve `WithPrefix` for runtime/dynamic prefixes (e.g. multi-tenant).

### WithSeparator

Sets the default separator for slice/array fields without a `sep` tag. The
built-in default is a single space.

```go
type Config struct {
	Hosts []string `env:"HOSTS"` // no sep tag -> uses the option
}
var c Config
env.UnmarshalMap(map[string]string{"HOSTS": "a,b,c"}, &c, env.WithSeparator(","))
```

### WithTimeLayout

Sets the default layout for `time.Time` fields without a `layout` tag. Accepts
a Go reference-time layout or a standard constant name. The built-in default
is RFC3339.

```go
env.Unmarshal(&c, env.WithTimeLayout("DateOnly"))
```

## Struct tags

```go
type Config struct {
	Host    string        `env:"HOST"`
	Port    int           `env:"PORT" def:"8080"`
	Hosts   []string      `env:"HOSTS" sep:","`
	Started time.Time      `env:"STARTED_AT" layout:"2006-01-02"`
	Token   string        `env:"TOKEN,required"`
	Secret  string        `env:"-"`
}
```

| Tag | Description |
|-----|-------------|
| `env` | The key name. `-` ignores the field entirely. Inline flags follow the name after a comma, e.g. `env:"KEY,required"`. |
| `def` | Default value used when the key is absent from the source. |
| `sep` | Separator for slice/array values (overrides `WithSeparator`). |
| `layout` | Layout for `time.Time` (overrides `WithTimeLayout`). A Go layout or a constant name such as `RFC3339`, `RFC1123`, `DateTime`, `DateOnly`, `TimeOnly`, `Kitchen`, `ANSIC`, `UnixDate`, `Stamp`. |

### required

`env:"KEY,required"` makes a field mandatory. If the key is absent from the
source **and** no `def` is provided, decoding returns an error that wraps
`ErrRequired`:

```go
type Config struct {
	Token string `env:"TOKEN,required"`
}
err := env.UnmarshalMap(map[string]string{}, &Config{})
// err: "env: required key is not set: TOKEN"
errors.Is(err, env.ErrRequired) // true
```

A `def` satisfies the requirement (the default is a deliberate handling of the
missing case), so `required` together with `def` never errors.

### Ignoring a field

`env:"-"` skips the field on both decode and encode — useful for computed or
secret fields you never want mapped.

## Supported types

| Category | Types |
|----------|-------|
| Strings  | `string` |
| Booleans | `bool` (`true`/`false`, also `1`/`0`, `t`/`f`, … per `strconv.ParseBool`) |
| Integers | `int`, `int8`, `int16`, `int32`, `int64` (with range checks) |
| Unsigned | `uint`, `uint8`, `uint16`, `uint32`, `uint64` (with range checks) |
| Floats   | `float32`, `float64` (shortest round-tripping representation) |
| URLs     | `url.URL` |
| Time     | `time.Duration` (`30s`, `1h30m`), `time.Time` (layout-driven) |
| Compound | nested structs, pointers, slices and arrays of the above |

```go
type Config struct {
	Debug    bool          `env:"DEBUG"`
	Workers  uint8         `env:"WORKERS" def:"4"`
	Ratio    float64       `env:"RATIO"`
	Endpoint url.URL       `env:"ENDPOINT"`
	Timeout  time.Duration `env:"TIMEOUT" def:"30s"`
	StartAt  time.Time     `env:"START_AT"` // RFC3339 by default
	Ports    []int         `env:"PORTS" sep:","`
	Limits   [3]int        `env:"LIMITS" sep:":"` // exact length enforced
}
```

Arrays enforce their length: decoding more elements than the array can hold is
an error. Decoding an empty value yields an empty slice (and leaves an array at
its zero values).

### Optional fields (pointers)

A pointer field models an *optional* value: a nil pointer means "absent". The
package handles absence consistently in both directions, so optional values
round-trip:

- **Decode** allocates a pointer only when there is a value to assign — the key
  is present (even if empty), or a `def` is set. If the key is absent and there
  is no default, the pointer stays `nil`.
- **Encode** omits a nil pointer (no key is written).
- A **nil element of a pointer slice** is positional: it is written as an empty
  value at its position (`[]*int{a, nil, b}` → `"1,,3"`).
- For a **nil pointer to a nested struct**, decoding allocates it only when the
  source has at least one key under its prefix; otherwise it stays `nil`.

```go
type Config struct {
	Port *int `env:"PORT"` // nil when PORT is unset, *value when it is
}
```

Why this design:

- **No `null` in `.env`.** Unlike JSON, a `.env` file has no null literal — a
  key is either set to a string or absent. The faithful representation of
  "unset" is therefore an absent key, which is exactly what encode produces and
  decode consumes. As a result `MarshalMap` → `UnmarshalMap` returns nil
  pointers back to `nil`.
- **It mirrors `encoding/json`.** `json.Unmarshal` allocates a pointer only when
  the key is present and leaves it nil otherwise — the same rule here.
- **It preserves optionality.** The whole point of a pointer in a config struct
  is to distinguish "not set" (`nil`) from "set to the zero value" (a pointer to
  `0`/`""`). Always allocating would erase that distinction.

## Custom marshaling

Implement these interfaces to take full control of a type, exactly like
`encoding/json`:

```go
type Marshaler interface {
	MarshalEnv() (map[string]string, error)
}

type Unmarshaler interface {
	UnmarshalEnv(data map[string]string) error
}
```

`MarshalEnv` returns the key/value pairs; the library decides where they go
(environment, map or file). `UnmarshalEnv` receives the already-resolved
(expanded) source map and fills the value itself — the reflective tag handling
is skipped entirely.

```go
type Config struct {
	Host string
	Port int
}

func (c *Config) UnmarshalEnv(data map[string]string) error {
	c.Host = data["HOST"]
	c.Port, _ = strconv.Atoi(data["PORT"])
	return nil
}

func (c Config) MarshalEnv() (map[string]string, error) {
	return map[string]string{
		"HOST": c.Host,
		"PORT": strconv.Itoa(c.Port),
	}, nil
}
```

## Environment helpers

Thin, dependency-free wrappers over the standard `os` package:

| Function | Equivalent |
|----------|------------|
| `Get(key) string`            | `os.Getenv` |
| `Set(key, value) error`      | `os.Setenv` |
| `Unset(key) error`           | `os.Unsetenv` |
| `Clear()`                    | `os.Clearenv` |
| `Environ() []string`         | `os.Environ` |
| `Expand(value) string`       | `os.Expand` with `os.Getenv` |
| `Lookup(key) (string, bool)` | `os.LookupEnv` |
| `Exists(keys ...string) bool`| true if every key is set |

## Errors

Validation failures are typed sentinels, testable with `errors.Is`:

| Error | Meaning |
|-------|---------|
| `ErrNilObject`    | the object passed to `Unmarshal`/`Marshal` is `nil` |
| `ErrNotPointer`   | the object is not a non-nil pointer to a struct |
| `ErrNotStruct`    | the object does not point to a struct |
| `ErrEmptyStruct`  | the struct has no fields |
| `ErrInvalidObject`| `Marshal` got something that is not a struct or pointer to one |
| `ErrRequired`     | a `required` field has no value and no default |

```go
if err := env.Unmarshal(&cfg); errors.Is(err, env.ErrRequired) {
	// a mandatory key was missing
}
```

Conversion errors from `strconv` and `time` are returned as-is, so
`errors.Is(err, strconv.ErrSyntax)` and friends also work.

## The .env file format

The parser follows the de-facto `.env` specification.

### Keys

A key must match `^[A-Za-z_][A-Za-z0-9_]*$` (POSIX environment variable names):
it starts with a letter or underscore and contains letters, digits and
underscores. The optional `export` prefix is accepted: `export KEY=value`.

### Values and whitespace

```ini
KEY=value          # a basic value
EMPTY=             # an empty value is valid -> ""
SPACED=  trimmed   # surrounding whitespace of unquoted values is trimmed
```

### Quotes

```ini
DOUBLE="value"     # variables and escapes are processed
SINGLE='value'     # literal: no expansion, no escapes
BACKTICK=`value`   # literal, may contain ' and " inside
```

Quoted values keep their inner whitespace. Single quotes and backticks are
fully literal, so `'$USER'` stays `$USER`.

### Escape sequences (double quotes only)

`\n`, `\t`, `\r`, `\\` and `\"` are interpreted inside double quotes. Single
quotes and backticks keep backslashes verbatim.

```ini
MESSAGE="line one\nline two"
```

### Multi-line values

A quoted value may span several physical lines:

```ini
KEY="line one
line two
line three"
```

### Comments

```ini
# A full-line comment.
KEY=value   # an inline comment
NOTE="a # inside quotes is part of the value"
```

A `#` starts a comment only outside quotes. If your value contains `#`, quote
it.

### Variable expansion

`${VAR}` and `$VAR` are expanded in unquoted and double-quoted values, resolved
against earlier keys in the same source and the existing environment. Single
quotes and backticks are literal.

```ini
USER=goloop
EMAIL="${USER}@example.com"   # -> goloop@example.com
LITERAL='${USER}'             # -> ${USER}
```

The `Raw`/`ParseRaw` variants disable expansion entirely.

## Recipes and tips

**Fail fast on incomplete configuration.** Mark mandatory keys `required` and
check the error once:

```go
if err := env.Unmarshal(&cfg); err != nil {
	log.Fatalf("config: %v", err)
}
```

**Keep tests clean.** Use the pure variants so tests never mutate global state:

```go
m := map[string]string{"HOST": "localhost", "PORT": "8080"}
var cfg Config
env.UnmarshalMap(m, &cfg)
```

**Layer configuration.** Load a base file, then override with a local one:

```go
env.Load(".env")            // base, does not override the environment
env.Overload(".env.local")  // local overrides
```

**Round-trip.** `MarshalMap` and `UnmarshalMap` are inverses:

```go
m, _ := env.MarshalMap(cfg)
var back Config
env.UnmarshalMap(m, &back) // back == cfg
```

**Embedded defaults.** Ship a default `.env` with `embed.FS` and load it before
the real environment:

```go
//go:embed defaults.env
var defaults embed.FS
f, _ := defaults.Open("defaults.env")
env.LoadReader(f) // does not override anything already set
```
