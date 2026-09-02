package env

import "os"

// Get is synonym for the os.Getenv, retrieves the value of the environment
// variable named by the key. It returns the value, which will be empty if
// the variable is not present.
//
// To distinguish between an empty value and an unset value, use Lookup.
func Get(key string) string {
	return os.Getenv(key)
}

// Set is synonym for the os.Setenv, sets the value of the environment
// variable named by the key. It returns an error, if any.
func Set(key, value string) error {
	return os.Setenv(key, value)
}

// Unset is synonym for the os.Unsetenv, unsets a single environment variable.
func Unset(key string) error {
	return os.Unsetenv(key)
}

// Clear is synonym for the os.Clearenv, deletes all environment variables.
func Clear() {
	os.Clearenv()
}

// Environ is synonym for the os.Environ, returns a copy of strings
// representing the environment, in the form "key=value".
func Environ() []string {
	return os.Environ()
}

// Expand replaces ${NAME} and $NAME in the string with the value of the
// matching environment variable, the same way the loaders in this
// package do. A reference to a variable that is not set becomes empty.
//
// A reference is recognised only when NAME is a name this format could
// have defined, that is [A-Za-z_][A-Za-z0-9_]*. Anything else keeps its
// "$": a price ("cost: $100"), a one-liner ("$1 == x") and a password
// ("pa$$word") are values, not substitutions. That is the one place this
// differs from os.Expand, which reads "$1" as the shell's first
// positional parameter and would leave "cost: 00" behind.
func Expand(value string) string {
	return expandVars(value, os.Getenv)
}

// Lookup is synonym for the [os.LookupEnv], retrieves the value of
// the environment variable named by the key. If the variable is
// present in the environment the value (which may be empty) is
// returned and the boolean is true. Otherwise the returned
// value will be empty and the boolean will be false.
func Lookup(key string) (string, bool) {
	return os.LookupEnv(key)
}
