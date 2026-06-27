package env

import (
	"encoding"
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// textUnmarshalerType is the reflect.Type of encoding.TextUnmarshaler.
var textUnmarshalerType = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()

// The implementsTextUnmarshaler reports whether t (or a pointer to it)
// implements encoding.TextUnmarshaler.
func implementsTextUnmarshaler(t reflect.Type) bool {
	return t.Implements(textUnmarshalerType) ||
		reflect.PointerTo(t).Implements(textUnmarshalerType)
}

// The hasKeyPrefix reports whether the source has at least one key starting
// with the given prefix. It decides whether an optional nested-struct pointer
// should be allocated.
func hasKeyPrefix(source map[string]string, prefix string) bool {
	for key := range source {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}

	return false
}

// The isNestedStruct reports whether t (a field type) is a struct decoded
// recursively, as opposed to a leaf handled by setValue. Leaves are: scalars,
// slices/arrays, url.URL, time.Time, anything implementing TextUnmarshaler, and
// any type with a registered parser (WithParser). This single classifier is
// used by both the absent-key guard and the pointer case so the two cannot
// drift apart.
func isNestedStruct(t reflect.Type, s settings) bool {
	// A registered parser makes the type a leaf (check the type itself and,
	// for a pointer, its element).
	if s.parsers[t] != nil {
		return false
	}
	if t.Kind() == reflect.Ptr {
		if s.parsers[t.Elem()] != nil {
			return false
		}
		t = t.Elem()
	}

	// A type that implements TextUnmarshaler is a leaf, not a nested struct.
	if implementsTextUnmarshaler(t) {
		return false
	}

	return t.Kind() == reflect.Struct &&
		t != urlType &&
		t != timeTimeType
}

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

	// Walk the cached field descriptors and save data from the source.
	// Unexported and env:"-" fields are already excluded by cachedFields.
	e := v.Elem()
	for _, fi := range cachedFields(t.Elem()) {
		// Per-field tag overrides the call-level default.
		sep := fi.sepTag
		if sep == "" {
			sep = s.separator
		}
		layout := fi.layoutTag
		if layout == "" {
			layout = s.timeLayout
		}

		tg := &tagGroup{
			key:      s.prefix + fi.name,
			value:    fi.def,
			sep:      sep,
			layout:   resolveLayout(layout),
			required: fi.required,
		}

		if !tg.isValid() {
			return fmt.Errorf(
				"the %s field does not have a valid key name value: %s",
				fi.goName,
				tg.key,
			)
		}

		// If the key exists - take its value from the source.
		value, ok := source[tg.key]
		if ok {
			tg.value = value
		}
		tg.present = ok

		// A required field must be present in the source unless a default
		// is provided.
		if tg.required && !ok && fi.def == "" {
			return fmt.Errorf("%w: %s", ErrRequired, tg.key)
		}

		// Set value to field.
		item := e.Field(fi.index)
		if err := setFieldValue(source, &item, tg, s); err != nil {
			return err
		}
	}

	return nil
}

// The setFieldValue sets value to field from the tag arguments. The source
// and s are threaded through for the recursive decoding of nested structs.
func setFieldValue(source map[string]string, item *reflect.Value, tg *tagGroup, s settings) error {
	child := settings{
		prefix:     tg.key + "_",
		separator:  s.separator,
		timeLayout: s.timeLayout,
		parsers:    s.parsers,
		encoders:   s.encoders,
	}

	// Absent key with no default: leave the field untouched, like
	// encoding/json (a present but empty value still clears the field).
	// Nested structs are excluded - they are populated by their sub-keys
	// regardless of their own key.
	if !tg.present && tg.value == "" && !isNestedStruct(item.Type(), s) {
		return nil
	}

	// keyErr wraps a leaf conversion error with the key for context. Errors
	// from recursive decodeStruct calls are already keyed, so they pass
	// through unwrapped.
	keyErr := func(err error) error {
		if err == nil {
			return nil
		}
		return fmt.Errorf("%s: %w", tg.key, err)
	}

	// A custom parser (WithParser) takes precedence over the built-in handling
	// for its type. Pointers flow through the Ptr case below, which dereferences
	// to setValue (which also consults the parser).
	if item.Kind() != reflect.Ptr && s.parsers[item.Type()] != nil {
		if !tg.present && tg.value == "" {
			return nil // absent, no default: leave the field untouched
		}
		rv, err := s.parsers[item.Type()](tg.value)
		if err != nil {
			return keyErr(err)
		}
		item.Set(rv)
		return nil
	}

	// A non-pointer type that implements TextUnmarshaler is a leaf regardless
	// of its kind (e.g. net.IP is a slice), so handle it before the kind
	// switch. Pointers flow through the Ptr case, which dereferences to
	// setValue. time.Time/url.URL keep their special handling inside setValue.
	if item.Kind() != reflect.Ptr && implementsTextUnmarshaler(item.Type()) {
		return keyErr(setValue(*item, tg.value, tg.layout, s))
	}

	switch item.Kind() {
	case reflect.Array:
		max := item.Type().Len()
		seq := splitN(tg.value, tg.sep, -1)
		if len(seq) > max {
			return fmt.Errorf("%s: %d overflows the [%d]array", tg.key, len(seq), max)
		}

		// Replace: clear the array, then fill the parsed elements.
		item.Set(reflect.Zero(item.Type()))
		if err := setSequence(item, seq, tg.layout, s); err != nil {
			return keyErr(err)
		}
	case reflect.Slice:
		seq := splitN(tg.value, tg.sep, -1)
		tmp := reflect.MakeSlice(item.Type(), len(seq), len(seq))
		if err := setSequence(&tmp, seq, tg.layout, s); err != nil {
			return keyErr(err)
		}

		// Replace the slice (like encoding/json), not append to it.
		item.Set(tmp)
	case reflect.Ptr:
		// A nil pointer is "absent". The absent case is already handled by the
		// guard above (leaf) or below (nested struct), so here we only allocate
		// when there is something to assign.
		elemType := item.Type().Elem()
		isLeaf := !isNestedStruct(elemType, s)

		if isLeaf {
			if item.IsNil() {
				item.Set(reflect.New(elemType))
			}
			if err := setValue(item.Elem(), tg.value, tg.layout, s); err != nil {
				return keyErr(err)
			}
			break
		}

		// Pointer to a nested struct: allocate only when the source has at
		// least one key under this prefix, then decode recursively.
		if item.IsNil() {
			if !hasKeyPrefix(source, tg.key+"_") {
				break
			}
			item.Set(reflect.New(elemType))
		}
		if err := decodeStruct(source, item.Interface(), child); err != nil {
			return err
		}
	case reflect.Struct:
		if item.Type() == urlType ||
			item.Type() == timeTimeType {
			// A leaf struct type handled by setValue.
			if err := setValue(*item, tg.value, tg.layout, s); err != nil {
				return keyErr(err)
			}
			break
		}

		// A nested struct: decode in place so fields absent from the source
		// keep their existing (default) values.
		if err := decodeStruct(source, item.Addr().Interface(), child); err != nil {
			return err
		}
	default:
		// Try to set correct value.
		if err := setValue(*item, tg.value, tg.layout, s); err != nil {
			return keyErr(err)
		}
	}

	return nil
}

// The setSequence sets slice into item, if item is slice or array.
func setSequence(item *reflect.Value, seq []string, layout string, s settings) error {
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
		if err := setValue(elem, unquoteElement(value), layout, s); err != nil {
			return err
		}
	}

	return nil
}

// The unquoteElement reverses quoteElement: a double-quoted element is stripped
// of its quotes and the \ and " escapes are removed. A bare element is returned
// unchanged.
func unquoteElement(v string) string {
	if len(v) < 2 || v[0] != '"' || v[len(v)-1] != '"' {
		return v
	}

	inner := v[1 : len(v)-1]
	if !strings.Contains(inner, "\\") {
		return inner
	}

	var b strings.Builder
	b.Grow(len(inner))
	for i := 0; i < len(inner); i++ {
		if inner[i] == '\\' && i+1 < len(inner) {
			i++
		}
		b.WriteByte(inner[i])
	}

	return b.String()
}

// The setValue sets value into item (field of the struct).
func setValue(item reflect.Value, value, layout string, s settings) error {
	// A custom parser (WithParser) wins over the built-in handling for its
	// type. This also covers slice/array elements and dereferenced pointers.
	if p := s.parsers[item.Type()]; p != nil {
		rv, err := p(value)
		if err != nil {
			return err
		}
		item.Set(rv)
		return nil
	}

	// time.Duration (an int64) and time.Time (a struct) are parsed by type,
	// before the generic kind handling. An empty value keeps the zero value.
	switch item.Type() {
	case timeDurationType:
		if value == "" {
			return nil
		}
		d, err := time.ParseDuration(value)
		if err != nil {
			return err
		}
		item.SetInt(int64(d))
		return nil
	case timeTimeType:
		if value == "" {
			return nil
		}
		tm, err := time.Parse(layout, value)
		if err != nil {
			return err
		}
		item.Set(reflect.ValueOf(tm))
		return nil
	case timeTimePtrType:
		if value == "" {
			return nil
		}
		tm, err := time.Parse(layout, value)
		if err != nil {
			return err
		}
		item.Set(reflect.ValueOf(&tm))
		return nil
	}

	kind := item.Kind()

	// The *url.URL pointer only.
	if kind == reflect.Ptr && item.Type() == urlPtrType {
		u, err := url.Parse(value)
		if err != nil {
			return err
		}
		item.Set(reflect.ValueOf(u))
		return nil
	}

	// Any other pointer (e.g. *int, *string): an empty value leaves it nil,
	// otherwise allocate and set the pointed-to value. This makes pointer
	// elements of slices/arrays ([]*int, [N]*string, ...) work.
	if kind == reflect.Ptr {
		if value == "" {
			return nil
		}
		if item.IsNil() {
			item.Set(reflect.New(item.Type().Elem()))
		}
		return setValue(item.Elem(), value, layout, s)
	}

	// The url.URL struct only.
	if kind == reflect.Struct && item.Type() == urlType {
		u, err := url.Parse(value)
		if err != nil {
			return err
		}
		item.Set(reflect.ValueOf(*u))
		return nil
	}

	// Any type implementing TextUnmarshaler (net.IP, netip.Addr, custom enums,
	// ...) is parsed via UnmarshalText. An empty value leaves the zero value.
	// This is checked after the special-cased time/url types above.
	if value != "" && item.CanAddr() {
		if u, ok := item.Addr().Interface().(encoding.TextUnmarshaler); ok {
			return u.UnmarshalText([]byte(value))
		}
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
	// For empty string returns zero.
	if len(value) == 0 {
		return 0, nil
	}

	// strconv.ParseInt performs the out-of-range check for the bit size.
	bitSize := 64
	switch kind {
	case reflect.Int:
		bitSize = strconv.IntSize
	case reflect.Int8:
		bitSize = 8
	case reflect.Int16:
		bitSize = 16
	case reflect.Int32:
		bitSize = 32
	case reflect.Int64:
		bitSize = 64
	default:
		return 0, fmt.Errorf("incorrect kind %v", kind)
	}

	return strconv.ParseInt(value, 10, bitSize)
}

// The strToUintKind converts string to uint64 type with out-of-range checking
// for uint. Returns 0 if value is empty.
func strToUintKind(value string, kind reflect.Kind) (uint64, error) {
	// For empty string returns zero.
	if len(value) == 0 {
		return 0, nil
	}

	// strconv.ParseUint performs the out-of-range check for the bit size.
	bitSize := 64
	switch kind {
	case reflect.Uint:
		bitSize = strconv.IntSize
	case reflect.Uint8:
		bitSize = 8
	case reflect.Uint16:
		bitSize = 16
	case reflect.Uint32:
		bitSize = 32
	case reflect.Uint64:
		bitSize = 64
	default:
		return 0, fmt.Errorf("incorrect kind %v", kind)
	}

	return strconv.ParseUint(value, 10, bitSize)
}

// The strToFloatKind converts a string to float64 with out-of-range
// checking for float. Returns 0 if value is empty.
func strToFloatKind(value string, kind reflect.Kind) (float64, error) {
	// For empty string returns zero.
	if len(value) == 0 {
		return 0.0, nil
	}

	// strconv.ParseFloat performs the out-of-range check for the bit size.
	bitSize := 64
	switch kind {
	case reflect.Float32:
		bitSize = 32
	case reflect.Float64:
		bitSize = 64
	default:
		return 0.0, fmt.Errorf("incorrect kind %v", kind)
	}

	return strconv.ParseFloat(value, bitSize)
}

// The strToBool convert string to bool type.
// Returns false if value is empty.
func strToBool(v string) (bool, error) {
	// For empty string returns false.
	if len(v) == 0 {
		return false, nil
	}

	// Try the standard literals first:
	// 1, t, T, TRUE, true, True, 0, f, F, FALSE, false, False.
	if r, err := strconv.ParseBool(v); err == nil {
		return r, nil
	}

	// Accept the common config synonyms (case-insensitive).
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "yes", "on":
		return true, nil
	case "no", "off":
		return false, nil
	}

	return false, fmt.Errorf("%q cannot be converted to a boolean", v)
}
