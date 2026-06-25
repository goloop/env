package env

import (
	"fmt"
	"math"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

// Unmarshaler is the interface implemented by types that can unmarshal
// themselves from a set of environment values. The data map holds the
// already-resolved (expanded) key/value pairs of the source.
type Unmarshaler interface {
	UnmarshalEnv(data map[string]string) error
}

// The validateStruct checks whether the object is a pointer to the structure,
// and returns reflect.Type and reflect.Value of the object. If the object is
// not a pointer to the structure or object is nil, it returns an error.
func validateStruct(obj any) (reflect.Type, reflect.Value, error) {
	rt, rv, err := reflect.TypeOf(obj), reflect.ValueOf(obj), error(nil)

	// Check object type
	// Object should be a pointer to a non-empty struct.
	if obj == nil {
		err = ErrNilObject
	} else if rv.Kind() != reflect.Ptr || rv.IsNil() {
		err = ErrNotPointer
	} else if rv.Type().Elem().Kind() != reflect.Struct {
		err = ErrNotStruct
	} else if rv.Elem().NumField() == 0 {
		err = ErrEmptyStruct
	}

	return rt, rv, err
}

// The unmarshalEnv read variables from the environment
// and save them into Go-struct.
//
// Method supports the following type of the fields: int, int8, int16, int32,
// int64, uin, uint8, uin16, uint32, in64, float32, float64, string, bool,
// struct, url.URL and pointers, array or slice from types like (i.e. *int,
// *uint, ..., []int, ..., []bool, ..., [2]*url.URL, etc.). The fields as
// a struct or pointer on the struct will be processed recursively.
//
// For other type of the fields (i.e chan, map ...) or upon occurrence other
// conversion problems will be returned an error.
//
// The prefix argument filters keys by a certain prefix and used as a marker
// of the nesting level during the recursive processing of object fields
// (as prefix for environment variables).
//
// The obj is a pointer to an initialized object where need to
// save variables from the environment.
func unmarshalEnv(prefix string, obj any) error {
	return decodeStruct(environMap(), obj, settings{
		prefix:    prefix,
		separator: defValueSep,
	})
}

// The decodeStruct reads values from the source map into the fields of obj,
// honouring the prefix and the default separator from s. Nested structs are
// processed recursively with the parent key as a prefix.
func decodeStruct(source map[string]string, obj any, s settings) error {
	t, v, err := validateStruct(obj)
	if err != nil {
		return err
	}

	// If objects implements Unmarshaler interface
	// try to calling a custom Unmarshal method.
	if unmarshaler, ok := obj.(Unmarshaler); ok {
		return unmarshaler.UnmarshalEnv(source)
	}

	// Walk through all the fields of the structure
	// and save data from the source.
	e := v.Elem()
	for i := 0; i < e.NumField(); i++ {
		field := t.Elem().Field(i)

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
			key:   s.prefix + key,
			value: field.Tag.Get(tagNameValue),
			sep:   sep,
		}

		if !tg.isValid() {
			return fmt.Errorf(
				"the %s field does not have a valid key name value: %s",
				field.Name,
				tg.key,
			)
		}

		// If the key exists - take its value from the source.
		if value, ok := source[tg.key]; ok {
			tg.value = value
		}

		// Set value to field.
		item := e.Field(i)
		if err := setFieldValue(source, &item, tg, s); err != nil {
			return err
		}
	}

	return nil
}

// The setFieldValue sets value to field from the tag arguments. The source
// and s are threaded through for the recursive decoding of nested structs.
func setFieldValue(source map[string]string, item *reflect.Value, tg *tagGroup, s settings) error {
	switch item.Kind() {
	case reflect.Array:
		max := item.Type().Len()
		seq := splitN(tg.value, tg.sep, -1)
		if len(seq) > max {
			return fmt.Errorf("%d overflows the [%d]array", len(seq), max)
		}

		if err := setSequence(item, seq); err != nil {
			return err
		}
	case reflect.Slice:
		seq := splitN(tg.value, tg.sep, -1)
		tmp := reflect.MakeSlice(item.Type(), len(seq), len(seq))
		if err := setSequence(&tmp, seq); err != nil {
			return err
		}

		item.Set(reflect.AppendSlice(*item, tmp))
	case reflect.Ptr:
		if item.Type().Elem().Kind() != reflect.Struct {
			// If the pointer of a structure.
			tmp := reflect.Indirect(*item)
			if err := setValue(tmp, tg.value); err != nil {
				return err
			}
			break
		} else if item.Type() == reflect.TypeOf((*url.URL)(nil)) {
			// If a pointer of a url.URL structure.
			if err := setValue(*item, tg.value); err != nil {
				return err
			}
			break
		}

		// If a pointer to a structure of the another's types (not a *url.URL).
		// Perform recursive analysis of nested structure fields.
		tmp := reflect.New(item.Type().Elem()).Interface()
		child := settings{prefix: tg.key + "_", separator: s.separator}
		if err := decodeStruct(source, tmp, child); err != nil {
			return err
		}

		item.Set(reflect.ValueOf(tmp))
	case reflect.Struct:
		if item.Type() == reflect.TypeOf(url.URL{}) {
			// If a url.URL structure.
			if err := setValue(*item, tg.value); err != nil {
				return err
			}
			break
		}

		// If a structure of the another's types (not a url.URL).
		// Perform recursive analysis of nested structure fields.
		tmp := reflect.New(item.Type()).Interface()
		child := settings{prefix: tg.key + "_", separator: s.separator}
		if err := decodeStruct(source, tmp, child); err != nil {
			return err
		}

		item.Set(reflect.ValueOf(tmp).Elem())
	default:
		// Try to set correct value.
		if err := setValue(*item, tg.value); err != nil {
			return err
		}
	}

	return nil
}

// The setSequence sets slice into item, if item is slice or array.
func setSequence(item *reflect.Value, seq []string) error {
	// Ignore empty sequences.
	if len(seq) == 0 || item.Len() == 0 {
		return nil
	}

	// Set values to the sequence.
	for i, value := range seq {
		elem := item.Index(i)
		if !elem.CanSet() {
			return fmt.Errorf("cannot set value %s at index %d", value, i)
		}
		if err := setValue(elem, value); err != nil {
			return err
		}
	}

	return nil
}

// The setValue sets value into item (field of the struct).
func setValue(item reflect.Value, value string) error {
	kind := item.Kind()

	// The *url.URL pointer only.
	if kind == reflect.Ptr && item.Type() == reflect.TypeOf((*url.URL)(nil)) {
		u, err := url.Parse(value)
		if err != nil {
			return err
		}
		item.Set(reflect.ValueOf(u))
		return nil
	}

	// The url.URL struct only.
	if kind == reflect.Struct && item.Type() == reflect.TypeOf(url.URL{}) {
		u, err := url.Parse(value)
		if err != nil {
			return err
		}
		item.Set(reflect.ValueOf(*u))
		return nil
	}

	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16,
		reflect.Int32, reflect.Int64:
		r, err := strToIntKind(value, kind)
		if err != nil {
			return err
		}
		item.SetInt(r)
	case reflect.Uint, reflect.Uint8, reflect.Uint16,
		reflect.Uint32, reflect.Uint64:
		r, err := strToUintKind(value, kind)
		if err != nil {
			return err
		}
		item.SetUint(r)
	case reflect.Float32, reflect.Float64:
		r, err := strToFloatKind(value, kind)
		if err != nil {
			return err
		}
		item.SetFloat(r)
	case reflect.Bool:
		r, err := strToBool(value)
		if err != nil {
			return err
		}
		item.SetBool(r)
	case reflect.String:
		item.SetString(value)
	default:
		return fmt.Errorf("incorrect type: %s", item.Type())
	}

	return nil
}

// The strToIntKind converts string to int64 type with out-of-range checking
// for int. Returns 0 if value is empty.
func strToIntKind(value string, kind reflect.Kind) (int64, error) {
	var min, max int64

	// For empty string returns zero.
	if len(value) == 0 {
		return 0, nil
	}

	// Convert string to int64.
	r, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, err
	}

	// Out of range checking.
	switch kind {
	case reflect.Int:
		if strconv.IntSize == 32 {
			min, max = math.MinInt32, math.MaxInt32
		} else {
			min, max = math.MinInt64, math.MaxInt64
		}
		if r < min || r > max {
			s := strconv.IntSize
			return 0, fmt.Errorf("%d is out of range for int (%d-bit)", r, s)
		}
	case reflect.Int8:
		min, max = math.MinInt8, math.MaxInt8
	case reflect.Int16:
		min, max = math.MinInt16, math.MaxInt16
	case reflect.Int32:
		min, max = math.MinInt32, math.MaxInt32
	case reflect.Int64:
		min, max = math.MinInt64, math.MaxInt64
	default:
		return 0, fmt.Errorf("incorrect kind %v", kind)
	}

	if kind != reflect.Int && (r < min || r > max) {
		return 0, fmt.Errorf("%d is out of range for %v", r, kind)
	}

	return r, nil
}

// The strToUintKind convert string to uint64 type with out-of-range checking
// for uint. Returns 0 if value is empty.
func strToUintKind(value string, kind reflect.Kind) (uint64, error) {
	var max uint64

	// For empty string returns zero.
	if len(value) == 0 {
		return 0, nil
	}

	// Convert string to uint64.
	r, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, err
	}

	// Out of range checking.
	switch kind {
	case reflect.Uint:
		if strconv.IntSize == 32 {
			max = math.MaxUint32
		} else {
			max = math.MaxUint64
		}
		if r > max {
			s := strconv.IntSize
			return 0, fmt.Errorf("%d is out of range for uint (%d-bit)", r, s)
		}
	case reflect.Uint8:
		max = math.MaxUint8
	case reflect.Uint16:
		max = math.MaxUint16
	case reflect.Uint32:
		max = math.MaxUint32
	case reflect.Uint64:
		max = math.MaxUint64
	default:
		return 0, fmt.Errorf("incorrect kind %v", kind)
	}

	if kind != reflect.Uint && r > max {
		return 0, fmt.Errorf("%d is out of range for %v", r, kind)
	}

	return r, nil
}

// The strToFloatKind converts a string to float64 with out-of-range
// checking for float. Returns 0 if value is empty.
func strToFloatKind(value string, kind reflect.Kind) (float64, error) {
	var min, max float64

	// For empty string returns zero.
	if len(value) == 0 {
		return 0.0, nil
	}

	// Convert string to Float64.
	r, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0.0, err
	}

	// Out of range checking.
	switch kind {
	case reflect.Float32:
		min, max = -math.MaxFloat32, math.MaxFloat32
	case reflect.Float64:
		min, max = -math.MaxFloat64, math.MaxFloat64
	default:
		return 0.0, fmt.Errorf("incorrect kind %v", kind)
	}

	if r < min || r > max {
		return 0.0, fmt.Errorf("%f is out of range for %v", r, kind)
	}

	return r, nil
}

// The strToBool convert string to bool type.
// Returns false if value is empty.
func strToBool(v string) (bool, error) {
	// For empty string returns false.
	if len(v) == 0 {
		return false, nil
	}

	// Try to convert string to bool.
	// It accepts 1, t, T, TRUE, true, True, 0, f, F, FALSE, false, False.
	r, err := strconv.ParseBool(v)
	if err == nil {
		return r, nil
	}

	// If strconv.ParseBool() fails, try to parse as a float and check if the
	// absolute value is greater than 0.7.
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return false, fmt.Errorf("'%s' cannot be converted to a boolean", v)
	}

	return math.Abs(f) > 0.7, nil
}
