package env

import (
	"bytes"
	"os"
	"regexp"
	"runtime"
)

const (
	// The tagNameKey the identifier of the tag that sets the key name.
	tagNameKey = "env"

	// The tagNameValue the identifier of the tag that sets the default value.
	tagNameValue = "def"

	// The tagNameSep the identifier of the tag that sets the separator
	// of the items in the string of value.
	tagNameSep = "sep"

	// The defValueSep is the default separator of the items
	// in the string of value.
	defValueSep = " "

	// The defValueIgnored is the value of the tagNameKey field that
	// should be ignored during processing.
	defValueIgnored = "-"
)

var (
	// The parallelTasks the number of parallel transliteration tasks.
	// By default, the number of threads is set as the number of CPU cores.
	parallelTasks = 1

	// The validKeyRgx is a regular expression to validate the key name.
	validKeyRgx = regexp.MustCompile(`^[A-Za-z_]{1}\w*$`)

	// The emptyLineRgx is a regular expression to check if a string
	// is empty (consists of characters that cannot be values).
	emptyLineRgx = regexp.MustCompile(`^(\s*)$|^(\s*[#].*)$`)

	// The valueRgx is a regular expression to check
	// the string which can be a value.
	valueRgx = regexp.MustCompile(`^=[^\s].*`)

	// The keyRgx is a regular expression to check
	// the string which can be a key.
	keyRgx = regexp.MustCompile(
		`^(?:\s*)?(?:export\s+)?(?P<key>[a-zA-Z_][a-zA-Z_0-9]*)=`,
	)
)

// Initializer.
func init() {
	// Set the number of parallel parsing tasks.
	ParallelTasks(runtime.NumCPU())
}

// Together sets the number of parallel transliteration tasks.
func ParallelTasks(pt int) int {
	// The maximum number of parallel tasks
	// is twice the number of CPU cores.
	max := runtime.NumCPU() * 2

	// clamp returns the value limited to the given range [min, max].
	clamp := func(value, min, max int) int {
		if value < min {
			return min
		}
		if value > max {
			return max
		}
		return value
	}

	// The minimum number of parallel tasks is 2.
	// And the maximum number is twice the number of CPU cores.
	parallelTasks = clamp(pt, 2, max)
	return parallelTasks
}

// Load loads new keys only (without updating existing keys) from env-file
// into environment. Handles variables like ${var} or $var in the value,
// replacing them with a real result.
//
// Returns an error if the env-file contains incorrect data,
// file is damaged or missing.
//
// Examples:
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
// Load values from configuration file into environment:
//
//	if err := env.Load(".env"); err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Printf("LAST_ID=%s\n", env.Get("LAST_ID"))
//	fmt.Printf("KEY_0=%s\n", env.Get("KEY_0"))
//	fmt.Printf("KEY_1=%s\n", env.Get("KEY_1"))
//	fmt.Printf("KEY_2=%s\n", env.Get("KEY_2"))
//	// Output:
//	//  LAST_ID=002
//	//  KEY_0=VALUE_DEF
//	//  KEY_1=VALUE_001
//	//  KEY_2=VALUE_002
//
// Where:
//   - KEY_0 - has not been replaced by VALUE_000;
//   - KEY_1 - loaded new value;
//   - KEY_2 - loaded new value and replaced ${LAST_ID}
//     to the value from environment.
func Load(filename string) error {
	expand, update, forced := true, false, false
	return readParseStore(filename, expand, update, forced)
}

// LoadSafe loads new keys only (without updating existing keys) from env-file
// into environment. Doesn't handles variables like ${var} or $var -
// doesn't turn them into a finite value.
//
// Returns an error if the env-file contains incorrect data,
// file is damaged or missing.
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
// Load values from configuration file into environment:
//
//	if err := env.LoadSafe(".env"); err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Printf("LAST_ID=%s\n", env.Get("LAST_ID"))
//	fmt.Printf("KEY_0=%s\n", env.Get("KEY_0"))
//	fmt.Printf("KEY_1=%s\n", env.Get("KEY_1"))
//	fmt.Printf("KEY_2=%s\n", env.Get("KEY_2"))
//	// Output:
//	//  LAST_ID=002
//	//  KEY_0=VALUE_DEF
//	//  KEY_1=VALUE_001
//	//  KEY_2=VALUE_${LAST_ID}
//
// Where:
//   - KEY_0 - has not been replaced by VALUE_000;
//   - KEY_1 - loaded new value;
//   - KEY_2 - loaded new value but doesn't replace ${LAST_ID}
//     to the value from environment.
func LoadSafe(filename string) error {
	expand, update, forced := false, false, false
	return readParseStore(filename, expand, update, forced)
}

// Update loads keys from the env-file into environment, update existing keys.
// Handles variables like ${var} or $var in the value,
// replacing them with a real result.
//
// Returns an error if the env-file contains incorrect data,
// file is damaged or missing.
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
// Load values from configuration file into environment:
//
//	if err := env.Update(".env"); err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Printf("LAST_ID=%s\n", env.Get("LAST_ID"))
//	fmt.Printf("KEY_0=%s\n", env.Get("KEY_0"))
//	fmt.Printf("KEY_1=%s\n", env.Get("KEY_1"))
//	fmt.Printf("KEY_2=%s\n", env.Get("KEY_2"))
//	// Output:
//	//  LAST_ID=002
//	//  KEY_0=VALUE_000
//	//  KEY_1=VALUE_001
//	//  KEY_2=VALUE_002
//
// Where:
//   - KEY_0 - replaced VALUE_DEF on VALUE_000;
//   - KEY_1 - loaded new value;
//   - KEY_2 - loaded new value and replaced ${LAST_ID}
//     to the value from environment.
func Update(filename string) error {
	expand, update, forced := true, true, false
	return readParseStore(filename, expand, update, forced)
}

// UpdateSafe loads keys from the env-file into environment,
// update existing keys. Doesn't handles variables like ${var} or $var -
// doesn't turn them into a finite value.
//
// Returns an error if the env-file contains incorrect data,
// file is damaged or missing.
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
// Load values from configuration file into environment:
//
//	if err := env.Update(".env"); err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Printf("LAST_ID=%s\n", env.Get("LAST_ID"))
//	fmt.Printf("KEY_0=%s\n", env.Get("KEY_0"))
//	fmt.Printf("KEY_1=%s\n", env.Get("KEY_1"))
//	fmt.Printf("KEY_2=%s\n", env.Get("KEY_2"))
//	// Output:
//	//  LAST_ID=002
//	//  KEY_0=VALUE_000
//	//  KEY_1=VALUE_001
//	//  KEY_2=VALUE_${LAST_ID}
//
// Where:
//   - KEY_0 - replaced VALUE_DEF on VALUE_000;
//   - KEY_1 - loaded new value;
//   - KEY_2 - loaded new value but doesn't replace ${LAST_ID}
//     to the value from environment.
func UpdateSafe(filename string) error {
	expand, update, forced := false, true, false
	return readParseStore(filename, expand, update, forced)
}

// Save saves the object to a file without changing the environment.
//
// # Example
//
// There is some configuration structure:
//
//	// Config it's struct of the server configuration.
//	type Config struct {
//		Host         string   `env:"HOST"`
//		Port         int      `env:"PORT"`
//		AllowedHosts []string `env:"ALLOWED_HOSTS" sep:":"` // parse by `:`.
//	}
//
// ...
//
//	var config = Config{
//		Host:         "localhost",
//		Port:         8080,
//		AllowedHosts: []string{"localhost", "127.0.0.1"},
//	}
//	env.Save("/tmp/.env", "", config)
//
// The result in the file /tmp/.env
//
//	HOST=localhost
//	PORT=8080
//	ALLOWED_HOSTS=localhost:127.0.0.1
func Save(filename, prefix string, obj interface{}) error {
	var result bytes.Buffer

	items, err := marshalEnv(prefix, obj, true) // don't change environment
	if err != nil {
		return err
	}

	for _, item := range items {
		result.WriteString(item)
		result.WriteString("\n")
	}

	return os.WriteFile(filename, result.Bytes(), 0o644)
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
//	if err := env.Update("./cmd/.env"); err != nil {
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

// Unmarshal parses data from the environment and store result into
// Go-structure that passed by pointer. If the obj isn't a pointer to
// struct or has fields of unsupported types will be returned an error.
//
// Method supports the following type of the fields: int, int8, int16, int32,
// int64, uin, uint8, uin16, uint32, in64, float32, float64, string, bool,
// struct, url.URL and pointers, array or slice from thous types (i.e. *int,
// *uint, ..., []int, ..., []bool, ..., [2]*url.URL, etc.). The fields as
// a struct or pointer on the struct will be processed recursively.
//
// If the structure implements Unmarshaler interface -
// the custom UnmarshalEnv method will be called.
//
// Use the following tags in the fields of structure to
// set the unmarshing parameters:
//
//	env  matches the name of the key in the environment;
//	def  default value (if empty, sets the default value
//	     for the field type of structure);
//	sep  sets the separator for lists/arrays (default ` ` - space).
//
// # Examples
//
// Some keys was set into environment as:
//
//	$ export HOST="0.0.0.0"
//	$ export PORT=8080
//	$ export ALLOWED_HOSTS=localhost:127.0.0.1
//	$ export SECRET_KEY=AgBsdjONL53IKa33LM9SNROvD3hZXfoz
//
// Structure example:
//
//	// Config structure for containing values from the environment.
//	// P.s. There is no need to describe all the keys in the environment,
//	// for example, we ignore the SECRET_KEY key.
//	type Config struct {
//		Host         string   `env:"HOST"`
//		Port         int      `env:"PORT" def:"80"`
//		AllowedHosts []string `env:"ALLOWED_HOSTS" sep:":"`
//	}
//
// Unmarshal data from the environment into Config structure.
//
//	var config Config
//	if err := env.Unmarshal("", &config); err != nil {
//		log.Fatal(err)
//	}
//
//	fmt.Printf("Host: %v\n", config.Host)
//	fmt.Printf("Port: %v\n", config.Port)
//	fmt.Printf("AllowedHosts: %v\n", config.AllowedHosts)
//	// Output:
//	//  Host: 0.0.0.0
//	//  Port: 8080
//	//  AllowedHosts: [localhost 127.0.0.1]
//
// If the structure will has custom UnmarshalEnv it will be called:
//
//	// UnmarshalEnv it's custom method for unmarshalling.
//	func (c *Config) UnmarshalEnv() error {
//	    c.Host = "192.168.0.1"
//	    c.Port = 80
//	    c.AllowedHosts = []string{"192.168.0.1"}
//	    return nil
//	}
//
//	...
//
//	// Output:
//	//  Host: 192.168.0.1
//	//  Port: 80
//	//  AllowedHosts: [192.168.0.1]
func Unmarshal(prefix string, obj interface{}) error {
	return unmarshalEnv(prefix, obj)
}

// Marshal converts the structure in to key/value and put it into environment
// with update old values. As the first value returns a list of keys that
// were correctly sets in the environment and nil or error information
// as second value.
//
// If the obj isn't a pointer to struct, struct or has fields of
// unsupported types will be returned an error.
//
// Method supports the following type of the fields: int, int8, int16, int32,
// int64, uin, uint8, uin16, uint32, in64, float32, float64, string, bool,
// struct, url.URL and pointers, array or slice from thous types (i.e. *int,
// *uint, ..., []int, ..., []bool, ..., [2]*url.URL, etc.). The fields as
// a struct or pointer on the struct will be processed recursively.
//
// If the structure implements Marshaler interface - the custom MarshalEnv
// method will be called.
//
// Use the following tags in the fields of structure to
// set the marshing parameters:
//
//   - env  matches the name of the key in the environment;
//   - def  default value (if empty, sets the default value
//     for the field type of structure);
//   - sep  sets the separator for lists/arrays (default ` ` - space).
//
// Structure example:
//
//	// Config structure for containing values from the environment.
//	type Config struct {
//		Host         string   `env:"HOST"`
//		Port         int      `env:"PORT" def:"80"`
//		AllowedHosts []string `env:"ALLOWED_HOSTS" sep:":"`
//	}
//
// Marshal data into environment from the Config.
//
//	var config = Config{
//		"localhost",
//		8080,
//		[]string{"localhost", "127.0.0.1"},
//	}
//
//	if _, err := env.Marshal("", config); err != nil {
//		log.Fatal(err)
//	}
//
//	fmt.Printf("Host: %v\n", env.Get("HOST"))
//	fmt.Printf("Port: %v\n", env.Get("PORT"))
//	fmt.Printf("AllowedHosts: %v\n", env.Get("ALLOWED_HOSTS"))
//	// Output:
//	//  Host: localhost
//	//  Port: 8080
//	//  AllowedHosts: localhost:127.0.0.1
//
// If the object has MarshalEnv and is not a nil pointer -
// will call its method to marshaling.
//
//	// MarshalEnv it's custom method for marshalling.
//	func (c *Config) MarshalEnv() ([]string, error) {
//		env.Set("HOST", "192.168.0.1")
//		env.Set("PORT", "80")
//		env.Set("ALLOWED_HOSTS", "192.168.0.1")
//		return []string{"HOST", "PORT", "ALLOWED_HOSTS"}, nil
//	}
//
//	...
//
//	// Output:
//	//  Host: 192.168.0.1
//	//  Port: 80
//	//  AllowedHosts: 192.168.0.1
func Marshal(prefix string, scope interface{}) ([]string, error) {
	return marshalEnv(prefix, scope, false)
}
