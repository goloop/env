package env

import (
	"bufio"
	"os"
)

// ReadParseStore reads env-file, parses this one by the key and value, and
// stores into environment. It's flexible function that can be used to build
// more specific tools.
//
// Arguments
//
//     filename  path to the env-file;
//     expand    if true replaces ${var} or $var on the values
//               from the current environment variables;
//     update    if true overwrites the value that has already been
//               set in the environment to new one from the env-file;
//     forced    if true ignores wrong entries in env-file and loads
//               all of possible options, without causing an exception.
//
// Examples
//
// There is `.env` env-file that contains:
//
//     # .env file
//     HOST=0.0.0.0
//     PORT=80
//     EMAIL=$USER@goloop.one
//
// Some variables are already exists in the environment:
//
//     $ env | grep -E "USER|HOST"
//     USER=goloop
//     HOST=localhost
//
// To correctly load data from env-file followed by updating the environment:
//
//     env.ReadParseStore(".env", true, true, false)
//
//     // USER=goloop
//     // HOST=0.0.0.0
//     // PORT=80
//     // EMAIL=goloop@goloop.one
//
// Loading new keys to the environment without updating existing ones:
//
//     env.ReadParseStore(".env", true, false, false)
//
//     // USER=goloop
//     // HOST=localhost          <= hasn't been updated
//     // PORT=80
//     // EMAIL=goloop@goloop.one
//
// Don't change values that contain keys:
//
//     env.ReadParseStore(".env", false, true, false)
//
//     // USER=goloop
//     // HOST=0.0.0.0
//     // PORT=80
//     // EMAIL=$USER@goloop.one  <= $USER hasn't been changed to real value
//
// Loading data from a damaged env-file. If the configuration env-file is used
// by other applications and can have incorrect key/value, it can be ignored.
// For example env-file contains incorrect key `1BC` (the variable name can't
// start with a digit):
//
//     # .env file
//     HOST=0.0.0.0
//     PORT=80
//     1BC=NO                     # <= incorrect variable
//     EMAIL=$USER@goloop.one
//
// There will be an error loading this file:
//
//     err := env.ReadParseStore(".env", true, true, false)
//     if err != nil {
//         log.Panic(err) // panic: missing variable name
//     }
//
// but we can use force method to ignore this line:
//
//     // ... set forced as true (last argument)
//     err := env.ReadParseStore(".env", true, true, true)
//
//     // now the err variable is nil and environment has:
//     // USER=goloop
//     // HOST=0.0.0.0
//     // PORT=80
//     // EMAIL=goloop@goloop.one
func ReadParseStore(filename string, expand, update, forced bool) (err error) {
	var (
		file       *os.File
		key, value string
	)

	// Open env-file.
	file, err = os.Open(filename)
	if err != nil {
		return // unable to open file
	}
	defer file.Close()

	// Parse file.
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		// Get current line and ignore empty string or comments.
		str := scanner.Text()
		if isEmpty(str) {
			continue
		}

		// Parse expression.
		// The string containing the expression must be of the
		// format as: [export] KEY=VALUE [# Comment]
		key, value, err = parseExpression(str)
		if err != nil {
			if forced {
				continue // ignore wrong entry
			}
			return // incorrect expression
		}

		// Overwrite or add new value.
		if _, ok := os.LookupEnv(key); update || !ok {
			if expand {
				value = Expand(value)
			}
			err = Set(key, value)
			if err != nil {
				return
			}
		}
	}

	return scanner.Err()
}

// Load loads new keys only (without updating existing keys) from env-file
// into environment. Handles variables like ${var} or $var in the value,
// replacing them with a real result.
//
// Returns an error if the env-file contains incorrect data or file
// is damaged or missing.
//
// Examples
//
// In this example, some variables are already set in the environment:
//
//     $ env | grep KEY_0
//     KEY_0=VALUE_DEF
//
// Configuration file `.env` contains:
//
//     LAST_ID=002
//     KEY_0=VALUE_000
//     KEY_1=VALUE_001
//     KEY_2=VALUE_${LAST_ID}
//
// Load values from configuration file into environment:
//
//     err := env.Load(".env")
//     if err != nil {
//         // error handling here ...
//     }
//
//     // LAST_ID=002
//     // KEY_0=VALUE_DEF  <= not replaced by VALUE_000
//     // KEY_1=VALUE_001  <= add new value
//     // KEY_2=VALUE_002  <= add new value and replaced ${LAST_ID}
//     //                     to the value from environment
//
// P.s. It's synonym for ReadParseStore (filename, true, false, false)
func Load(filename string) error {
	var expand, update, forced = true, false, false
	return ReadParseStore(filename, expand, update, forced)
}

// LoadSafe loads new keys only (without updating existing keys) from env-file
// into environment. Don't handles variables like ${var} or $var in the value.
//
// Returns an error if the env-file contains incorrect data or file
// is damaged or missing.
//
// Examples
//
// In this example, some variables are already set in the environment:
//
//     $ env | grep KEY_0
//     KEY_0=VALUE_DEF
//
// Configuration file `.env` contains:
//
//     LAST_ID=002
//     KEY_0=VALUE_000
//     KEY_1=VALUE_001
//     KEY_2=VALUE_${LAST_ID}
//
// Load values from configuration file into environment:
//
//     err := env.LoadSafe(".env")
//     if err != nil {
//         // error handling here ...
//     }
//
//     // LAST_ID=002
//     // KEY_0=VALUE_DEF         <= not replaced by VALUE_000
//     // KEY_1=VALUE_001         <= add new value
//     // KEY_2=VALUE_${LAST_ID}  <= add new value and don't replace ${LAST_ID}
//
// P.s. It's synonym for ReadParseStore (filename, false, false, false)
func LoadSafe(filename string) error {
	var expand, update, forced = false, false, false
	return ReadParseStore(filename, expand, update, forced)
}

// Update loads keys from the env-file into environment, update existing keys.
// Handles variables like ${var} or $var in the value,
// replacing them with a real result.
//
// Returns an error if the env-file contains incorrect data or file
// is damaged or missing.
//
// Examples
//
// In this example, some variables are already set in the environment:
//
//     $ env | grep KEY_0
//     KEY_0=VALUE_DEF
//
// Configuration file `.env` contains:
//
//     LAST_ID=002
//     KEY_0=VALUE_000
//     KEY_1=VALUE_001
//     KEY_2=VALUE_${LAST_ID}
//
// Load values from configuration file into environment:
//
//     err := env.Update(".env")
//     if err != nil {
//         // error handling here ...
//     }
//
//     // LAST_ID=002
//     // KEY_0=VALUE_000  <= replaced VALUE_DEF on VALUE_000
//     // KEY_1=VALUE_001  <= add new value
//     // KEY_2=VALUE_002  <= add new value and replaced ${LAST_ID}
//     //                     to the value from environment
//
// P.s. It's synonym for ReadParseStore (filename, true, true, false)
func Update(filename string) error {
	var expand, update, forced = true, true, false
	return ReadParseStore(filename, expand, update, forced)
}

// UpdateSafe loads keys from the env-file into environment, update existing keys.
// Don't handles variables like ${var} or $var in the value.
//
// Returns an error if the env-file contains incorrect data or file
// is damaged or missing.
//
// Examples
//
// In this example, some variables are already set in the environment:
//
//     $ env | grep KEY_0
//     KEY_0=VALUE_DEF
//
// Configuration file `.env` contains:
//
//     LAST_ID=002
//     KEY_0=VALUE_000
//     KEY_1=VALUE_001
//     KEY_2=VALUE_${LAST_ID}
//
// Load values from configuration file into environment:
//
//     err := env.UpdateSafe(".env")
//     if err != nil {
//         // error handling here ...
//     }
//
//     // LAST_ID=002
//     // KEY_0=VALUE_000         <= replaced VALUE_DEF on VALUE_000
//     // KEY_1=VALUE_001         <= add new value
//     // KEY_2=VALUE_${LAST_ID}  <= add new value and don't replace ${LAST_ID}
//
// P.s. It's synonym for ReadParseStore (filename, false, true, false)
func UpdateSafe(filename string) error {
	var expand, update, forced = false, true, false
	return ReadParseStore(filename, expand, update, forced)
}

// Exists returns true if all given keys exist in the environment.
//
// Examples
//
//
// In this example, some variables are already set in the environment:
//
//     $ env | grep KEY_0
//     KEY_0=VALUE_DEF
//
//
// Configuration file `.env` contains:
//
//     LAST_ID=002
//     KEY_0=VALUE_000
//     KEY_1=VALUE_001
//     KEY_2=VALUE_${LAST_ID}
//
// Check if a variable exists in the environment:
//
//    env.Exists("KEY_0")          // true
//    env.Exists("KEY_1")          // false
//    env.Exists("KEY_0", "KEY_1") // false
//
//    err := env.Update(".env")
//    if err != nil {
//         // error handling here ...
//    }
//
//    env.Exists("KEY_0")          // true
//    env.Exists("KEY_1")          // true
//    env.Exists("KEY_0", "KEY_1") // true
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
// If the structure implements Unmarshaler interface - the custom UnmarshalENV
// method will be called.
//
// Field's tag format is `env:"key[,value[,sep]]"`, where:
//
//     key    matches the name of the key in the environment;
//     value  default value (if empty, set the default
//            value for the structure field type);
//     sep    optional argument, sets the separator for lists (default `:`).
//
// Examples
//
// Some keys was set into environment as:
//
//     $ export HOST="0.0.0.0"
//     $ export PORT=8080
//     $ export ALLOWED_HOSTS=localhost:127.0.0.1
//     $ export SECRET_KEY=AgBsdjONL53IKa33LM9SNROvD3hZXfoz
//
// Structure example:
//
//     // Config structure for containing values from the environment.
//     // P.s. No need to describe all the keys that are in the environment,
//     // for example, we ignore the SECRET_KEY key.
//     type Config struct {
//         Host         string   `env:"HOST"`
//         Port         int      `env:"PORT"`
//         AllowedHosts []string `env:"ALLOWED_HOSTS,,:"`
//     }
//
// Unmarshal data from the environment into Config structure.
//
//     // Important: pointer to initialized structure!
//     var config = &Config{}
//
//     err := env.Unmarshal(config)
//     if err != nil {
//         // error handling here ...
//     }
//
//     config.Host         // "0.0.0.0"
//     config.Port         // 8080
//     config.AllowedHosts // []string{"localhost", "127.0.0.1"}
//
// If the structure will havs custom UnmarshalENV it will be called:
//
//     // UnmarshalENV it's custom method for unmarshalling.
//     func (c *Config) UnmarshalENV() error {
//         c.Host = "192.168.0.1"
//         c.Port = 80
//         c.AllowedHosts = []string{"192.168.0.1"}
//
//         return nil
//     }
//     ...
//     // The result will be the data that sets in the custom
//     // unmarshalling method.
//     config.Host         // "192.168.0.1"
//     config.Port         // 80
//     config.AllowedHosts // []string{"192.168.0.1"}
func Unmarshal(prefix string, obj interface{}) error {
	return unmarshalENV(prefix, obj)
}

// Marshal converts the structure in to key/value and put it into environment
// with update old values. The first return value returns a map of the data
// that was correct set into environment. The seconden - error or nil.
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
// If the structure implements Marshaler interface - the custom MarshalENV
// method will be called.
//
// Field's tag format is `env:"key[,value[,sep]]"`, where:
//
//     key    matches the name of the key in the environment;
//     value  default value (if empty, set the default
//            value for the structure field type);
//     sep    optional argument, sets the separator for lists (default `:`).
//
// Structure example:
//
//     // Config structure.
//     type Config struct {
//         Host         string   `env:"HOST"`
//         Port         int      `env:"PORT"`
//         AllowedHosts []string `env:"ALLOWED_HOSTS,,:"`
//     }
//
// Marshal data into environment from the Config.
//
//     // It can be a structure or a pointer on a structure.
//     var config = &Config{
//         "localhost",
//         8080,
//         []string{"localhost", "127.0.0.1"},
//     }
//
//     // Returns:
//     // map[ALLOWED_HOSTS:localhost:127.0.0.1 HOST:localhost PORT:8080], nil
//     _, err := env.Marshal(config)
//     if err != nil {
//         // error handling here ...
//     }
//
//     env.Get("HOST")          // "localhost"
//     env.Get("PORT")          // "8080"
//     env.Get("ALLOWED_HOSTS") // "localhost:127.0.0.1"
//
// If object has MarshalENV and isn't a nil pointer - will be calls it
// method to scope convertation.
//
//     // MarshalENV it's custom method for marshalling.
//     func (c *Config) MarshalENV() ([]string, error ){
//         os.Setenv("HOST", "192.168.0.1")
//         os.Setenv("PORT", "80")
//         os.Setenv("ALLOWED_HOSTS", "192.168.0.1")
//
//         return []string{}, nil
//     }
//     ...
//     // The result will be the data that sets in the custom
//     // unmarshalling method.
//     env.Get("HOST")          // "192.168.0.1"
//     env.Get("PORT")          // "80"
//     env.Get("ALLOWED_HOSTS") // "192.168.0.1"
func Marshal(prefix string, scope interface{}) ([]string, error) {
	return marshalENV(prefix, scope)
}
