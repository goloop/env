package env

import (
	"fmt"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// Marshaler is the interface implemented by types that can marshal themselves
// into a set of environment values. The returned map holds the key/value pairs;
// the library decides where they go (environment, map or file).
type Marshaler interface {
	MarshalEnv() (map[string]string, error)
}

// The pair is a single key/value entry produced by encoding a struct.
type pair struct {
	key   string
	value string
}

// The marshalEnv encodes obj and writes the result into the environment
// (unless idle is true), returning the produced "KEY=value" lines. It is the
// internal entry point kept for compatibility with the existing test suite.
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

// The encodeStruct converts the fields of obj into an ordered list of key/value
// pairs, honouring the prefix and default separator from s. It has no side
// effects: callers decide whether to write the pairs to the environment, a map
// or a file.
//
// If obj implements Marshaler, its MarshalEnv result is returned with keys
// sorted for a deterministic order.
func encodeStruct(obj any, s settings) ([]pair, error) {
	// Convert *object to object and mean that we use
	// reflection on the object but not a pointer on it.
	rt, rv := reflect.TypeOf(obj), reflect.ValueOf(obj)
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
		rv = rv.Elem()
	}

	// The obj argument should be a initialized object.
	if rt.Kind() != reflect.Struct || !rv.IsValid() {
		return nil, ErrInvalidObject
	}

	// Get a pointer to the object.
	ptr := reflect.New(rt)
	ptr.Elem().Set(rv)

	// Implements Marshaler interface.
	if ptr.Type().Implements(reflect.TypeOf((*Marshaler)(nil)).Elem()) {
		// Try to run custom MarshalEnv function.
		if m := ptr.MethodByName("MarshalEnv"); m.IsValid() {
			tmp := m.Call([]reflect.Value{}) // len == 2
			if err, _ := tmp[1].Interface().(error); err != nil {
				return nil, fmt.Errorf("custom marshal method: %w", err)
			}

			return mapToPairs(tmp[0].Interface().(map[string]string)), nil
		}
	}

	// Walk through the fields.
	result := make([]pair, 0, rv.NumField())
	for i := 0; i < rv.NumField(); i++ {
		field := rt.Field(i)

		// Get parameters from tags.
		// The name of the key.
		key := strings.TrimSpace(field.Tag.Get(tagNameKey))
		if key == "" {
			key = field.Name
		}

		// Separator value for slices/arrays.
		sep := field.Tag.Get(tagNameSep)
		if sep == "" {
			sep = s.separator
		}

		// Create tag group.
		tg := &tagGroup{
			key:   key,
			value: field.Tag.Get(tagNameValue),
			sep:   sep,
		}

		if !tg.isValid() {
			return nil, fmt.Errorf(
				"the %s field does not have a valid key name value: %s",
				field.Name,
				tg.key,
			)
		}

		// Get item.
		item := rv.Field(i)
		if item.Kind() == reflect.Ptr {
			item = item.Elem()
		}

		switch item.Kind() {
		case reflect.Array, reflect.Slice:
			value, err := getSequence(&item, tg.sep)
			if err != nil {
				return nil, err
			}
			tg.value = value
		case reflect.Struct:
			// Support for url.URL struct.
			if u, ok := item.Interface().(url.URL); ok {
				tg.value = u.String()
				break // break switch
			}

			// Another struct.
			// Recursive analysis of the nested structure.
			child := settings{prefix: s.prefix + tg.key + "_", separator: s.separator}
			nested, err := encodeStruct(item.Interface(), child)
			if err != nil {
				return nil, err
			}

			result = append(result, nested...)
			continue // nested struct contributes its own pairs
		default:
			value, err := toStr(item)
			if err != nil {
				return nil, err
			}
			tg.value = value
		} // switch

		result = append(result, pair{key: s.prefix + tg.key, value: tg.value})
	} // for

	return result, nil
}

// The mapToPairs converts a map into a slice of pairs sorted by key so the
// output order is deterministic (used for custom Marshaler results).
func mapToPairs(m map[string]string) []pair {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	pairs := make([]pair, 0, len(m))
	for _, k := range keys {
		pairs = append(pairs, pair{key: k, value: m[k]})
	}

	return pairs
}

// The getSequence get sequence as string.
func getSequence(item *reflect.Value, sep string) (string, error) {
	var (
		kind reflect.Kind
		max  int
	)

	// Type checking and instance adjustment.
	switch item.Kind() {
	case reflect.Array:
		kind = item.Index(0).Kind()
		max = item.Type().Len()
	case reflect.Slice:
		tmp := reflect.MakeSlice(item.Type(), 1, 1)
		kind = tmp.Index(0).Kind()
		max = item.Len()
	default:
		return "", fmt.Errorf("incorrect type: %s", item.Type())
	}

	// Use strings.Builder for efficient string concatenation.
	var sb strings.Builder

	// For pointers and structures.
	if kind == reflect.Ptr || kind == reflect.Struct {
		for i := 0; i < max; i++ {
			elem := item.Index(i)
			if kind == reflect.Ptr {
				elem = item.Index(i).Elem()
			}

			v, err := toStr(elem)
			if err != nil {
				return "", err
			}

			if i > 0 {
				sb.WriteString(sep)
			}
			sb.WriteString(v)
		}
	} else {
		for i := 0; i < max; i++ {
			v, err := toStr(item.Index(i))
			if err != nil {
				return "", err
			}

			if i > 0 {
				sb.WriteString(sep)
			}
			sb.WriteString(v)
		}
	}

	return sb.String(), nil
}

// The toStr converts any item to string.
func toStr(item reflect.Value) (string, error) {
	switch item.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16,
		reflect.Int32, reflect.Int64:
		return strconv.FormatInt(item.Int(), 10), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16,
		reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(item.Uint(), 10), nil
	case reflect.Float32, reflect.Float64:
		// Use the shortest representation that round-trips (`%f` forced
		// 6 decimals and broke the round-trip: 3.14 -> "3.140000").
		bitSize := 64
		if item.Kind() == reflect.Float32 {
			bitSize = 32
		}
		return strconv.FormatFloat(item.Float(), 'g', -1, bitSize), nil
	case reflect.Bool:
		return strconv.FormatBool(item.Bool()), nil
	case reflect.String:
		return item.String(), nil
	case reflect.Struct:
		// Support for url.URL struct only.
		if u, ok := item.Interface().(url.URL); ok {
			return u.String(), nil
		}
	}

	return "", fmt.Errorf("incorrect type: %s", item.Type())
}
