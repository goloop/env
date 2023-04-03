package env

import (
	"errors"
	"fmt"
	"math"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
)

// Unmarshaler is the interface implements by types that can
// unmarshal an environment variables of themselves.
type Unmarshaler interface {
	UnmarshalEnv() error
}

// The validateStruct checks whether the object is a pointer to the structure,
// and returns reflect.Type and reflect.Value of the object. If the object is
// not a pointer to the structure or object is nil, it returns an error.
func validateStruct(obj interface{}) (reflect.Type, reflect.Value, error) {
	t, v, err := reflect.TypeOf(obj), reflect.ValueOf(obj), error(nil)

	switch {
	case obj == nil:
		// Object is nil.
		err = errors.New("obj is nil")
	case v.Kind() != reflect.Ptr || v.IsNil():
		// Object is not a pointer or pointer is nil.
		err = errors.New("obj should be a non-nil pointer to a struct")
	case v.Type().Elem().Kind() != reflect.Struct:
		// Object is not a pointer to a struct.
		err = errors.New("obj should be a pointer to a struct")
	case v.Elem().NumField() == 0:
		// Object is a pointer to an empty struct (without fields).
		err = errors.New("obj should be a pointer to a non-empty struct")
	}

	return t, v, err
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
func unmarshalEnv(prefix string, obj interface{}) error {
	t, v, err := validateStruct(obj)
	if err != nil {
		return err
	}

	// If objects implements Unmarshaler interface
	// try to calling a custom Unmarshal method.
	if v.Type().Implements(reflect.TypeOf((*Unmarshaler)(nil)).Elem()) {
		if m := v.MethodByName("UnmarshalEnv"); m.IsValid() {
			tmp := m.Call([]reflect.Value{}) // len == 1
			if err := tmp[0].Interface(); err != nil {
				return fmt.Errorf("%v", err)
			}
			return nil
		}
	}

	// Note: It makes no sense to execute the following code in goroutines,
	// because the environment variables are global and the access to them
	// is not thread-safe.

	// Walk through all the fields of the structure
	// and save data from the environment.
	e := v.Elem()
	for i := 0; i < e.NumField(); i++ {
		field := t.Elem().Field(i)

		// Get parameters from tags.
		// The name of the key.
		key := strings.Trim(field.Tag.Get(tagNameKey), " ")
		if key == "" {
			key = field.Name
		}

		// Separator value for slices/arrays.
		sep := field.Tag.Get(tagNameSep)
		if sep == "" {
			sep = defValueSep
		}

		// Create tag group.
		tg := &tagGroup{
			key:   fmt.Sprintf("%s%s", prefix, key),
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

		// If the key exists - take its value from environment.
		if value, ok := os.LookupEnv(tg.key); ok {
			tg.value = value
		}

		// Set value to field.
		item := e.FieldByName(field.Name)
		if err := setFieldValue(&item, tg); err != nil {
			return err
		}
	}

	return nil
}

// The setFieldValue sets value to field from the tag arguments.
func setFieldValue(item *reflect.Value, tg *tagGroup) error {
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
		if err := unmarshalEnv(fmt.Sprintf("%s_", tg.key), tmp); err != nil {
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
		if err := unmarshalEnv(fmt.Sprintf("%s_", tg.key), tmp); err != nil {
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
	if len(seq) == 0 ||
		(item.Index(0).Kind() == reflect.Array && item.Type().Len() == 0) ||
		(item.Index(0).Kind() == reflect.Slice && item.Len() == 0) {
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
	case reflect.Ptr:
		// The *url.URL pointer only.
		if item.Type() != reflect.TypeOf((*url.URL)(nil)) {
			return fmt.Errorf("incorrect type: %s", item.Type())
		}

		u, err := url.Parse(value)
		if err != nil {
			return err
		}
		item.Set(reflect.ValueOf(u))
	case reflect.Struct:
		// The url.URL struct only.
		if item.Type() != reflect.TypeOf(url.URL{}) {
			return fmt.Errorf("incorrect type: %s", item.Type())
		}

		u, err := url.Parse(value)
		if err != nil {
			return err
		}
		item.Set(reflect.ValueOf(*u))
	default:
		return fmt.Errorf("incorrect type: %s", item.Type())
	}

	return nil
}

// The strToIntKind converts string to int64 type with out-of-range checking
// for int. Returns 0 if value is empty.
func strToIntKind(value string, kind reflect.Kind) (r int64, err error) {
	// For empty string returns zero.
	if len(value) == 0 {
		return 0, nil
	}

	// Convert string to int64.
	r, err = strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, err
	}

	switch kind {
	case reflect.Int:
		if strconv.IntSize == 32 {
			// For 32-bit platform it is necessary to check overflow.
			if r < math.MinInt32 || r > math.MaxInt32 {
				return 0, fmt.Errorf("%d is out of range for int (int32)", r)
			}
		} else {
			// For 64-bit platform it is necessary to check overflow.
			if r < math.MinInt64 || r > math.MaxInt64 {
				return 0, fmt.Errorf("%d is out of range for int (int64)", r)
			}
		}
	case reflect.Int8:
		if r < math.MinInt8 || r > math.MaxInt8 {
			return 0, fmt.Errorf("%d is out of range for int8", r)
		}
	case reflect.Int16:
		if r < math.MinInt16 || r > math.MaxInt16 {
			return 0, fmt.Errorf("%d is out of range for int16", r)
		}
	case reflect.Int32:
		if r < math.MinInt32 || r > math.MaxInt32 {
			return 0, fmt.Errorf("%d is out of range for int32", r)
		}
	case reflect.Int64:
		if r < math.MinInt64 || r > math.MaxInt64 {
			return 0, fmt.Errorf("%d is out of range for int64", r)
		}
	default:
		r, err = 0, fmt.Errorf("incorrect kind %v", kind)
	}

	return
}

// The strToUintKind convert string to uint64 type with out-of-range checking
// for uint. Returns 0 if value is empty.
func strToUintKind(value string, kind reflect.Kind) (r uint64, err error) {
	// For empty string returns zero.
	if len(value) == 0 {
		return 0, nil
	}

	// Convert string to uint64.
	r, err = strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, err
	}

	switch kind {
	case reflect.Uint:
		// For 32-bit platform it is necessary to check overflow.
		if strconv.IntSize == 32 {
			if r > math.MaxUint32 {
				return 0, fmt.Errorf("%d is out of range for uint (uint32)", r)
			}
		} else {
			if r > math.MaxUint64 {
				return 0, fmt.Errorf("%d is out of range for uint (uint64)", r)
			}
		}
	case reflect.Uint8:
		if r > math.MaxUint8 {
			return 0, fmt.Errorf("%d is out of range for uint8", r)
		}
	case reflect.Uint16:
		if r > math.MaxUint16 {
			return 0, fmt.Errorf("%d is out of range for uint16", r)
		}
	case reflect.Uint32:
		if r > math.MaxUint32 {
			return 0, fmt.Errorf("%d is out of range for uint32", r)
		}
	case reflect.Uint64:
		if r > math.MaxUint64 {
			return 0, fmt.Errorf("%d is out of range for uint64", r)
		}
	default:
		r, err = 0, fmt.Errorf("incorrect kind %v", kind)
	}

	return
}

// The strToFloatKind converts a string to float64 with out-of-range checking
// for float. Returns 0 if value is empty.
func strToFloatKind(value string, kind reflect.Kind) (r float64, err error) {
	// For empty string returns zero.
	if len(value) == 0 {
		return 0.0, nil
	}

	// Convert string to Float64.
	r, err = strconv.ParseFloat(value, 64)
	if err != nil {
		return 0.0, err
	}

	switch kind {
	case reflect.Float32:
		if r < -math.MaxFloat32 || r > math.MaxFloat32 {
			return 0.0, fmt.Errorf("%f is out of range for float32", r)
		}
	case reflect.Float64:
		if r < -math.MaxFloat64 || r > math.MaxFloat64 {
			return 0.0, fmt.Errorf("%f is out of range for float64", r)
		}
	default:
		r, err = 0, fmt.Errorf("incorrect kind")
	}

	return
}

// The strToBool convert string to bool type.
// Returns false if value is empty.
func strToBool(v string) (bool, error) {
	// For empty string returns false.
	if len(v) == 0 {
		return false, nil
	}

	// Convert string to bool.
	// It accepts 1, t, T, TRUE, true, True, 0, f, F, FALSE, false, False.
	// Any other value returns an error.
	r, err := strconv.ParseBool(v)
	if err != nil {
		// For example, the range -0.7 to 0.7 is false,
		// and the rest of the values are true.
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return r, fmt.Errorf("'%s' cannot be converted to a boolean", v)
		}

		r = math.Abs(f) > 0.7 // definition of state
	}

	return r, nil
}
