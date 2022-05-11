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

// Expand is synonym for the os.Expand, replaces ${var} or $var in the
// string according to the values of the current environment variables.
// References to undefined variables are replaced by the empty string.
func Expand(value string) string {
	return os.Expand(value, os.Getenv)
}

// Lookup is synonym for the os.LookupEnv, retrieves the value of
// the environment variable named by the key. If the variable is
// present in the environment the value (which may be empty) is
// returned and the boolean is true. Otherwise the returned
// value will be empty and the boolean will be false.
func Lookup(key string) (string, bool) {
	return os.LookupEnv(key)
}
