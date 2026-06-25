package env

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// These helpers exist only for the internal test suite; they are not part of
// the package and used to live in the production files.

// The sts converts a slice or an array of any type to a string, inserting sep
// between the elements.
//
//	sts([]int{1, 2, 3}, ",")          // "1,2,3"
//	sts([]string{"1", "2", "3"}, ";") // "1;2;3"
func sts(seq any, sep string) (string, error) {
	var sb strings.Builder

	kind := reflect.TypeOf(seq).Kind()
	if kind != reflect.Array && kind != reflect.Slice {
		return "", errors.New("input is not a slice or array")
	}

	s := reflect.ValueOf(seq)
	for i := 0; i < s.Len(); i++ {
		if i > 0 {
			sb.WriteString(sep)
		}
		fmt.Fprintf(&sb, "%v", s.Index(i))
	}

	return sb.String(), nil
}

// The fts returns the value of a struct field as a string, looking it up by a
// key-like name (KEY_A is converted to the Go-style field name KeyA). It
// returns an empty string when the field is missing.
func fts(v any, name, sep string) string {
	r := reflect.ValueOf(v)
	if r.Kind() == reflect.Ptr {
		r = r.Elem()
	}
	if r.Kind() != reflect.Struct {
		return ""
	}

	// Convert a KEY_A-style name to the Go field name KeyA.
	if strings.Contains(name, "_") {
		words := strings.Fields(strings.ToLower(strings.ReplaceAll(name, "_", " ")))
		for i, w := range words {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
		name = strings.Join(words, "")
	}

	f := reflect.Indirect(r).FieldByName(name)
	if !f.IsValid() {
		return ""
	}

	var value string
	switch f.Kind() {
	case reflect.Slice, reflect.Array:
		if sep == "" {
			value = fmt.Sprintf("%v", f)
		} else {
			value = strings.Join(strings.Fields(fmt.Sprint(f)), sep)
		}
		value = strings.Trim(value, "[]")
	default:
		value = fmt.Sprintf("%v", f)
	}

	return value
}

// The unmarshalEnv is the v1-style internal wrapper kept for the test suite: it
// decodes the process environment into obj using a verbatim prefix.
func unmarshalEnv(prefix string, obj any) error {
	return decodeStruct(environMap(), obj, settings{
		prefix:    prefix,
		separator: defValueSep,
	})
}

// The marshalEnv is the v1-style internal wrapper kept for the test suite: it
// encodes obj, writes the pairs to the environment unless idle is true, and
// returns the produced "KEY=value" lines.
func marshalEnv(prefix string, obj any, idle bool) ([]string, error) {
	pairs, err := encodeStruct(obj, settings{prefix: prefix, separator: defValueSep})
	if err != nil {
		return nil, err
	}

	result := make([]string, 0, len(pairs))
	for _, p := range pairs {
		if !idle {
			if err := Set(p.key, p.value); err != nil {
				return result, err
			}
		}
		result = append(result, p.key+"="+p.value)
	}

	return result, nil
}
