package env

import (
	"net/url"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"
)

// fieldInfo is the call-independent description of one struct field, derived
// from its tags. It is cached per struct type so the tags are parsed once.
type fieldInfo struct {
	index     int
	name      string // key name (env tag, or field name)
	goName    string // Go field name (for error messages)
	sepTag    string // sep tag, "" if absent
	layoutTag string // layout tag, "" if absent
	def       string // def tag (default value)
	required  bool
}

// fieldCache maps a struct reflect.Type to its cached []fieldInfo.
var fieldCache sync.Map

// The cachedFields returns the decoded field descriptors for the struct type t,
// computing them once and reusing them on later calls (like encoding/json's
// field cache). Unexported and env:"-" fields are excluded.
func cachedFields(t reflect.Type) []fieldInfo {
	if v, ok := fieldCache.Load(t); ok {
		return v.([]fieldInfo)
	}

	fields := make([]fieldInfo, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" {
			continue // unexported, like encoding/json
		}

		name, required := parseEnvTag(f.Tag.Get(tagNameKey))
		if name == defValueIgnored {
			continue // env:"-"
		}
		if name == "" {
			name = f.Name
		}

		fields = append(fields, fieldInfo{
			index:     i,
			name:      name,
			goName:    f.Name,
			sepTag:    f.Tag.Get(tagNameSep),
			layoutTag: f.Tag.Get(tagNameLayout),
			def:       f.Tag.Get(tagNameValue),
			required:  required,
		})
	}

	v, _ := fieldCache.LoadOrStore(t, fields)
	return v.([]fieldInfo)
}

// Cached reflect.Types for the special-cased field types (avoids recomputing
// them on every field/element in the hot path).
var (
	timeDurationType = reflect.TypeOf(time.Duration(0))
	timeTimeType     = reflect.TypeOf(time.Time{})
	timeTimePtrType  = reflect.TypeOf((*time.Time)(nil))
	urlType          = reflect.TypeOf(url.URL{})
	urlPtrType       = reflect.TypeOf((*url.URL)(nil))
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
	requireAll bool // WithRequiredAll: every leaf field must be present

	// Custom per-type parsers/encoders registered with WithParser/WithEncoder.
	// Both maps are nil unless an option registers a type (a nil-map read is
	// safe, so the hot path costs only a lookup).
	parsers  map[reflect.Type]func(string) (reflect.Value, error)
	encoders map[reflect.Type]func(reflect.Value) (string, error)
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
