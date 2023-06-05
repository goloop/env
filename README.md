[//]: # (!!!Don't modify the README.md, use `make readme` to generate it!!!)

[![Go Report Card](https://goreportcard.com/badge/github.com/goloop/env)](https://goreportcard.com/report/github.com/goloop/env) [![License](https://img.shields.io/badge/license-BSD-blue)](https://github.com/goloop/env/blob/master/LICENSE) [![License](https://img.shields.io/badge/godoc-YES-green)](https://godoc.org/github.com/goloop/env)

*Version: v1.0.1*

# env

Package env implements various methods that allow storing environment variables
in GoLang structures.


## Package features

The module contains synonyms for the standard methods from the os module like: Get, Set, Unset, Clear, Environ etc., to manage of environment and implements a special methods which allow saves variables from environment to a structures.

The main features:

- set variables of environment from the variables of env-file;
- convert (marshal) Go-structure to the variables of environment;
- pack (unmarshal) variables of environment to the Go-structure;
- set any variables to the environment;
- delete variables from the environment;
- determine the presence of a variable in the environment;
- get value of variable by key from the environment;
- cleaning environment.


The env-file can contain any environment valid constructions, for example:

```shell
# Supported:
# .................................................................
# Comments as separate entries.
KEY_000="value for key 000" # comment as part of the var declaration string
export KEY_001="value for key 001" # with export command
KEY_002=one:two:three # list with default separators, colon
KEY_003=one,two,three # ... or comma separator, or etc...
KEY_004=7 # numbers
KEY_005=John # string without quotes
PREFIX_KEY_005="Bob" # prefix filtering

# Empty line (up/down).

KEY_006=one,"two,three",four # grouped values as one | two,three | four
KEY_007=${KEY_005}00$KEY_004 # concatenation with variables: John007

KEY_008_LABEL="Service A" # deep embedded field
PREFIX_KEY_008_LABEL="Service B" # deep embedded field with prefix

# Not supported:
# .................................................................
# KEY_009= # empty value without quotes, need to use as: KEY_009=''
# 010_KEY=5 # incorrect variable name (name cannot starts with a digit)
# KEY_011="broken value, for example hasn't closing quote of the end
```

## Installation

To install the package you can use `go get`:

```
$ go get -u github.com/goloop/env
```


## Usage

To use this module import it as:

```go
import "github.com/goloop/env"
```


## Quick start

Let's marshal the env-file presented above to Go-structure.

```go
package main

import (
	"fmt"
	"log"

	"github.com/goloop/env"
)

// The key8 demonstrates a deeply nested field.
type key8 struct {
	Label string `env:"LABEL"`
}

// The config is object of some configuration.
type config struct {
	Key002 []string `env:"KEY_002" def:"[a:b:c]" sep:":"` // with default value
	Key005 string   `env:"KEY_005"`
	Key006 []string `env:"KEY_006" sep:","`
	Key008 key8     `env:"KEY_008"` // nested structure
}

func main() {
	// The file has several configurations with different prefixes.
	var a, b config

	// Update the environment.
	if err := env.Update(".env"); err != nil {
		log.Fatal(err)
	}

	// Unmarshal environment.
	env.Unmarshal("", &a)        // unmarshal all variables
	env.Unmarshal("PREFIX_", &b) // unmarshal variables with prefix only

	fmt.Printf("%v\n", a)
	fmt.Printf("%v\n", b)
	// Output:
	//  {[one two three] John [one "two,three" four] {Service A}}
	//  {[[a:b:c]] Bob [] {Service B}}
}
```

## Details

### Tags

Use the following tags in the fields of structure to
set the unmarshing parameters:

 - **env**  matches the name of the key in the environment;
 - **def**  default value (if empty, sets the default value for the field type of structure);
 - **sep**  sets the separator for lists/arrays (default ` ` - space).

### Examples

There is a web-project that is develop and tests on the local computer and
deploys on the production server. On the production server some settings
(for example `host` and `port`) forced stored in an environment but on the
local computer data must be loaded from the file (because different team
members have different launch options).

For example, the local-configuration file `.env` contains:

```shell
HOST=0.0.0.0
PORT=8080
ALLOWED_HOSTS="localhost:127.0.0.1"

# In configurations file and/or in the environment there can be
# a many of variables that willn't be parsed, like this:
SECRET_KEY="N0XRABLZ5ZZY6dCNgD7pLjTIlx8v4G5d"
```

We need to load the missing configurations from the file into the environment.
Convert data from the environment into Go-structure. And use this
data to start the server.

**Note:** We need load the missing variables but not update the existing ones.

```go
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
	AllowedHosts []string `env:"ALLOWED_HOSTS" sep:":"` // parse by `:`.
}

// Addr returns the server's address.
func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// The homeHandler it is handler of the home page.
func homeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello World!")
}

func main() {
	var config = Config{}

	// Load configurations from the env-file into an environment and
	// from environment to Go-struct.
	// Note: We use the Load (but not Update) method for as not to overwrite
	// the data in the environment because on the production server this
	// data can be set forcibly.
	env.Load(".env")
	env.Unmarshal("", &config)

	// Make routing.
	http.HandleFunc("/", homeHandler)

	// Run.
	log.Printf(
		"\nServer started on %s\nAllowed hosts: %v\n",
		config.Addr(),
		config.AllowedHosts,
	)
	log.Fatal(http.ListenAndServe(config.Addr(), nil))
	// Output:
	//  Server started on 0.0.0.0:0
	//  Allowed hosts: [localhost 127.0.0.1]
}
```
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

Exists returns true if all given keys exists in the environment.


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

     fmt.Printf("KEY_0 is %v\n", env.Exists("KEY_0"))
     fmt.Printf("KEY_1 is %v\n", env.Exists("KEY_1"))
     fmt.Printf("KEY_0 and KEY_1 is %v\n\n", env.Exists("KEY_0", "KEY_1"))

     if err := env.Update("./cmd/.env"); err != nil {
       log.Fatal(err)
     }

    	fmt.Printf("KEY_0 is %v\n", env.Exists("KEY_0"))
    	fmt.Printf("KEY_1 is %v\n", env.Exists("KEY_1"))
    	fmt.Printf("KEY_0 and KEY_1 is %v\n", env.Exists("KEY_0", "KEY_1"))

    	// Output:
    	//  KEY_0 is true
    	//  KEY_1 is false
    	//  KEY_0 and KEY_1 is false
    	//
    	//  KEY_0 is true
    	//  KEY_1 is true
    	//  KEY_0 and KEY_1 is true

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

To distinguish between an empty value and an unset value, use Lookup.

#### func  Load

    func Load(filename string) error

Load loads new keys only (without updating existing keys) from env-file into
environment. Handles variables like ${var} or $var in the value, replacing them
with a real result.

Returns an error if the env-file contains incorrect data, file is damaged or
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

    if err := env.Load(".env"); err != nil {
        log.Fatal(err)
    }

    fmt.Printf("LAST_ID=%s\n", env.Get("LAST_ID"))
    fmt.Printf("KEY_0=%s\n", env.Get("KEY_0"))
    fmt.Printf("KEY_1=%s\n", env.Get("KEY_1"))
    fmt.Printf("KEY_2=%s\n", env.Get("KEY_2"))
    // Output:
    //  LAST_ID=002
    //  KEY_0=VALUE_DEF
    //  KEY_1=VALUE_001
    //  KEY_2=VALUE_002

Where:

    - KEY_0 - has not been replaced by VALUE_000;
    - KEY_1 - loaded new value;
    - KEY_2 - loaded new value and replaced ${LAST_ID}
              to the value from environment.

#### func  LoadSafe

    func LoadSafe(filename string) error

LoadSafe loads new keys only (without updating existing keys) from env-file into
environment. Doesn't handles variables like ${var} or $var - doesn't turn them
into a finite value.

Returns an error if the env-file contains incorrect data, file is damaged or
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

    if err := env.LoadSafe(".env"); err != nil {
        log.Fatal(err)
    }

    fmt.Printf("LAST_ID=%s\n", env.Get("LAST_ID"))
    fmt.Printf("KEY_0=%s\n", env.Get("KEY_0"))
    fmt.Printf("KEY_1=%s\n", env.Get("KEY_1"))
    fmt.Printf("KEY_2=%s\n", env.Get("KEY_2"))
    // Output:
    //  LAST_ID=002
    //  KEY_0=VALUE_DEF
    //  KEY_1=VALUE_001
    //  KEY_2=VALUE_${LAST_ID}

Where:

    - KEY_0 - has not been replaced by VALUE_000;
    - KEY_1 - loaded new value;
    - KEY_2 - loaded new value but doesn't replace ${LAST_ID}
              to the value from environment.

#### func  Lookup

    func Lookup(key string) (string, bool)

Lookup is synonym for the os.LookupEnv, retrieves the value of the environment
variable named by the key. If the variable is present in the environment the
value (which may be empty) is returned and the boolean is true. Otherwise the
returned value will be empty and the boolean will be false.

#### func  Marshal

    func Marshal(prefix string, scope interface{}) ([]string, error)

Marshal converts the structure in to key/value and put it into environment with
update old values. As the first value returns a list of keys that were correctly
sets in the environment and nil or error information as second value.

If the obj isn't a pointer to struct, struct or has fields of unsupported types
will be returned an error.

Method supports the following type of the fields: int, int8, int16, int32,
int64, uin, uint8, uin16, uint32, in64, float32, float64, string, bool, struct,
url.URL and pointers, array or slice from thous types (i.e. *int, *uint, ...,
[]int, ..., []bool, ..., [2]*url.URL, etc.). The fields as a struct or pointer
on the struct will be processed recursively.

If the structure implements Marshaler interface - the custom MarshalEnv method
will be called.

Use the following tags in the fields of structure to set the marshing
parameters:

    env  matches the name of the key in the environment;
    def  default value (if empty, sets the default value
         for the field type of structure);
    sep  sets the separator for lists/arrays (default ` ` - space).

Structure example:

    // Config structure for containing values from the environment.
    type Config struct {
    	Host         string   `env:"HOST"`
    	Port         int      `env:"PORT" def:"80"`
    	AllowedHosts []string `env:"ALLOWED_HOSTS" sep:":"`
    }

Marshal data into environment from the Config.

    var config = Config{
    	"localhost",
    	8080,
    	[]string{"localhost", "127.0.0.1"},
    }

    if _, err := env.Marshal("", config); err != nil {
    	log.Fatal(err)
    }

    fmt.Printf("Host: %v\n", env.Get("HOST"))
    fmt.Printf("Port: %v\n", env.Get("PORT"))
    fmt.Printf("AllowedHosts: %v\n", env.Get("ALLOWED_HOSTS"))
    // Output:
    //  Host: localhost
    //  Port: 8080
    //  AllowedHosts: localhost:127.0.0.1

If the object has MarshalEnv and is not a nil pointer - will call its method to
marshaling.

    // MarshalEnv it's custom method for marshalling.
    func (c *Config) MarshalEnv() ([]string, error) {
    	env.Set("HOST", "192.168.0.1")
    	env.Set("PORT", "80")
    	env.Set("ALLOWED_HOSTS", "192.168.0.1")
    	return []string{"HOST", "PORT", "ALLOWED_HOSTS"}, nil
    }

    ...

    // Output:
    //  Host: 192.168.0.1
    //  Port: 80
    //  AllowedHosts: 192.168.0.1

#### func  Save

    func Save(filename, prefix string, obj interface{}) error

Save saves the object to a file without changing the environment.


Example

There is some configuration structure:

    // Config it's struct of the server configuration.
    type Config struct {
    	Host         string   `env:"HOST"`
    	Port         int      `env:"PORT"`
    	AllowedHosts []string `env:"ALLOWED_HOSTS" sep:":"` // parse by `:`.
    }

...

    var config = Config{
    	Host:         "localhost",
    	Port:         8080,
    	AllowedHosts: []string{"localhost", "127.0.0.1"},
    }
    env.Save("/tmp/.env", "", config)

The result in the file /tmp/.env

    HOST=localhost
    PORT=8080
    ALLOWED_HOSTS=localhost:127.0.0.1

#### func  Set

    func Set(key, value string) error

Set is synonym for the os.Setenv, sets the value of the environment variable
named by the key. It returns an error, if any.

#### func  Unmarshal

    func Unmarshal(prefix string, obj interface{}) error

Unmarshal parses data from the environment and store result into Go-structure
that passed by pointer. If the obj isn't a pointer to struct or has fields of
unsupported types will be returned an error.

Method supports the following type of the fields: int, int8, int16, int32,
int64, uin, uint8, uin16, uint32, in64, float32, float64, string, bool, struct,
url.URL and pointers, array or slice from thous types (i.e. *int, *uint, ...,
[]int, ..., []bool, ..., [2]*url.URL, etc.). The fields as a struct or pointer
on the struct will be processed recursively.

If the structure implements Unmarshaler interface - the custom UnmarshalEnv
method will be called.

Use the following tags in the fields of structure to set the unmarshing
parameters:

    env  matches the name of the key in the environment;
    def  default value (if empty, sets the default value
         for the field type of structure);
    sep  sets the separator for lists/arrays (default ` ` - space).


Examples

Some keys was set into environment as:

    $ export HOST="0.0.0.0"
    $ export PORT=8080
    $ export ALLOWED_HOSTS=localhost:127.0.0.1
    $ export SECRET_KEY=AgBsdjONL53IKa33LM9SNROvD3hZXfoz

Structure example:

    // Config structure for containing values from the environment.
    // P.s. There is no need to describe all the keys in the environment,
    // for example, we ignore the SECRET_KEY key.
    type Config struct {
    	Host         string   `env:"HOST"`
    	Port         int      `env:"PORT" def:"80"`
    	AllowedHosts []string `env:"ALLOWED_HOSTS" sep:":"`
    }

Unmarshal data from the environment into Config structure.

    var config Config
    if err := env.Unmarshal("", &config); err != nil {
    	log.Fatal(err)
    }

    fmt.Printf("Host: %v\n", config.Host)
    fmt.Printf("Port: %v\n", config.Port)
    fmt.Printf("AllowedHosts: %v\n", config.AllowedHosts)
    // Output:
    //  Host: 0.0.0.0
    //  Port: 8080
    //  AllowedHosts: [localhost 127.0.0.1]

If the structure will has custom UnmarshalEnv it will be called:

    // UnmarshalEnv it's custom method for unmarshalling.
    func (c *Config) UnmarshalEnv() error {
        c.Host = "192.168.0.1"
        c.Port = 80
        c.AllowedHosts = []string{"192.168.0.1"}
        return nil
    }

    ...

    // Output:
    //  Host: 192.168.0.1
    //  Port: 80
    //  AllowedHosts: [192.168.0.1]

#### func  Unset

    func Unset(key string) error

Unset is synonym for the os.Unsetenv, unsets a single environment variable.

#### func  Update

    func Update(filename string) error

Update loads keys from the env-file into environment, update existing keys.
Handles variables like ${var} or $var in the value, replacing them with a real
result.

Returns an error if the env-file contains incorrect data, file is damaged or
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

    if err := env.Update(".env"); err != nil {
        log.Fatal(err)
    }

    fmt.Printf("LAST_ID=%s\n", env.Get("LAST_ID"))
    fmt.Printf("KEY_0=%s\n", env.Get("KEY_0"))
    fmt.Printf("KEY_1=%s\n", env.Get("KEY_1"))
    fmt.Printf("KEY_2=%s\n", env.Get("KEY_2"))
    // Output:
    //  LAST_ID=002
    //  KEY_0=VALUE_000
    //  KEY_1=VALUE_001
    //  KEY_2=VALUE_002

Where:

    - KEY_0 - replaced VALUE_DEF on VALUE_000;
    - KEY_1 - loaded new value;
    - KEY_2 - loaded new value and replaced ${LAST_ID}
              to the value from environment.

#### func  UpdateSafe

    func UpdateSafe(filename string) error

UpdateSafe loads keys from the env-file into environment, update existing keys.
Doesn't handles variables like ${var} or $var - doesn't turn them into a finite
value.

Returns an error if the env-file contains incorrect data, file is damaged or
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

    if err := env.Update(".env"); err != nil {
        log.Fatal(err)
    }

    fmt.Printf("LAST_ID=%s\n", env.Get("LAST_ID"))
    fmt.Printf("KEY_0=%s\n", env.Get("KEY_0"))
    fmt.Printf("KEY_1=%s\n", env.Get("KEY_1"))
    fmt.Printf("KEY_2=%s\n", env.Get("KEY_2"))
    // Output:
    //  LAST_ID=002
    //  KEY_0=VALUE_000
    //  KEY_1=VALUE_001
    //  KEY_2=VALUE_${LAST_ID}

Where:

    - KEY_0 - replaced VALUE_DEF on VALUE_000;
    - KEY_1 - loaded new value;
    - KEY_2 - loaded new value but doesn't replace ${LAST_ID}
              to the value from environment.

#### func  Version

    func Version() string

Version returns the version of the module.

#### type Marshaler

    type Marshaler interface {
    	MarshalEnv() ([]string, error)
    }


Marshaler is the interface implemented by types that can marshal themselves into
valid object.

#### type Unmarshaler

    type Unmarshaler interface {
    	UnmarshalEnv() error
    }


Unmarshaler is the interface implements by types that can unmarshal an
environment variables of themselves.
