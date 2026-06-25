package env

import (
	"os"
	"strings"
	"time"
)

// The settings holds the resolved options for a marshal/unmarshal call.
//
// The prefix is used verbatim by the engine (the public API normalizes it,
// appending "_" when needed); the separator is the default used for
// slices/arrays of fields that do not carry a sep tag; timeLayout is the
// default layout for time.Time fields without a layout tag.
type settings struct {
	prefix     string
	separator  string
	timeLayout string
	fileMode   os.FileMode
}

// The resolveLayout maps a layout name to a Go reference-time layout. It
// accepts the names of the standard time constants for convenience and falls
// back to treating the value as a literal layout. An empty value defaults to
// RFC3339.
func resolveLayout(name string) string {
	switch name {
	case "", "RFC3339":
		return time.RFC3339
	case "RFC3339Nano":
		return time.RFC3339Nano
	case "RFC1123":
		return time.RFC1123
	case "RFC1123Z":
		return time.RFC1123Z
	case "RFC822":
		return time.RFC822
	case "RFC822Z":
		return time.RFC822Z
	case "RFC850":
		return time.RFC850
	case "ANSIC":
		return time.ANSIC
	case "UnixDate":
		return time.UnixDate
	case "Kitchen":
		return time.Kitchen
	case "Stamp":
		return time.Stamp
	case "DateTime":
		return time.DateTime
	case "DateOnly":
		return time.DateOnly
	case "TimeOnly":
		return time.TimeOnly
	default:
		return name
	}
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
