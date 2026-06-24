package env

import (
	"os"
	"strings"
)

// The settings holds the resolved options for a marshal/unmarshal call.
//
// The prefix is used verbatim by the engine (the public API normalizes it,
// appending "_" when needed); the separator is the default used for
// slices/arrays of fields that do not carry a sep tag.
type settings struct {
	prefix    string
	separator string
}

// The environMap snapshots the process environment as a key/value map.
func environMap() map[string]string {
	env := os.Environ()
	m := make(map[string]string, len(env))
	for _, kv := range env {
		if i := strings.IndexByte(kv, '='); i >= 0 {
			m[kv[:i]] = kv[i+1:]
		}
	}

	return m
}
