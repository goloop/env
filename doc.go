// Package env provides a variety of methods for managing and streamlining
// environment variables. It supports loading data from .env files into the
// environment and offers seamless data transfer between the environment and
// custom data structures, allowing for effortless updates to structure fields
// from environment variables or vice versa, set environment variables from
// structure fields. The env package supports all standard data types, as well
// as the url.URL type.
package env

const version = "1.0.1"

// Version returns the version of the module.
func Version() string {
	return "v" + version
}
