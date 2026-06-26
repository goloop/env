// Package env bridges .env files, the process environment and Go structures.
//
// It does three things:
//
//  1. Loads .env files into the process environment (a small Load/Overload
//     API: Load, Overload, LoadRaw, OverloadRaw, LoadReader).
//  2. Maps the environment to and from Go structs (an encoding/json-style API:
//     Unmarshal, Marshal and their Map/File variants) with struct tags,
//     defaults, validation and rich type support.
//  3. Parses .env data into plain maps without side effects (Read, Parse).
//
// # Loading
//
// Load and friends read one or more .env files (variadic; with no argument
// they default to ".env") into the process environment. Load keeps existing
// keys; Overload overwrites them. The Raw variants do not expand ${VAR}/$VAR.
//
//	if err := env.Load(".env"); err != nil {
//	    log.Fatal(err)
//	}
//
// # Decoding into a struct
//
// Unmarshal reads the process environment into a struct; UnmarshalMap and
// UnmarshalFile read a map or a file directly without touching the
// environment.
//
//	type Config struct {
//	    Host    string        `env:"HOST"`
//	    Port    int           `env:"PORT" def:"80"`
//	    Hosts   []string      `env:"ALLOWED_HOSTS" sep:":"`
//	    Timeout time.Duration `env:"TIMEOUT" def:"30s"`
//	}
//
//	var cfg Config
//	if err := env.Unmarshal(&cfg); err != nil {
//	    log.Fatal(err)
//	}
//
// # Encoding a struct
//
// Marshal writes a struct into the environment; MarshalMap and MarshalFile
// produce a map or a file without changing the environment.
//
// # Options
//
// Options set call-level defaults that a per-field tag can override
// (precedence: field tag > option > built-in default):
//
//   - WithPrefix sets a key namespace; levels are joined with "_", so
//     WithPrefix("APP") maps PORT to APP_PORT.
//   - WithSeparator sets the default list separator.
//   - WithTimeLayout sets the default time.Time layout.
//
// # Struct tags
//
//   - env: the key name; "-" ignores the field; an inline "required" flag
//     (env:"KEY,required") makes it mandatory.
//   - def: a default value used when the key is absent.
//   - sep: the separator for slice/array values (default: a comma).
//   - layout: the layout for time.Time fields (default: RFC3339).
//
// # Supported types
//
// All sized int/uint, float32/64, string, bool, url.URL, time.Duration,
// time.Time, any type implementing encoding.TextMarshaler/TextUnmarshaler
// (e.g. net.IP, custom enums), nested structs, pointers and slices/arrays of
// these.
//
// A pointer field is optional: it is decoded as nil when its key is absent and
// omitted on encode, so optional values round-trip (see DOC.md for details).
//
// # Custom marshaling
//
// Types implementing Marshaler or Unmarshaler take full control, mirroring
// encoding/json: MarshalEnv returns a map of key/value pairs and UnmarshalEnv
// receives the resolved source map.
//
// # The .env format
//
// The parser follows the de-facto .env specification: single/double/backtick
// quotes, escape sequences in double quotes (\n, \t, \r, \\, \"), multi-line
// quoted values, full-line and inline comments, the optional export prefix and
// ${VAR}/$VAR expansion (in unquoted and double-quoted values only).
//
// # Concurrency
//
// Loading and marshaling act on the global process environment. Beyond the
// guarantees of the standard os package there is no extra synchronization, so
// callers should not load and read the same keys concurrently. The map- and
// file-based variants (Read, Parse, UnmarshalMap, MarshalMap, UnmarshalFile,
// MarshalFile) have no global side effects.
//
// See DOC.md (English) and DOC.UK.md (Ukrainian) for the full reference.
package env
