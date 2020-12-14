[//]: # (!!!Don't modify the README.md, use `make readme` to generate it!!!)


[![Go Report Card](https://goreportcard.com/badge/github.com/goloop/env)](https://goreportcard.com/report/github.com/goloop/env) [![License](https://img.shields.io/badge/license-BSD-blue)](https://github.com/goloop/env/blob/master/LICENSE) [![License](https://img.shields.io/badge/godoc-YES-green)](https://godoc.org/github.com/goloop/env)

*Version: 0.0.6*


# env

The env module contains synonyms for the standard methods from the os module to
manage environment and implements various additional methods which allow save
environment variables to a GoLang structure.

    - set environment variables from env-file;
    - marshal Go-structure to the environment;
    - unmarshal environment variables to Go-structure;
    - set variables to the environment;
    - delete variables from the environment;
    - determine the presence of a variable in the environment;
    - get value of variable by key from the environment;
    - cleaning environment.

The env-file can contain any environment valid constructions:

    # Supported:
    # comment as a separate entry
    KEY_000="value for key 000" # comment as part of a line
    export KEY_001="value for key 001" # has an export command
    KEY_002=one:two:three # list with default separator
    KEY_003=one,two,three # list with `,` comma separator
    KEY_004=7 # numbers
    KEY_005=John # string without quotes
    # empty line

    KEY_006=one,"two,three",four # grouped values as one | two,three | four
    KEY_007=${KEY_005}00$KEY_004 # variables: John007

    # Not supported:
    # KEY_008= # empty value without quotes, use as: KEY_008=''
    # 009_KEY=5 # incorrect variable name (name cannot starts with a digit)
    # KEY_010="broken value, has no closing quote

## Installation

To install the package you can use `go get`:

    $ go get -u github.com/goloop/env

## Usage

To use this module import it as:

    import "github.com/goloop/env"

### Tag format

Field's tag format is `env:"key[,value[,sep]]"`, where:

    key    matches the name of the key in the environment;
    value  default value (if empty, set the default
           value for the structure field type);
    sep    optional argument, sets the separator for lists (default `:`).

### Quick start

There is a web-project that is develop and tests on the local computer and
deploys on the production server. On the production server some settings (for
example `host` and `port`) forced stored in an environment but on the local
computer data must be loaded from the file (because different team members have
different launch options).

For example, the local-configuration file `.env` contains:

    HOST=0.0.0.0
    PORT=8080
    ALLOWED_HOSTS="localhost:127.0.0.1"

    # In configurations file and/or in the environment there can be
    # a many of variables that willn't be parsed, like this:
    SECRET_KEY="N0XRABLZ5ZZY6dCNgD7pLjTIlx8v4G5d"

We need to load the missing configurations from the file into the environment.
Convert data from the environment into Go-structure. And use this data to start
the server.

**Note:** We need load the missing variables but not update the existing ones.

    package main

    import (
        "fmt"
        "log"
        "net/http"

        "github.com/goloop/env"
    )

    // Config it's struct of the server configuration.
    type Config struct {
        Host         string   `env:"HOST"`
        Port         int      `env:"PORT"`
        AllowedHosts []string `env:"ALLOWED_HOSTS,,:"` // parse by `:`.

        // P.s. It isn't necessary to specify all the keys
        // that are available in the environment.
    }

    // Addr returns the server's address - concatenate host
    // and port into one string.
    func (c *Config) Addr() string {
        return fmt.Sprintf("%s:%d", c.Host, c.Port)
    }

    // Home it is handler of the home page.
    func Home(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "Hello World!")
    }

    func main() {
        var config = Config{}

        // Load configurations from the env-file into an environment.
        // P.s. We use the Load (but not Update) method for as not to overwrite
        // the data in the environment because on the production server this
        // data can be set forcibly.
        env.Load(".env") // set correct the path to the file with variables

        // Parsing of the environment data and storing the result
        // into object by pointer.
        env.Unmarshal(&config)

        // Make routing.
        http.HandleFunc("/", Home)

        // Run.
        log.Printf("Server started on %s\n", config.Addr())
        log.Printf("Allowed hosts: %v\n", config.AllowedHosts)
        http.ListenAndServe(config.Addr(), nil)
    }


## Usage

#### func  Clear

    func Clear()

Clear is synonym for the os.Clearenv, deletes all environment variables.

#### func  Environ

    func Environ() []string

Environ is synonym for the os.Environ, returns a copy of strings representing
the environment, in the form "key=value".

#### func  Exists

    func Exists(keys ...string) bool

Exists returns true if all given keys exist in the environment.


Examples

In this example, some variables are already set in the environment:

    $ env | grep KEY_0
    KEY_0=VALUE_DEF

Configuration file `.env` contains:

    LAST_ID=002
    KEY_0=VALUE_000
    KEY_1=VALUE_001
    KEY_2=VALUE_${LAST_ID}

Check if a variable exists in the environment:

    env.Exists("KEY_0")          // true
    env.Exists("KEY_1")          // false
    env.Exists("KEY_0", "KEY_1") // false

    err := env.Update(".env")
    if err != nil {
         // error handling here ...
    }

    env.Exists("KEY_0")          // true
    env.Exists("KEY_1")          // true
    env.Exists("KEY_0", "KEY_1") // true

#### func  Expand

    func Expand(value string) string

Expand is synonym for the os.Expand, replaces ${var} or $var in the string
according to the values of the current environment variables. References to
undefined variables are replaced by the empty string.

#### func  Get

    func Get(key string) string

Get is synonym for the os.Getenv, retrieves the value of the environment
variable named by the key. It returns the value, which will be empty if the
variable is not present.

#### func  Load

    func Load(filename string) error

Load loads new keys only (without updating existing keys) from env-file into
environment. Handles variables like ${var} or $var in the value, replacing them
with a real result.

Returns an error if the env-file contains incorrect data or file is damaged or
missing.


Examples

In this example, some variables are already set in the environment:

    $ env | grep KEY_0
    KEY_0=VALUE_DEF

Configuration file `.env` contains:

    LAST_ID=002
    KEY_0=VALUE_000
    KEY_1=VALUE_001
    KEY_2=VALUE_${LAST_ID}

Load values from configuration file into environment:

    err := env.Load(".env")
    if err != nil {
        // error handling here ...
    }

    // LAST_ID=002
    // KEY_0=VALUE_DEF  <= not replaced by VALUE_000
    // KEY_1=VALUE_001  <= add new value
    // KEY_2=VALUE_002  <= add new value and replaced ${LAST_ID}
    //                     to the value from environment

P.s. It's synonym for ReadParseStore (filename, true, false, false)

#### func  LoadSafe

    func LoadSafe(filename string) error

LoadSafe loads new keys only (without updating existing keys) from env-file into
environment. Don't handles variables like ${var} or $var in the value.

Returns an error if the env-file contains incorrect data or file is damaged or
missing.


Examples

In this example, some variables are already set in the environment:

    $ env | grep KEY_0
    KEY_0=VALUE_DEF

Configuration file `.env` contains:

    LAST_ID=002
    KEY_0=VALUE_000
    KEY_1=VALUE_001
    KEY_2=VALUE_${LAST_ID}

Load values from configuration file into environment:

    err := env.LoadSafe(".env")
    if err != nil {
        // error handling here ...
    }

    // LAST_ID=002
    // KEY_0=VALUE_DEF         <= not replaced by VALUE_000
    // KEY_1=VALUE_001         <= add new value
    // KEY_2=VALUE_${LAST_ID}  <= add new value and don't replace ${LAST_ID}

P.s. It's synonym for ReadParseStore (filename, false, false, false)

#### func  Marshal

    func Marshal(scope interface{}) ([]string, error)

Marshal converts the structure in to key/value and put it into environment with
update old values. The first return value returns a map of the data that was
correct set into environment. The seconden - error or nil.

If the obj isn't a pointer to struct, struct or has fields of unsupported types
will be returned an error.

Method supports the following type of the fields: int, int8, int16, int32,
int64, uin, uint8, uin16, uint32, in64, float32, float64, string, bool, struct,
url.URL and pointers, array or slice from thous types (i.e. *int, *uint, ...,
[]int, ..., []bool, ..., [2]*url.URL, etc.). The fields as a struct or pointer
on the struct will be processed recursively.

If the structure implements Marshaler interface - the custom MarshalENV method
will be called.

Field's tag format is `env:"key[,value[,sep]]"`, where:

    key    matches the name of the key in the environment;
    value  default value (if empty, set the default
           value for the structure field type);
    sep    optional argument, sets the separator for lists (default `:`).

Structure example:

    // Config structure.
    type Config struct {
        Host         string   `env:"HOST"`
        Port         int      `env:"PORT"`
        AllowedHosts []string `env:"ALLOWED_HOSTS,,:"`
    }

Marshal data into environment from the Config.

    // It can be a structure or a pointer on a structure.
    var config = &Config{
        "localhost",
        8080,
        []string{"localhost", "127.0.0.1"},
    }

    // Returns:
    // map[ALLOWED_HOSTS:localhost:127.0.0.1 HOST:localhost PORT:8080], nil
    _, err := env.Marshal(config)
    if err != nil {
        // error handling here ...
    }

    env.Get("HOST")          // "localhost"
    env.Get("PORT")          // "8080"
    env.Get("ALLOWED_HOSTS") // "localhost:127.0.0.1"

If object has MarshalENV and isn't a nil pointer - will be calls it method to
scope convertation.

    // MarshalENV it's custom method for marshalling.
    func (c *Config) MarshalENV() ([]string, error ){
        os.Setenv("HOST", "192.168.0.1")
        os.Setenv("PORT", "80")
        os.Setenv("ALLOWED_HOSTS", "192.168.0.1")

        return []string{}, nil
    }
    ...
    // The result will be the data that sets in the custom
    // unmarshalling method.
    env.Get("HOST")          // "192.168.0.1"
    env.Get("PORT")          // "80"
    env.Get("ALLOWED_HOSTS") // "192.168.0.1"

#### func  ReadParseStore

    func ReadParseStore(filename string, expand, update, forced bool) (err error)

Arguments

    filename  path to the env-file;
    expand    if true replaces ${var} or $var on the values
              from the current environment variables;
    update    if true overwrites the value that has already been
              set in the environment to new one from the env-file;
    forced    if true ignores wrong entries in env-file and loads
              all of possible options, without causing an exception.


Examples

There is `.env` env-file that contains:

    # .env file
    HOST=0.0.0.0
    PORT=80
    EMAIL=$USER@goloop.one

Some variables are already exists in the environment:

    $ env | grep -E "USER|HOST"
    USER=goloop
    HOST=localhost

To correctly load data from env-file followed by updating the environment:

    env.ReadParseStore(".env", true, true, false)

    // USER=goloop
    // HOST=0.0.0.0
    // PORT=80
    // EMAIL=goloop@goloop.one

Loading new keys to the environment without updating existing ones:

    env.ReadParseStore(".env", true, false, false)

    // USER=goloop
    // HOST=localhost          <= hasn't been updated
    // PORT=80
    // EMAIL=goloop@goloop.one

Don't change values that contain keys:

    env.ReadParseStore(".env", false, true, false)

    // USER=goloop
    // HOST=0.0.0.0
    // PORT=80
    // EMAIL=$USER@goloop.one  <= $USER hasn't been changed to real value

Loading data from a damaged env-file. If the configuration env-file is used by
other applications and can have incorrect key/value, it can be ignored. For
example env-file contains incorrect key `1BC` (the variable name can't start
with a digit):

    # .env file
    HOST=0.0.0.0
    PORT=80
    1BC=NO                     # <= incorrect variable
    EMAIL=$USER@goloop.one

There will be an error loading this file:

    err := env.ReadParseStore(".env", true, true, false)
    if err != nil {
        log.Panic(err) // panic: missing variable name
    }

but we can use force method to ignore this line:

    // ... set forced as true (last argument)
    err := env.ReadParseStore(".env", true, true, true)

    // now the err variable is nil and environment has:
    // USER=goloop
    // HOST=0.0.0.0
    // PORT=80
    // EMAIL=goloop@goloop.one

#### func  Set

    func Set(key, value string) error

Set is synonym for the os.Setenv, sets the value of the environment variable
named by the key. It returns an error, if any.

#### func  Unmarshal

    func Unmarshal(obj interface{}) error

Unmarshal parses data from the environment and store result into Go-structure
that passed by pointer. If the obj isn't a pointer to struct or has fields of
unsupported types will be returned an error.

Method supports the following type of the fields: int, int8, int16, int32,
int64, uin, uint8, uin16, uint32, in64, float32, float64, string, bool, struct,
url.URL and pointers, array or slice from thous types (i.e. *int, *uint, ...,
[]int, ..., []bool, ..., [2]*url.URL, etc.). The fields as a struct or pointer
on the struct will be processed recursively.

If the structure implements Unmarshaler interface - the custom UnmarshalENV
method will be called.

Field's tag format is `env:"key[,value[,sep]]"`, where:

    key    matches the name of the key in the environment;
    value  default value (if empty, set the default
           value for the structure field type);
    sep    optional argument, sets the separator for lists (default `:`).


Examples

Some keys was set into environment as:

    $ export HOST="0.0.0.0"
    $ export PORT=8080
    $ export ALLOWED_HOSTS=localhost:127.0.0.1
    $ export SECRET_KEY=AgBsdjONL53IKa33LM9SNROvD3hZXfoz

Structure example:

    // Config structure for containing values from the environment.
    // P.s. No need to describe all the keys that are in the environment,
    // for example, we ignore the SECRET_KEY key.
    type Config struct {
        Host         string   `env:"HOST"`
        Port         int      `env:"PORT"`
        AllowedHosts []string `env:"ALLOWED_HOSTS,,:"`
    }

Unmarshal data from the environment into Config structure.

    // Important: pointer to initialized structure!
    var config = &Config{}

    err := env.Unmarshal(config)
    if err != nil {
        // error handling here ...
    }

    config.Host         // "0.0.0.0"
    config.Port         // 8080
    config.AllowedHosts // []string{"localhost", "127.0.0.1"}

If the structure will havs custom UnmarshalENV it will be called:

    // UnmarshalENV it's custom method for unmarshalling.
    func (c *Config) UnmarshalENV() error {
        c.Host = "192.168.0.1"
        c.Port = 80
        c.AllowedHosts = []string{"192.168.0.1"}

        return nil
    }
    ...
    // The result will be the data that sets in the custom
    // unmarshalling method.
    config.Host         // "192.168.0.1"
    config.Port         // 80
    config.AllowedHosts // []string{"192.168.0.1"}

#### func  Unset

    func Unset(key string) error

Unset is synonym for the os.Unsetenv, unsets a single environment variable.

#### func  Update

    func Update(filename string) error

Update loads keys from the env-file into environment, update existing keys.
Handles variables like ${var} or $var in the value, replacing them with a real
result.

Returns an error if the env-file contains incorrect data or file is damaged or
missing.


Examples

In this example, some variables are already set in the environment:

    $ env | grep KEY_0
    KEY_0=VALUE_DEF

Configuration file `.env` contains:

    LAST_ID=002
    KEY_0=VALUE_000
    KEY_1=VALUE_001
    KEY_2=VALUE_${LAST_ID}

Load values from configuration file into environment:

    err := env.Update(".env")
    if err != nil {
        // error handling here ...
    }

    // LAST_ID=002
    // KEY_0=VALUE_000  <= replaced VALUE_DEF on VALUE_000
    // KEY_1=VALUE_001  <= add new value
    // KEY_2=VALUE_002  <= add new value and replaced ${LAST_ID}
    //                     to the value from environment

P.s. It's synonym for ReadParseStore (filename, true, true, false)

#### func  UpdateSafe

    func UpdateSafe(filename string) error

UpdateSafe loads keys from the env-file into environment, update existing keys.
Don't handles variables like ${var} or $var in the value.

Returns an error if the env-file contains incorrect data or file is damaged or
missing.


Examples

In this example, some variables are already set in the environment:

    $ env | grep KEY_0
    KEY_0=VALUE_DEF

Configuration file `.env` contains:

    LAST_ID=002
    KEY_0=VALUE_000
    KEY_1=VALUE_001
    KEY_2=VALUE_${LAST_ID}

Load values from configuration file into environment:

    err := env.UpdateSafe(".env")
    if err != nil {
        // error handling here ...
    }

    // LAST_ID=002
    // KEY_0=VALUE_000         <= replaced VALUE_DEF on VALUE_000
    // KEY_1=VALUE_001         <= add new value
    // KEY_2=VALUE_${LAST_ID}  <= add new value and don't replace ${LAST_ID}

P.s. It's synonym for ReadParseStore (filename, false, true, false)

#### type Marshaler

    type Marshaler interface {
    	MarshalENV() ([]string, error)
    }


Marshaler is the interface implemented by types that can marshal themselves into
valid object.

#### type Unmarshaler

    type Unmarshaler interface {
    	UnmarshalENV() error
    }


Unmarshaler is the interface implements by types that can unmarshal an
environment variables of themselves.
