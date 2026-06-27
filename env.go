package env

import (
	"bytes"
	"io"
	"iter"
	"os"
	"regexp"
)

const (
	// The tagNameKey the identifier of the tag that sets the key name.
	tagNameKey = "env"

	// The tagNameValue the identifier of the tag that sets the default value.
	tagNameValue = "def"

	// The tagNameSep the identifier of the tag that sets the separator
	// of the items in the string of value.
	tagNameSep = "sep"

	// The tagNameLayout the identifier of the tag that sets the layout
	// for time.Time fields.
	tagNameLayout = "layout"

	// The tagFlagRequired is the inline env-tag flag that marks a field
	// as required, e.g. `env:"KEY,required"`.
	tagFlagRequired = "required"

	// The defValueSep is the default separator of the items in the string of
	// value. Comma is the conventional list separator and avoids the data
	// loss a space default causes for values that contain spaces. Override it
	// per field with the sep tag, or per call with WithSeparator.
	defValueSep = ","

	// The defValueIgnored is the value of the tagNameKey field that
	// should be ignored during processing.
	defValueIgnored = "-"
)

var (
	// The validKeyRgx is a regular expression to validate the key name.
	validKeyRgx = regexp.MustCompile(`^[A-Za-z_]{1}\w*$`)

	// The emptyLineRgx is a regular expression to check if a string
	// is empty (consists of characters that cannot be values).
	emptyLineRgx = regexp.MustCompile(`^(\s*)$|^(\s*[#].*)$`)

	// The keyRgx is a regular expression to check
	// the string which can be a key.
	keyRgx = regexp.MustCompile(
		`^(?:\s*)?(?:export\s+)?(?P<key>[a-zA-Z_][a-zA-Z_0-9]*)=`,
	)
)

// Load reads the given .env files into the process environment, keeping any
// keys that are already set. Variables like ${VAR} or $VAR are expanded. With
// no arguments it loads ".env" from the current directory; with several files
// the first value set for a key wins.
//
//	if err := env.Load(".env"); err != nil {
//		log.Fatal(err)
//	}
func Load(filenames ...string) error {
	return loadFiles(filenames, true, false)
}

// Overload is like Load but overwrites keys that already exist in the
// environment.
func Overload(filenames ...string) error {
	return loadFiles(filenames, true, true)
}

// LoadRaw is like Load but does not expand ${VAR}/$VAR: values are stored
// verbatim.
func LoadRaw(filenames ...string) error {
	return loadFiles(filenames, false, false)
}

// OverloadRaw is like Overload but does not expand ${VAR}/$VAR.
func OverloadRaw(filenames ...string) error {
	return loadFiles(filenames, false, true)
}

// loadFiles loads each file into the environment. An empty list defaults to
// ".env".
func loadFiles(filenames []string, expand, update bool) error {
	if len(filenames) == 0 {
		filenames = []string{".env"}
	}

	for _, filename := range filenames {
		if err := readParseStore(filename, expand, update, false); err != nil {
			return err
		}
	}

	return nil
}

// MustLoad is like Load but panics if loading fails. It is convenient in init
// or main, where a missing or invalid configuration should stop the program.
//
//	func init() { env.MustLoad(".env") }
func MustLoad(filenames ...string) {
	if err := Load(filenames...); err != nil {
		panic(err)
	}
}

// LoadReader reads .env data from r into the process environment, keeping
// existing keys and expanding ${VAR}/$VAR. It is handy for embedded files
// (embed.FS), network sources or strings.
func LoadReader(r io.Reader) error {
	m, err := parse(r, true)
	if err != nil {
		return err
	}

	for key, value := range m {
		if _, ok := os.LookupEnv(key); !ok {
			if err := Set(key, value); err != nil {
				return err
			}
		}
	}

	return nil
}

// Read parses the given .env files into a map without touching the process
// environment, expanding ${VAR}/$VAR. With no arguments it reads ".env"; with
// several files the first value set for a key wins.
func Read(filenames ...string) (map[string]string, error) {
	return readFiles(filenames, true)
}

// ReadRaw is like Read but does not expand ${VAR}/$VAR.
func ReadRaw(filenames ...string) (map[string]string, error) {
	return readFiles(filenames, false)
}

// readFiles parses each file into a single map (first value for a key wins).
func readFiles(filenames []string, expand bool) (map[string]string, error) {
	if len(filenames) == 0 {
		filenames = []string{".env"}
	}

	result := make(map[string]string)
	for _, filename := range filenames {
		file, err := os.Open(filename)
		if err != nil {
			return nil, err
		}

		m, err := parse(file, expand)
		file.Close()
		if err != nil {
			return nil, err
		}

		for key, value := range m {
			if _, ok := result[key]; !ok {
				result[key] = value
			}
		}
	}

	return result, nil
}

// Parse reads .env data from r into a map without touching the process
// environment, expanding ${VAR}/$VAR.
func Parse(r io.Reader) (map[string]string, error) {
	return parse(r, true)
}

// ParseRaw is like Parse but does not expand ${VAR}/$VAR.
func ParseRaw(r io.Reader) (map[string]string, error) {
	return parse(r, false)
}

// All returns an iterator over the key/value pairs parsed from the given .env
// files (default ".env"), expanding ${VAR}/$VAR, without touching the process
// environment. It is a convenience over Read for ranging without building a
// map yourself:
//
//	for key, value := range env.All(".env") {
//	    fmt.Println(key, value)
//	}
//
// Read or parse errors are ignored (the iterator yields nothing); use Read if
// you need to handle them.
func All(filenames ...string) iter.Seq2[string, string] {
	return func(yield func(string, string) bool) {
		m, err := readFiles(filenames, true)
		if err != nil {
			return
		}
		for key, value := range m {
			if !yield(key, value) {
				return
			}
		}
	}
}

// ReadSeq is the error-aware counterpart of All: it reads the given .env files
// (default ".env"), expanding ${VAR}/$VAR, and returns an iterator over their
// key/value pairs, or an error if reading or parsing fails. Use it instead of
// All when you need to handle failures.
//
//	seq, err := env.ReadSeq(".env")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for key, value := range seq {
//	    fmt.Println(key, value)
//	}
func ReadSeq(filenames ...string) (iter.Seq2[string, string], error) {
	m, err := readFiles(filenames, true)
	if err != nil {
		return nil, err
	}

	return func(yield func(string, string) bool) {
		for key, value := range m {
			if !yield(key, value) {
				return
			}
		}
	}, nil
}

// MarshalFile writes the struct obj into the file as KEY=value lines without
// changing the process environment.
//
// Use options to control the output, e.g. WithPrefix, WithSeparator or
// WithFileMode. See Marshal for the list of supported field types and tags.
//
//	var config = Config{Host: "localhost", Port: 8080}
//	env.MarshalFile("/tmp/.env", config)
func MarshalFile(filename string, obj any, opts ...Option) error {
	st := newSettings(opts...)
	b, err := encodeLines(obj, st)
	if err != nil {
		return err
	}

	return os.WriteFile(filename, b, st.fileMode)
}

// MarshalWriter writes the struct obj as KEY=value lines to w, the symmetric
// counterpart of LoadReader, without changing the process environment. It is
// handy for logs, network connections or buffers. See Marshal for the supported
// field types and tags.
//
//	var buf bytes.Buffer
//	env.MarshalWriter(&buf, config)
func MarshalWriter(w io.Writer, obj any, opts ...Option) error {
	b, err := encodeLines(obj, newSettings(opts...))
	if err != nil {
		return err
	}

	_, err = w.Write(b)
	return err
}

// The encodeLines encodes obj into the KEY=value text body, quoting values that
// need it so the result round-trips through the parser.
func encodeLines(obj any, st settings) ([]byte, error) {
	pairs, err := encodeStruct(obj, st)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	for _, p := range pairs {
		buf.WriteString(p.key)
		buf.WriteByte('=')
		buf.WriteString(quoteEnvValue(p.value))
		buf.WriteByte('\n')
	}

	return buf.Bytes(), nil
}

// MarshalMap converts the struct obj into a map of key/value pairs without
// changing the process environment. It is the pure counterpart of Marshal and
// pairs with UnmarshalMap for round-tripping.
func MarshalMap(obj any, opts ...Option) (map[string]string, error) {
	pairs, err := encodeStruct(obj, newSettings(opts...))
	if err != nil {
		return nil, err
	}

	m := make(map[string]string, len(pairs))
	for _, p := range pairs {
		m[p.key] = p.value
	}

	return m, nil
}

// Exists returns true if all given keys exists in the environment.
//
// # Examples
//
// In this example, some variables are already set in the environment:
//
//	$ env | grep KEY_0
//	KEY_0=VALUE_DEF
//
// Configuration file `.env` contains:
//
//	LAST_ID=002
//	KEY_0=VALUE_000
//	KEY_1=VALUE_001
//	KEY_2=VALUE_${LAST_ID}
//
// Check if a variable exists in the environment:
//
//	fmt.Printf("KEY_0 is %v\n", env.Exists("KEY_0"))
//	fmt.Printf("KEY_1 is %v\n", env.Exists("KEY_1"))
//	fmt.Printf("KEY_0 and KEY_1 is %v\n\n", env.Exists("KEY_0", "KEY_1"))
//
//	if err := env.Load(".env"); err != nil {
//	  log.Fatal(err)
//	}
//
//	fmt.Printf("KEY_0 is %v\n", env.Exists("KEY_0"))
//	fmt.Printf("KEY_1 is %v\n", env.Exists("KEY_1"))
//	fmt.Printf("KEY_0 and KEY_1 is %v\n", env.Exists("KEY_0", "KEY_1"))
//
//	// Output:
//	//  KEY_0 is true
//	//  KEY_1 is false
//	//  KEY_0 and KEY_1 is false
//	//
//	//  KEY_0 is true
//	//  KEY_1 is true
//	//  KEY_0 and KEY_1 is true
func Exists(keys ...string) bool {
	for _, key := range keys {
		if _, ok := os.LookupEnv(key); !ok {
			return false
		}
	}
	return true
}

// Unmarshal reads values from the process environment into the struct pointed
// to by obj. By default keys match field names (or the env tag); use
// WithPrefix to read a namespaced subset (WithPrefix("APP") reads APP_*).
//
// Supported field types: all int/uint sizes, float32/64, string, bool,
// url.URL, time.Duration, time.Time, any type implementing
// encoding.TextUnmarshaler (e.g. net.IP, custom enums), nested structs,
// pointers, and arrays/slices of those. Tags: env (key name, "-" to ignore,
// ",required" to require), def (default value), sep (list separator), layout
// (time format).
//
// If obj implements Unmarshaler, its UnmarshalEnv method is called with the
// resolved key/value map instead of the reflective decoding.
//
//	var config Config
//	if err := env.Unmarshal(&config); err != nil {
//		log.Fatal(err)
//	}
func Unmarshal(obj any, opts ...Option) error {
	return decodeStruct(environMap(), obj, newSettings(opts...))
}

// UnmarshalMap reads values from the given map m into the struct pointed to by
// obj, without touching the process environment. It is the pure counterpart of
// Unmarshal, useful for tests and multi-tenant configuration.
func UnmarshalMap(m map[string]string, obj any, opts ...Option) error {
	return decodeStruct(m, obj, newSettings(opts...))
}

// UnmarshalFile reads the .env file into the struct pointed to by obj without
// changing the process environment, expanding ${VAR}/$VAR. For the raw form
// (no expansion) use ParseRaw together with UnmarshalMap.
func UnmarshalFile(filename string, obj any, opts ...Option) error {
	m, err := readFiles([]string{filename}, true)
	if err != nil {
		return err
	}

	return decodeStruct(m, obj, newSettings(opts...))
}

// UnmarshalReader reads .env data from r and decodes it into the struct pointed
// to by obj, expanding ${VAR}/$VAR, without touching the process environment.
// It is the symmetric counterpart of LoadReader and is handy for embedded files
// (embed.FS), the network or a string. For the raw form (no expansion) use
// ParseRaw together with UnmarshalMap.
func UnmarshalReader(r io.Reader, obj any, opts ...Option) error {
	m, err := parse(r, true)
	if err != nil {
		return err
	}

	return decodeStruct(m, obj, newSettings(opts...))
}

// Marshal writes the fields of the struct obj into the process environment,
// overwriting existing keys. Use options such as WithPrefix or WithSeparator
// to control the output.
//
// Supported field types: all int/uint sizes, float32/64, string, bool,
// url.URL, time.Duration, time.Time, any type implementing
// encoding.TextMarshaler (e.g. net.IP, custom enums), nested structs,
// pointers, and arrays/slices of those. Tags: env (key name, "-" to ignore),
// def (default value), sep (list separator), layout (time format).
//
// If obj implements Marshaler, its MarshalEnv method provides the key/value
// pairs that are written. To produce the pairs without changing the
// environment use MarshalMap; to write them to a file use MarshalFile.
//
//	var config = Config{Host: "localhost", Port: 8080}
//	if err := env.Marshal(config); err != nil {
//		log.Fatal(err)
//	}
func Marshal(obj any, opts ...Option) error {
	pairs, err := encodeStruct(obj, newSettings(opts...))
	if err != nil {
		return err
	}

	for _, p := range pairs {
		if err := Set(p.key, p.value); err != nil {
			return err
		}
	}

	return nil
}
