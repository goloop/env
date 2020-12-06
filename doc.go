/*
# env

The env module contains synonyms for the standard methods from the os module
to manage environment and implements various additional methods which allow
save environment variables to a GoLang structure.

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


## Quick start

There is a web-project that is develop and tests on the local computer and
deploys on the production server. On the production server some settings
(for example `host` and `port`) forced stored in an environment but on the
local computer data must be loaded from the file (because different team
members have different launch options).

For example, the local-configuration file `.env` contains:

    HOST=0.0.0.0
    PORT=8080
    ALLOWED_HOSTS="localhost:127.0.0.1"

    # In configurations file and/or in the environment there can be
    # a many of variables that willn't be parsed, like this:
    SECRET_KEY="N0XRABLZ5ZZY6dCNgD7pLjTIlx8v4G5d"

We need to load the missing configurations from the file into the environment.
Convert data from the environment into Go-structure. And use this
data to start the server.

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
*/
package env
