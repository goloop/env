package env

import (
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strings"
)

// Marshaler is the interface implemented by types that can marshal
// themselves into valid object.
type Marshaler interface {
	MarshalENV() ([]string, error)
}

// The marshalENV saves object's fields to environment.
//
// Method supports the following field's types: int, int8, int16, int32, int64,
// uin, uint8, uin16, uint32, in64, float32, float64, string, bool, url.URL
// and pointers, array or slice from thous types (i.e. *int, ...,
// []int, ..., []bool, ..., [2]*url.URL, etc.). The nested structures will be
// processed recursively.
//
// For other filed's types (like chan, map ...) will be returned an error.
func marshalENV(obj interface{}, pfx string) (result []string, err error) {
	// Note: convert *object to object and mean that we use reflection
	//       on the object but not a pointer on it.
	var rt, rv = reflect.TypeOf(obj), reflect.ValueOf(obj)
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
		rv = rv.Elem()
	}

	// The obj argument should be a initialized object.
	if rt.Kind() != reflect.Struct || !rv.IsValid() {
		err = errors.New("can't marshal non-object into environment")
		return
	}

	// Get a pointer to an object.
	ptr := reflect.New(rt)
	tmp := ptr.Elem()
	tmp.Set(rv)

	// Implements Marshaler interface.
	if ptr.Type().Implements(reflect.TypeOf((*Marshaler)(nil)).Elem()) {
		// Try to run custom MarshalENV function.
		if m := ptr.MethodByName("MarshalENV"); m.IsValid() {
			tmp := m.Call([]reflect.Value{})
			value := tmp[0].Interface()
			err := tmp[1].Interface()
			if err != nil {
				return []string{}, fmt.Errorf("marshal: %v", err)
			}
			return value.([]string), nil
		}
	}

	// Walk through the fields.
	result = make([]string, 0, rv.NumField())
	for i := 0; i < rv.NumField(); i++ {
		field := rt.Field(i)
		args := getTagArgs(field.Tag.Get("env"), field.Name)
		if !args.IsValid() {
			err = fmt.Errorf("invalid key name: %s", args.Key)
			return
		} else if args.IsIgnored() {
			continue
		}

		// Get item.
		item := rv.FieldByName(field.Name)
		if item.Kind() == reflect.Ptr {
			item = item.Elem()
		}

		switch item.Kind() {
		case reflect.Array, reflect.Slice:
			args.Value, err = getSequence(&item, args.Sep)
			if err != nil {
				return result, err
			}
		case reflect.Struct:
			// Support for url.URL struct only.
			if u, ok := item.Interface().(url.URL); ok {
				args.Value = u.String()
				break // break switch
			}

			// Another struct.
			p := fmt.Sprintf("%s%s_", pfx, args.Key)
			value, err := marshalENV(item.Interface(), p)
			if err != nil {
				return result, err
			}

			result = append(result, value...)
			continue // value of the recursive field is not to saved
		default:
			args.Value, err = toStr(item)
			if err != nil {
				return result, err
			}
		} // switch

		// Set into environment and add to result list.
		args.Key = fmt.Sprintf("%s%s", pfx, args.Key)
		err = Set(args.Key, args.Value)
		if err != nil {
			return result, err
		}
		result = append(result, fmt.Sprintf("%s=%s", args.Key, args.Value))
	} // for

	return result, nil
}

// getSequence get sequence as string.
func getSequence(item *reflect.Value, sep string) (string, error) {
	var (
		result string
		kind   reflect.Kind
		max    int
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

	// Item list string display.
	result = strings.Replace(fmt.Sprint(*item), " ", sep, -1)

	// For pointers and structures.
	if kind == reflect.Ptr || kind == reflect.Struct {
		var tmp = []string{}
		for i := 0; i < max; i++ {
			var elem = item.Index(i)
			if kind == reflect.Ptr {
				elem = item.Index(i).Elem()
			}

			v, err := toStr(elem)
			if err != nil {
				return "", err
			}

			tmp = append(tmp, v)
		}
		result = strings.Replace(fmt.Sprint(tmp), " ", sep, -1)
	}

	return strings.Trim(result, "[]"+sep), nil
}

// toStr converts item to string.
func toStr(item reflect.Value) (string, error) {
	var value string

	kind := item.Kind()
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16,
		reflect.Int32, reflect.Int64:
		value = fmt.Sprintf("%d", item.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16,
		reflect.Uint32, reflect.Uint64:
		value = fmt.Sprintf("%d", item.Uint())
	case reflect.Float32, reflect.Float64:
		value = fmt.Sprintf("%f", item.Float())
	case reflect.Bool:
		value = fmt.Sprintf("%t", item.Bool())
	case reflect.String:
		value = item.String()
	case reflect.Struct:
		// Support for url.URL struct only.
		if u, ok := item.Interface().(url.URL); ok {
			value = u.String()
			break
		}
		fallthrough
	default:
		return "", fmt.Errorf("incorrect type: %s", item.Type())
	}

	return value, nil
}
