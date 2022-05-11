package env

import (
	"fmt"
	"reflect"
	"strings"
)

// These functions are not used as a subsystem of the env package functions,
// they are used for testing package functions only.

// The sts converts a slice of any type to a string. If pass a value
// other than a slice or array, an empty string will be returned.
//
// Examples:
//   sts([]int{1,2,3}, ",") // "1,2,3"
//   sts([]string{"1", "2", "3"}, ",") // "1,2,3"
func sts(slice interface{}, sep string) string {
	var result string

	switch reflect.TypeOf(slice).Kind() {
	case reflect.Array:
		fallthrough
	case reflect.Slice:
		s := reflect.ValueOf(slice)
		for i := 0; i < s.Len(); i++ {
			if i == 0 {
				result = fmt.Sprint(s.Index(i))
			} else {
				result = fmt.Sprintf("%s%s%v", result, sep, s.Index(i))
			}
		}
	}

	return result
}

// The fts returns data as string from the struct by field name.
// If name gets the name of the key-like (with `_` separator),
// for example KEY_A it will be converted to go-like name - KeyA.
//
// If the specified field is missing from the structure,
// an empty string will be returned.
func fts(v interface{}, name, sep string) string {
	// Correct the field name to go-style.
	if strings.Contains(name, "_") {
		var sb strings.Builder
		for _, chunk := range strings.Split(name, "_") {
			sb.WriteString(strings.Title(strings.ToLower(chunk)))
		}
		name = sb.String()
	}

	// Get the value of the field.
	r := reflect.ValueOf(v)
	f := reflect.Indirect(r).FieldByName(name)

	switch {
	case !f.IsValid():
		return ""
	case f.Kind() == reflect.Slice || f.Kind() == reflect.Array:
		return strings.Trim(strings.Replace(fmt.Sprint(f), " ", sep, -1), "[]")
	}

	return fmt.Sprint(f)
}
