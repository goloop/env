package env

import (
	"errors"
	"fmt"
	"math"
	"net/url"
	"reflect"
	"strconv"
)

// Unmarshaler is the interface implements by types that can
// unmarshal an environment variables of themselves.
type Unmarshaler interface {
	UnmarshalENV() error
}

// The unmarshalENV read variables from the environment and save them into obj.
// The obj should passed as a pointer on the object (Object Pointer).
//
// Method supports the following type of the fields: int, int8, int16, int32,
// int64, uin, uint8, uin16, uint32, in64, float32, float64, string, bool,
// struct, url.URL and pointers, array or slice from thous types (i.e. *int,
// *uint, ..., []int, ..., []bool, ..., [2]*url.URL, etc.). The fields as
// a struct or pointer on the struct will be processed recursively.
//
// For other type of the filed (i.e chan, map ...) or upon occurrence other
// conversion problems will be returned an error.
//
// The pfx argument serves as a marker of the nesting level during the
// recursive processing of object fields (as prefix for environment variables).
func unmarshalENV(obj interface{}, pfx string) (err error) {
	var t, e = reflect.TypeOf(obj), reflect.ValueOf(obj)

	// The obj argument should be a pointer to initialized object.
	if obj == nil ||
		t.Kind() != reflect.Ptr || // check for pointer first ...
		t.Elem().Kind() != reflect.Struct || // ... and after on the struct
		!e.Elem().IsValid() {
		return errors.New("cannot unmarshal environment into not *struct type")
	}

	// If objects implements Unmarshaler interface try to calling
	// a custom Unmarshal method.
	if e.Type().Implements(reflect.TypeOf((*Unmarshaler)(nil)).Elem()) {
		if m := e.MethodByName("UnmarshalENV"); m.IsValid() {
			tmp := m.Call([]reflect.Value{})
			err := tmp[0].Interface()
			if err != nil {
				return fmt.Errorf("%v", err)
			}
			return nil
		}
	}

	// Walk through all the fields of the structure and
	// load data from the environment.
	elem := e.Elem()
	for i := 0; i < elem.NumField(); i++ {
		// Parse tag arguments of the field.
		field := t.Elem().Field(i)
		args := getTagArgs(field.Tag.Get("env"), field.Name)
		if !args.IsValid() {
			return fmt.Errorf("invalid key name: %s", args.Key)
		} else if args.IsIgnored() {
			continue
		}

		// If recursive analysis is performed set nesting level.
		args.Key = fmt.Sprintf("%s%s", pfx, args.Key)

		// If the value is defined in environment set it into value.
		if Exists(args.Key) {
			args.Value = Get(args.Key)
		}

		// Set value to field.
		item := elem.FieldByName(field.Name)
		err := setFieldValue(&item, args)
		if err != nil {
			return fmt.Errorf("%v", err)
		}
	}

	return nil
}

// The setFieldValue sets value to field from the tag araguments.
func setFieldValue(item *reflect.Value, args *tagArgs) error {
	switch item.Kind() {
	case reflect.Array:
		max := item.Type().Len()
		seq := splitN(args.Value, args.Sep, -1)
		if len(seq) > max {
			return fmt.Errorf("%d overflows the [%d]array", len(seq), max)
		}

		err := setSequence(item, seq)
		if err != nil {
			return err
		}
	case reflect.Slice:
		seq := splitN(args.Value, args.Sep, -1)
		tmp := reflect.MakeSlice(item.Type(), len(seq), len(seq))
		err := setSequence(&tmp, seq)
		if err != nil {
			return err
		}
		item.Set(reflect.AppendSlice(*item, tmp))
	case reflect.Ptr:
		if item.Type().Elem().Kind() != reflect.Struct {
			// If the pointer is not to a structure.
			tmp := reflect.Indirect(*item)
			err := setValue(tmp, args.Value)
			if err != nil {
				return err
			}
			break
		} else if item.Type() == reflect.TypeOf((*url.URL)(nil)) {
			// If a pointer to a structure of the url.URL.
			err := setValue(*item, args.Value)
			if err != nil {
				return err
			}
			break
		}

		// If a pointer to a structure of the another's types.
		// P.s. Not a *url.URL.
		tmp := reflect.New(item.Type().Elem()).Interface()
		err := unmarshalENV(tmp, fmt.Sprintf("%s_", args.Key))
		if err != nil {
			return err
		}
		item.Set(reflect.ValueOf(tmp))
	case reflect.Struct:
		if item.Type() == reflect.TypeOf(url.URL{}) {
			// If a url.URL structure.
			err := setValue(*item, args.Value)
			if err != nil {
				return err
			}
			break
		}

		// If a structure of the another's types.
		// P.s. Not a url.URL.
		tmp := reflect.New(item.Type()).Interface()
		err := unmarshalENV(tmp, fmt.Sprintf("%s_", args.Key))
		if err != nil {
			return err
		}
		item.Set(reflect.ValueOf(tmp).Elem())
	default:
		// Try to set correct value.
		err := setValue(*item, args.Value)
		if err != nil {
			return err
		}
	}
	return nil
}

// The setSequence sets slice into item, if item is slice or array.
func setSequence(item *reflect.Value, seq []string) (err error) {
	if len(seq) == 0 ||
		(item.Index(0).Kind() == reflect.Array && item.Type().Len() == 0) ||
		(item.Index(0).Kind() == reflect.Slice && item.Len() == 0) {
		return nil
	}

	for i, value := range seq {
		elem := item.Index(i)
		err := setValue(elem, value)
		if err != nil {
			return err
		}
	}

	return nil
}

// The setValue sets value into item.
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

// The strToIntKind converts string to int64 type with checking for conversion
// to intX type. Returns default value for int type if value is empty.
//
// P.s. The intX determined by reflect.Kind.
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
		// For 32-bit platform it is necessary to check overflow.
		if strconv.IntSize == 32 {
			if r < math.MinInt32 || r > math.MaxInt32 {
				return 0, fmt.Errorf("%d overflows int (int32)", r)
			}
		}
	case reflect.Int8:
		if r < math.MinInt8 || r > math.MaxInt8 {
			return 0, fmt.Errorf("%d overflows int8", r)
		}
	case reflect.Int16:
		if r < math.MinInt16 || r > math.MaxInt16 {
			return 0, fmt.Errorf("%d overflows int16", r)
		}
	case reflect.Int32:
		if r < math.MinInt32 || r > math.MaxInt32 {
			return 0, fmt.Errorf("%d overflows int32", r)
		}
	case reflect.Int64:
		// pass
	default:
		r, err = 0, fmt.Errorf("incorrect kind %v", kind)
	}

	return
}

// strToUintKind convert string to uint64 type with checking for conversion
// to uintX type. Returns default value for uint type if value is empty.
//
// P.s. The uintX determined by reflect.Kind.
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
		if strconv.IntSize == 32 && r > math.MaxUint32 {
			return 0, fmt.Errorf("%d overflows uint (uint32)", r)
		}
	case reflect.Uint8:
		if r > math.MaxUint8 {
			return 0, fmt.Errorf("%d overflows uint8", r)
		}
	case reflect.Uint16:
		if r > math.MaxUint16 {
			return 0, fmt.Errorf("%d overflows uint16", r)
		}
	case reflect.Uint32:
		if r > math.MaxUint32 {
			return 0, fmt.Errorf("strToUint32: %d overflows uint32", r)
		}
	case reflect.Uint64:
		// pass
	default:
		r, err = 0, fmt.Errorf("incorrect kind %v", kind)
	}

	return
}

// strToFloatKind convert string to float64 type with checking for conversion
// to floatX type. Returns default value for float64 type if value is empty.
//
// P.s. The floatX determined by reflect.Kind.
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
		if math.Abs(r) > math.MaxFloat32 {
			return 0.0, fmt.Errorf("%f overflows float32", r)
		}
	case reflect.Float64:
		// pass
	default:
		r, err = 0, fmt.Errorf("incorrect kind")
	}

	return
}

// strToBool convert string to bool type. Returns: result, error.
// Returns default value for bool type if value is empty.
func strToBool(value string) (bool, error) {
	var epsilon = math.Nextafter(1, 2) - 1

	// For empty string returns false.
	if len(value) == 0 {
		return false, nil
	}

	r, errB := strconv.ParseBool(value)
	if errB != nil {
		f, errF := strconv.ParseFloat(value, 64)
		if errF != nil {
			return r, errB
		}

		if math.Abs(f) > epsilon {
			r = true
		}
	}

	return r, nil
}
