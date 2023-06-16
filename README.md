[![Go Report Card](https://goreportcard.com/badge/github.com/goloop/env)](https://goreportcard.com/report/github.com/goloop/env) [![License](https://img.shields.io/badge/godoc-A+-brightgreen)](https://godoc.org/github.com/goloop/env) [![License](https://img.shields.io/badge/license-MIT-brightgreen)](https://github.com/goloop/env/blob/master/LICENSE) [![Stay with Ukraine](https://img.shields.io/static/v1?label=Stay%20with&message=Ukraine%20♥&color=ffd700&labelColor=007acc&style=flat)](https://u24.gov.ua/)

# env

The env package provides a variety of methods for managing environment variables. It supports loading data from `.env` files into the environment and provides data transfer between the environment and custom Go data structures, allowing you to effortlessly update structure fields from environment variables or vice versa, set environment variables from Go structure fields. The env package supports all standard Go data types (strings, numbers, boolean expressions, slices, arrays, etc.), as well as the complex `url.URL` type.

## Features

The main features of the env module include:

  - setting environment variables from variables defined in an env-file;
  - converting (marshaling) a Go structure to environment variables;
  - extracting (unmarshaling) environment variables to a Go structure;
  - setting any variables to the environment;
  - deleting variables from the environment;
  - checking for the presence of a variable in the environment;
  - retrieving the value of a variable by key from the environment;
  - clearing the environment.

In addition, additional methods for working with `.env` files and data exchange between environment variables and the Go structs are implemented.

The module provides synonyms for the standard methods from the os module, such as `Get` for `os.Getenv`, `Set` for `os.Setenv`, `Unset` for `os.Unsetenv`, `Clear` for `os.Clearenv`, and Environ for `os.Environ`, to manage the environment. Additionally, it implements custom methods that enable saving variables from the environment into structures.

## Env-file supports

The env package has full support for the syntax of `.env` files. Let's look at the basic rules for creating `.env` files supported by this package.

To demonstrate how the env package works, we'll use almost identical code, where we'll only change the names and types of the fields that need to be loaded from the environment and the formatting of the output. Therefore, for all the examples in this section, we will use the basic code:

```go
// Demonstration of the env package.
package main

import (
    "fmt"
    "log"
    "github.com/goloop/env"
)

// Environment is a structure for displaying data about the environment.
// In a real project, this can be a structure like Config, Settings, etc.
type Environment struct{}

func main() {
    // The prefix is used to filter the environment variables.
    var prefix string = ""

    // Load the environment variables from the .env file,
    // and update the environment variables.
    if err := env.Update(".env"); err != nil {
    	log.Fatal(err)
    }

    // Unmarshal the environment variables into a structure.
    // The object must be a pointer to a non-empty structure, so
    // the default fieldless Environment structure isn`t appropriate.
    e := Environment{}
    if err := env.Unmarshal(prefix, &e); err != nil {
    	log.Fatal(err)
    }

    // Print the structure.
    fmt.Printf("%v\n", e)
}
```

### Comments

Comments in the `.env` file begin with the `#` symbol. Anything after the `#` will not be treated as an environment variable.

```shell
# Comment as a separate line.
KEY_000="just a string here" # comment at the end of the expression
KEY_001="here the # symbol is part of the value" # value contains # symbol

# The file can contain empty lines to separate different blocks
# of variables by value, and comments that can occupy several lines.

KEY_002=33
```

This syntax is valid for the shell environment:

```
$ source .env
$ echo -e "$KEY_000\n$KEY_001\n$KEY_002"
just a string here
here the # symbol is part of the value
33
```

And is also valid for the env package:

```go
...
type Environment struct {
    Key000 string `env:"KEY_000"`
    Key001 string `env:"KEY_001"`
    Key002 int    `env:"KEY_002"`
}

...
func main() {
    ...
    fmt.Printf("%v\n%v\n%v", e.Key000, e.Key001, e.Key002)
    // Output:
    // just a string here
    // here the # symbol is part of the value
    // 33
}
```

### Variables

Each line in the `.env` file must contain only one environment variable declaration expression. Strings containing more than one environment variable may not be parsed correctly.

The example below will work correctly, but you shouldn't do it this way:

```shell
KEY_000=One
KEY_001=Two KEY_002=Three # don't do that
```

This syntax is valid for the shell environment:

```
$ source .env
$ echo -e "$KEY_000\n$KEY_001\n$KEY_002"
One
Two
Three
```

And is also valid for the env package:

```go
...
type Environment struct {
    Key000 string `env:"KEY_000"`
    Key001 string `env:"KEY_001"`
    Key002 string `env:"KEY_002"`
}

...
func main() {
    ...
    fmt.Printf("%v\n%v\n%v", e.Key000, e.Key001, e.Key002)
    // Output:
    // One
    // Two
    // Three
}
```


The environment variable must meet the following requirements:

  - use only symbols of the Latin alphabet;
  - use uppercase to declare a variable;
  - to separate the words characterizing the variable, use the underscore character `_`;
  - the name of the variable cannot begin with a number or other symbol other than the Latin alphabet;
  - the variable can be declared after the export command.

```shell
# Bad variable names:
# 010_KEY=5 # incorrect variable name
lower=true # unwanted variable name

# Good variable names:
HOST=127.0.0.1 # short variable name
ALLOWED_HOSTS=localhost,127.0.0.1 # multi-word variable name
export PORT=8080
```

This syntax is valid for the shell environment:

```
$ source .env
$ echo -e "$lower\n$HOST\n$ALLOWED_HOSTS\n$PORT"
true
127.0.0.1
localhost,127.0.0.1
8080
```

And is also valid for the env package:

```go
...
type Environment struct {
    // Key000    int      `env:"010_KEY"` # incorrect tag-name
    Lower        bool     `env:"lower"`
    Host         string   `env:"HOST"`
    Port         int      `env:"PORT"`
    AllowedHosts []string `env:"ALLOWED_HOSTS" sep:","`
}

...
func main() {
    ...
    fmt.Printf("%v\n%v\n%v\n%v", e.Lower, e.Host, e.AllowedHosts, e.Port)
    // Output:
    // true
    // 127.0.0.1
    // [localhost 127.0.0.1]
    // 8080
}
```

### Values

Rules for setting the value:
  - values are set after the `=` symbol;
  - if the value is a string that containing spaces, it must be enclosed in quotation marks.

```shell
USER=support # string without spaces
CREDO="Do it well and reuse it" # string contains spaces
AGE=35 # integer type
WEIGHT=105.3 # float type
LINKS=https://github.com/goloop,https://goloop.one
EMAIL=$USER@goloop.one # concatenation with variables: goloop@gmail.com
```

This syntax is valid for the shell environment:

```
$ source .env
$ echo -e "$USER\n$CREDO\n$AGE\n$WEIGHT\n$LINKS\n$EMAIL"
support
Do it well and reuse it
35
105.3
https://github.com/goloop,https://goloop.one
support@goloop.one
```

And is also valid for the env package:

```go
...
type Environment struct {
    User   string   `env:"USER"`
    Age    int      `env:"AGE"`
    Weight float32  `env:"WEIGHT"`
    Links  []string `env:"LINKS" sep:","`
    Email  string   `env:"EMAIL"`
}

...
func main() {
    ...
    fmt.Printf("%v\n%v\n%v\n%v\n%v", e.User, e.Age, e.Weight, e.Links, e.Email)
    // Output:
    // support
    // 35
    // 105.3
    // [https://github.com/goloop https://goloop.one]
    // support@goloop.one
}
```


### Prefixes

One `.env` file can contain configuration data of different projects with the same names. Prefixes should be used to select project-specific configuration parameters.

The following file contains settings for 3 services: `A`, `B`, `C`:

```shell
# Project A.
PROJECT_A_HOST=192.168.0.1
PROJECT_A_PORT=8081
PROJECT_A_USER=john

# Project B.
PROJECT_B_HOST=192.168.0.2
PROJECT_B_PORT=8082
PROJECT_B_USER=bob

# Project C.
PROJECT_C_HOST=192.168.0.3
PROJECT_C_PORT=8083
PROJECT_C_USER=alan
```

For example, let's load project `B` data.

Pay attention:

  - in the data structure, we do not specify the prefix, only the name of the variable;
  - use the `Unmarshal` method by passing the value of the prefix to it as the first argument.

```go
...
type Environment struct {
    Host string `env:"HOST"`
    Port int    `env:"PORT"`
    User string `env:"USER"`
}
...

func main() {
    var prefix string = "PROJECT_B_"
    ...

    fmt.Printf("%v\n%v\n%v", e.Host, e.Port, e.User)
    // Output:
    // 192.168.0.2
    // 8082
    // bob
}
```

### Nested objects

Data can be grouped into custom structures. For example:

```shell
# Project A.
PROJECT_A_USERNAME=john
PROJECT_A_PASSWORD=527bd5b5d689e2c32ae974c6229ff785
PROJECT_A_USER_NAME="John Smith"
PROJECT_A_USER_EMAIL=$PROJECT_A_USERNAME@gmail.com
PROJECT_A_USER_RIGHTS_IS_STAFF=true
PROJECT_A_USER_RIGHTS_IS_SUPERUSER=false

# Project B.
PROJECT_B_USERNAME=bob
PROJECT_B_PASSWORD=9f9d51bc70ef21ca5c14f307980a29d8
PROJECT_B_USER_NAME="Bob Dahn"
PROJECT_B_USER_EMAIL=$PROJECT_A_USERNAME@gmail.com
PROJECT_B_USER_RIGHTS_IS_STAFF=true
PROJECT_B_USER_RIGHTS_IS_SUPERUSER=true
```

For example, let's load project `A` data.

```go
...
type Rights struct {
    IsStaff     bool `env:"IS_STAFF"`
    IsSuperuser bool `env:"IS_SUPERUSER"`
}

type User struct {
    Name   string `env:"NAME"`
    Email  string `env:"EMAIL"`
    Rights Rights `env:"RIGHTS"`
}

type Environment struct {
    Username string `env:"USERNAME"`
    Password string `env:"PASSWORD"`
    User     User   `env:"USER"`
}
...

func main() {
    var prefix string = "PROJECT_A_"
    ...

    fmt.Printf("%v\n%v\n", e.Username, e.Password)
    fmt.Printf("%v\n%v\n", e.User.Name, e.User.Email)
    fmt.Printf("%v\n%v\n", e.User.Rights.IsStaff, e.User.Rights.IsSuperuser)
    // Output:
    // john
    // 527bd5b5d689e2c32ae974c6229ff785
    // John Smith
    // john@gmail.com
    // true
    // false
}
```

### Generalization

The env-file can contain any environment valid constructions, for example:

```shell
# Supported:
# .................................................................
# Comments as separate entries.
KEY_000="value for key 000" # comment as part of the var declaration
export KEY_001="value for key 001" # with export command
KEY_002=one:two:three # list with default separators, colon
KEY_003=one,two,three # ... or comma separator, or etc...
KEY_004=7 # numbers
KEY_005=John # string without quotes
PREFIX_KEY_005="Bob" # prefix filtering

# Empty line (up/down).

KEY_006=one,"two,three",four # grouped values as: one|two,three|four
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

There are several ways of using this package.

  - transfer variables from env-files to the environment;
  - transfer data from the environment into Go structures;
  - saving Go structure's fields to the environment;
  - saving Go structure's fields to the env files.

### Parsing env files

Parsing of env-files takes place in concurrency mode, runtime.NumCPU() is used by default for the number of goroutines.

To change the number of goroutines you need to use the `ParallelTasks` method.


There are several methods for parsing env files:

  - `Load` loads new keys only;
  - `LoadSafe` loads new keys only and doesn't handles variables like `${var}` or `$var` - doesn't turn them into a finite value;
  - `Update` loads keys from the env-file into environment, update existing keys;
  - `UpdateSafe` loads keys from the env-file into environment, update existing keys, and doesn't handles variables like `${var}` or `$var` - doesn't turn them into a finite value.

The `Update` function works like the `source` command in UNIX-Like operating systems.



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

 - env - matches the name of the key in the environment;
 - def - default value (if empty, sets the default value for the field type of structure);
 - sep - sets the separator for lists/arrays (default ` ` - space).

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
	//  Server started on 0.0.0.0:8080
	//  Allowed hosts: [localhost 127.0.0.1]
}
```
