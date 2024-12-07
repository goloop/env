// Package env provides a comprehensive solution for managing environment
// variables in Go applications. It offers a rich set of features for handling
// environment configuration through both .env files and runtime environment
// variables.
//
// Core Features:
//   - Concurrent parsing of .env files with configurable parallelism
//   - Bidirectional mapping between environment variables and Go structures
//   - Support for nested structures and complex data types
//   - Advanced type conversion with validation
//   - URL parsing and validation support
//   - Flexible prefix-based filtering
//   - Custom marshaling and unmarshaling interfaces
//
// The package supports loading configuration from .env files with features like:
//   - Variable expansion (${VAR} or $VAR syntax)
//   - Quoted values with escape sequences
//   - Comments and inline comments
//   - Export statements
//   - Multi-line values
//   - Default values
//   - Custom separators for arrays/slices
//
// Type Support:
// The package handles all common Go types including:
//   - Basic types: string, bool, int/uint (all sizes), float32/64
//   - Complex types: url.URL, custom structs
//   - Collections: arrays, slices
//   - Nested structures with automatic prefix handling
//   - Pointers to supported types
//
// Structure Tags:
//   - env: specifies the environment variable name
//   - def: provides default values
//   - sep: defines separator for array/slice values
//
// Example usage:
//
//	type Config struct {
//	    Host    string   `env:"HOST" def:"localhost"`
//	    Port    int      `env:"PORT" def:"8080"`
//	    IPs     []string `env:"ALLOWED_IPS" sep:","`
//	    APIUrl  url.URL  `env:"API_URL"`
//	}
//
//	func main() {
//	    var cfg Config
//	    // Load .env file and parse it
//	    if err := env.Load(".env"); err != nil {
//	        log.Fatal(err)
//	    }
//	    // Map environment variables to structure
//	    if err := env.Unmarshal("", &cfg); err != nil {
//	        log.Fatal(err)
//	    }
//	}
//
// The package is designed to be efficient and safe, with careful handling of
// concurrent operations and proper error management. It provides a clean API
// that follows Go idioms while offering powerful features for complex
// configuration scenarios.
package env
