// Package env bridges .env files, the process environment and Go structures.
//
// It does three things:
//
//  1. Loads .env files into the process environment (a small Load/Overload
//     API: Load, Overload, LoadRaw, OverloadRaw, LoadReader, MustLoad).
//  2. Maps the environment to and from Go structs (an encoding/json-style API:
//     Unmarshal, Marshal and their Map/File/Reader/Writer/String variants) with
//     struct tags, defaults, validation and rich type support.
//  3. Parses .env data into plain maps without side effects (Read, Parse, All,
//     ReadSeq).
//
// # Loading
//
// Load and friends read one or more .env files (variadic; with no argument
// they default to ".env") into the process environment. Load keeps existing
// keys; Overload overwrites them. The Raw variants do not expand ${VAR}/$VAR.
// MustLoad is like Load but panics on error (handy in init or main).
//
//	if err := env.Load(".env"); err != nil {
//	    log.Fatal(err)
//	}
//
// # Decoding into a struct
//
// Unmarshal reads the process environment into a struct; UnmarshalMap,
// UnmarshalFile, UnmarshalReader and UnmarshalString read a map, a file, an
// io.Reader or a string directly without touching the environment.
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
// Marshal writes a struct into the environment; MarshalMap, MarshalFile,
// MarshalWriter and MarshalString produce a map, a file, an io.Writer or a
// string without changing the environment.
//
// The File/Reader/Writer/String encode and decode functions each have a Raw
// variant (e.g. UnmarshalFileRaw, MarshalStringRaw) that skips ${VAR}/$VAR
// expansion, so any value round-trips verbatim.
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
//   - WithFileMode sets the file permissions used by MarshalFile.
//   - WithParser/WithEncoder register a decoder/encoder for a custom type that
//     does not implement encoding.TextUnmarshaler/TextMarshaler.
//   - WithRequiredAll makes every leaf field required during decoding.
//
// # Struct tags
//
//   - env: the key name; "-" ignores the field; an inline "required" flag
//     (env:"KEY,required") makes it mandatory, and an inline "absolute" flag
//     (env:"DATABASE_URL,absolute") names the variable in full, ignoring the
//     prefix its enclosing structs would otherwise contribute - which is how
//     a nested struct reaches a name the deployment already fixed.
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
// The parser follows the de-facto .env format: single/double/backtick
// quotes, escape sequences in double quotes (\n, \t, \r, \\, \"), multi-line
// quoted values, full-line and inline comments, the optional export prefix and
// ${VAR}/$VAR expansion (in unquoted and double-quoted values only).
//
// A reference is only a reference when it names a key: VAR must match
// [A-Za-z_][A-Za-z0-9_]*, the same names this format can define. Anything else
// keeps its "$" as written, so a price ("cost: $100"), a one-liner
// ("$1 == x") and a password ("pa$$word") are values rather than
// substitutions. Single quotes and backticks remain the literal forms; there
// is no separate escape for "$".
//
// # Concurrency
//
// Loading and marshaling act on the global process environment. Beyond the
// guarantees of the standard os package there is no extra synchronization, so
// callers should not load and read the same keys concurrently. The map-, file-
// and reader/writer-based variants (Read, Parse, All, UnmarshalMap, MarshalMap,
// UnmarshalFile, MarshalFile, UnmarshalReader, MarshalWriter) have no global
// side effects.
//
// # Validating what was decoded
//
// The canonical shape of a config loader is three steps, and the third is
// yours:
//
//	func Load(files ...string) (*Config, error) {
//	    _ = env.Load(files...)
//	    var c Config
//	    if err := env.Unmarshal(&c); err != nil {
//	        return nil, err
//	    }
//	    if err := c.Validate(); err != nil {
//	        return nil, err
//	    }
//	    return &c, nil
//	}
//
// The "required" flag covers presence, which is the part a tag can express.
// Rules that involve more than one field cannot be: that two secrets must
// differ, that a limit must sit below a ceiling, that a key must be long
// enough to sign with. Those live in a Validate method the application writes
// and calls, and calling it at startup is what turns a misconfiguration into a
// process that refuses to start rather than one that fails on the first
// request that happens to need the value.
//
// Unmarshal deliberately does not call such a method for you, even when the
// target has one. The footgun is symmetrical - forgetting to write the method
// is exactly as easy as forgetting to call it - and an implicit call would
// have a decoder invoking application logic, which is a surprising thing for a
// decoder to do and a hard thing to find when it misbehaves.
//
// See DOC.md (English) and DOC.UK.md (Ukrainian) for the full reference.
package env
