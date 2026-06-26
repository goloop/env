package env

import (
	"encoding"
	"fmt"
	"net/url"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"
)

// textMarshalerType is the reflect.Type of encoding.TextMarshaler.
var textMarshalerType = reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem()

// The implementsTextMarshaler reports whether t (or a pointer to it)
// implements encoding.TextMarshaler.
func implementsTextMarshaler(t reflect.Type) bool {
	return t.Implements(textMarshalerType) ||
		reflect.PointerTo(t).Implements(textMarshalerType)
}

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

// The encodeStruct converts the fields of obj into an ordered list of key/value
// pairs, honouring the prefix and default separator from s. It has no side
// effects: callers decide whether to write the pairs to the environment, a map
// or a file.
//
// If obj implements Marshaler, its MarshalEnv result is returned with keys
// sorted for a deterministic order.
func encodeStruct(obj any, s settings) ([]pair, error) {
	if obj == nil {
		return nil, ErrNilObject
	}

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

			// Apply the prefix to the custom keys too, so WithPrefix behaves
			// the same as for reflective structs.
			pairs := mapToPairs(tmp[0].Interface().(map[string]string))
			if s.prefix != "" {
				for i := range pairs {
					pairs[i].key = s.prefix + pairs[i].key
				}
			}
			return pairs, nil
		}
	}

	// Walk the cached field descriptors over an addressable copy, so
	// pointer-receiver methods (e.g. a *T-receiver TextMarshaler) can be called.
	// Unexported and env:"-" fields are already excluded by cachedFields.
	ev := ptr.Elem()
	result := make([]pair, 0, rt.NumField())
	for _, fi := range cachedFields(rt) {
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
			key:    fi.name,
			value:  fi.def,
			sep:    sep,
			layout: resolveLayout(layout),
		}

		if !tg.isValid() {
			return nil, fmt.Errorf(
				"the %s field does not have a valid key name value: %s",
				fi.goName,
				tg.key,
			)
		}

		// Get item. A nil pointer field means "absent value": skip it so the
		// key is omitted (it round-trips back to nil on decode).
		item := ev.Field(fi.index)
		if item.Kind() == reflect.Ptr {
			if item.IsNil() {
				continue
			}
			item = item.Elem()
		}

		// A custom encoder (WithEncoder) or a TextMarshaler is a leaf regardless
		// of its kind (e.g. net.IP is a slice), so format it via toStr before
		// the kind switch. time.Time still uses its layout (handled in toStr).
		if s.encoders[item.Type()] != nil || implementsTextMarshaler(item.Type()) {
			value, err := toStr(item, tg.layout, s)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", s.prefix+tg.key, err)
			}
			result = append(result, pair{key: s.prefix + tg.key, value: value})
			continue
		}

		switch item.Kind() {
		case reflect.Array, reflect.Slice:
			value, err := getSequence(&item, tg.sep, tg.layout, s)
			if err != nil {
				return nil, err
			}
			tg.value = value
		case reflect.Struct:
			// Support for url.URL and time.Time structs.
			if u, ok := item.Interface().(url.URL); ok {
				tg.value = u.String()
				break // break switch
			}
			if tm, ok := item.Interface().(time.Time); ok {
				tg.value = tm.Format(tg.layout)
				break // break switch
			}

			// Another struct.
			// Recursive analysis of the nested structure.
			child := settings{
				prefix:     s.prefix + tg.key + "_",
				separator:  s.separator,
				timeLayout: s.timeLayout,
				parsers:    s.parsers,
				encoders:   s.encoders,
			}
			nested, err := encodeStruct(item.Interface(), child)
			if err != nil {
				return nil, err
			}

			result = append(result, nested...)
			continue // nested struct contributes its own pairs
		default:
			value, err := toStr(item, tg.layout, s)
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
	slices.Sort(keys)

	pairs := make([]pair, 0, len(m))
	for _, k := range keys {
		pairs = append(pairs, pair{key: k, value: m[k]})
	}

	return pairs
}

// The getSequence joins the elements of a slice or array into a string. A nil
// pointer element is written as an empty value at its position (so it
// round-trips back to a nil element on decode).
func getSequence(item *reflect.Value, sep, layout string, s settings) (string, error) {
	var max int
	switch item.Kind() {
	case reflect.Array:
		max = item.Type().Len()
	case reflect.Slice:
		max = item.Len()
	default:
		return "", fmt.Errorf("incorrect type: %s", item.Type())
	}

	var sb strings.Builder
	for i := 0; i < max; i++ {
		if i > 0 {
			sb.WriteString(sep)
		}

		elem := item.Index(i)
		if elem.Kind() == reflect.Ptr {
			if elem.IsNil() {
				continue // nil element -> empty value at this position
			}
			elem = elem.Elem()
		}

		v, err := toStr(elem, layout, s)
		if err != nil {
			return "", err
		}
		sb.WriteString(quoteElement(v, sep))
	}

	return sb.String(), nil
}

// The quoteElement wraps a sequence element in double quotes (escaping \ and ")
// when it would otherwise be mis-split on decode: it contains the separator or
// starts with a quote character. splitN groups quoted spans, so the quoted
// element survives the split and is unquoted symmetrically by unquoteElement.
func quoteElement(v, sep string) string {
	if v == "" {
		return v
	}
	// Quote when the element contains the separator or any character splitN
	// groups on (quotes/brackets), which would otherwise mis-split it.
	if !strings.Contains(v, sep) && !strings.ContainsAny(v, "\"'`()[]{}") {
		return v
	}

	var b strings.Builder
	b.Grow(len(v) + 2)
	b.WriteByte('"')
	for i := 0; i < len(v); i++ {
		if v[i] == '\\' || v[i] == '"' {
			b.WriteByte('\\')
		}
		b.WriteByte(v[i])
	}
	b.WriteByte('"')

	return b.String()
}

// The toStr converts any item to string.
func toStr(item reflect.Value, layout string, s settings) (string, error) {
	// A custom encoder (WithEncoder) wins over the built-in handling for its
	// type. This also covers slice/array elements.
	if e := s.encoders[item.Type()]; e != nil {
		return e(item)
	}

	// time.Duration and time.Time are formatted by type, before the generic
	// kind handling (Duration's kind is int64, Time's kind is struct).
	switch item.Type() {
	case timeDurationType:
		return time.Duration(item.Int()).String(), nil
	case timeTimeType:
		return item.Interface().(time.Time).Format(layout), nil
	}

	// Any type implementing TextMarshaler (net.IP, netip.Addr, custom enums,
	// ...) is formatted via MarshalText. Try the addressable pointer first so a
	// pointer-receiver MarshalText is honoured, then the value. Checked after
	// the special-cased time types above so time.Time keeps its layout.
	var tm encoding.TextMarshaler
	if item.CanAddr() {
		tm, _ = item.Addr().Interface().(encoding.TextMarshaler)
	}
	if tm == nil {
		tm, _ = item.Interface().(encoding.TextMarshaler)
	}
	if tm != nil {
		b, err := tm.MarshalText()
		if err != nil {
			return "", err
		}
		return string(b), nil
	}

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

// The quoteEnvValue formats value so it round-trips through the parser when
// written as a KEY=value line. A value that would be misread when written bare
// (a newline, leading/trailing whitespace, an inline-comment "#", or a leading
// quote) is wrapped in double quotes with the escapes the parser understands
// (\n, \t, \r, \\, \"). Plain values are returned unchanged.
func quoteEnvValue(value string) string {
	if !needsQuoting(value) {
		return value
	}

	// A value containing '$' must be written non-expandably: the reader expands
	// ${VAR}/$VAR in unquoted and double-quoted values, but not in single-quoted
	// or backtick-quoted ones. Pick a quote the value does not itself contain.
	if strings.ContainsRune(value, '$') {
		if !strings.ContainsRune(value, '\'') {
			return "'" + value + "'"
		}
		if !strings.ContainsRune(value, '`') {
			return "`" + value + "`"
		}
		// '$' together with both ' and `: fall through to double quotes (the
		// '$' will be expanded on read — a rare, documented edge).
	}

	var b strings.Builder
	b.Grow(len(value) + 2)
	b.WriteByte('"')
	for _, r := range value {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')

	return b.String()
}

// The needsQuoting reports whether value must be quoted to survive a round-trip
// as an unquoted .env value.
func needsQuoting(value string) bool {
	if value == "" {
		return false // KEY= is a valid empty value
	}

	// Unquoted values are trimmed, so edge whitespace would be lost.
	if value[0] == ' ' || value[0] == '\t' ||
		value[len(value)-1] == ' ' || value[len(value)-1] == '\t' {
		return true
	}

	// A leading quote/backtick would be parsed as a quoted value.
	if value[0] == '"' || value[0] == '\'' || value[0] == '`' {
		return true
	}

	for i := 0; i < len(value); i++ {
		switch value[i] {
		case '\n', '\r':
			return true
		case '$':
			return true // would be expanded as ${VAR}/$VAR on read
		case '#':
			// A "#" preceded by whitespace would start an inline comment.
			if i > 0 && (value[i-1] == ' ' || value[i-1] == '\t') {
				return true
			}
		}
	}

	return false
}
