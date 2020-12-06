package env

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	emptyRgx = regexp.MustCompile(`^(\s*)$|^(\s*[#].*)$`)
	valueRgx = regexp.MustCompile(`^=[^\s].*`)
	keyRgx   = regexp.MustCompile(
		`^(?:\s*)?(?:export\s+)?(?P<key>[a-zA-Z_][a-zA-Z_0-9]*)=`,
	)
)

// The sts converts slice to string.
// The function isn't intended for production and is used in testing
// when superficially comparing slices of different types.
//
// Examples:
//    sts([]int{1,2,3}, ",") == sts([]string{"1", "2", "3"}, ",") // true
func sts(slice interface{}, sep string) (r string) {
	switch reflect.TypeOf(slice).Kind() {
	case reflect.Array:
		fallthrough
	case reflect.Slice:
		s := reflect.ValueOf(slice)
		for i := 0; i < s.Len(); i++ {
			r += fmt.Sprint(s.Index(i)) + sep
		}
	}

	return strings.TrimSuffix(r, sep)
}

// The fts returns data as string from the struct by field name.
func fts(v interface{}, name, sep string) string {
	if strings.Contains(name, "_") {
		var tmp string
		for _, item := range strings.Split(name, "_") {
			tmp += strings.Title(strings.ToLower(item))
		}
		name = tmp
	}

	r := reflect.ValueOf(v)
	f := reflect.Indirect(r).FieldByName(name)

	if f.Kind() == reflect.Slice || f.Kind() == reflect.Array {
		return strings.Trim(strings.Replace(fmt.Sprint(f), " ", sep, -1), "[]")
	}

	return fmt.Sprint(f)
}

// The split splits the string at the specified rune-marker ignoring the
// position of the marker inside of the group: `...`, '...', "..."
// and (...), {...}, [...].
//
// Examples:
//    split("a,b,c,d", ',')     // ["a", "b", "c", "d"]
//    split("a,(b,c),d", ',')   // ["a", "(b,c)", "d"]
//    split("'a,b',c,d", ',')   // ["'a,b'", "c", "d"]
//    split("a,\"b,c\",d", ',') // ["a", "\"b,c\"", "d"]
func split(str string, sep string) (r []string) {
	var (
		level int
		host  rune
		char  rune
		tmp   string

		flips    = map[rune]rune{'}': '{', ']': '[', ')': '('}
		quotes   = "\"'`"
		brackets = "({["
	)

	// The contains returns true if all items from the separators
	// were found in the string and it's all the same.
	contains := func(str string, separators ...rune) bool {
		var last = -1
		for _, sep := range separators {
			ir := strings.IndexRune(str, sep)
			if ir < 0 || (last >= 0 && last != ir) {
				return false
			}
			last = ir
		}

		return true
	}

	// Allocate the max memory size for storage all fields.
	r = make([]string, 0, strings.Count(str, ",")+1)

	// Split value.
	// for _, char = range str {
	for i := 0; i < utf8.RuneCountInString(str); i++ {
		char = rune(str[i])
		switch {
		case level == 0 && contains(quotes+brackets, char):
			host, level = char, level+1
		case contains(quotes, host, char):
			level, host = 0, 0
		case contains(brackets, host, flips[char]):
			level--
			if level <= 0 {
				level, host = 0, 0
			}
		case sep == str[i:i+utf8.RuneCountInString(sep)] && level == 0:
			i += utf8.RuneCountInString(sep) - 1
			r = append(r, tmp)
			tmp = ""
			continue
		}

		tmp += string(char)
	}

	// Add last piece to the result.
	if len(tmp) != 0 || string(char) == sep {
		r = append(r, tmp)
	}

	return
}

// isEmpty returns true if string contains separators or comment only.
func isEmpty(str string) bool {
	return emptyRgx.Match([]byte(str))
}

// removeInlineComment removes the comment in the string.
// Only in strings where the value is enclosed in quotes.
func removeInlineComment(str, quote string) string {
	// If the comment is in the string.
	if strings.Contains(str, "#") {
		chunks := strings.Split(str, "#")
		for i := range chunks {
			str := strings.Join(chunks[:i], "#")
			if len(str) > 0 && strings.Count(str, quote)%2 == 0 {
				return strings.TrimSpace(str)
			}
		}
	}
	return str
}

// parseExpression breaks expression into key and value, ignore
// comments and any spaces.
//
// Note: value must be an expression.
func parseExpression(exp string) (key, value string, err error) {
	var (
		quote  string = "\""
		marker string = fmt.Sprintf("<::%d::>", time.Now().Unix())
	)

	// Get key.
	// Remove `export` prefix, `=` suffix and trim spaces.
	tmp := keyRgx.FindStringSubmatch(exp)
	if len(tmp) < 2 {
		err = fmt.Errorf("missing variable name")
		return
	}
	key = tmp[1]

	// Get value.
	// ... the `=` sign in the string.
	value = exp[strings.Index(exp, "="):]
	if !valueRgx.Match([]byte(value)) {
		err = fmt.Errorf("incorrect value: %s", value)
		return
	}
	value = strings.TrimSpace(value[1:])

	switch {
	case strings.HasPrefix(value, "'"):
		quote = "'"
		fallthrough
	case strings.HasPrefix(value, "\""):
		// Replace escaped quotes, remove comment in the string,
		// check begin- and end- quotes and back escaped quotes.
		value = strings.Replace(value, fmt.Sprintf("\\%s", quote), marker, -1)
		value = removeInlineComment(value, quote)
		if strings.Count(value, quote)%2 != 0 { // begin- and end- quotes
			err = fmt.Errorf("incorrect value: %s", value)
			return
		}
		value = value[1 : len(value)-1] // remove begin- and end- quotes
		// ... change `\"` and `\'` to `"` and `'`.
		value = strings.Replace(value, marker, quote, -1)
	default:
		if strings.Contains(value, "#") {
			// Split by sharp sign and for string without quotes -
			// the first element has the meaning only.
			chunks := strings.Split(value, "#")
			chunks = strings.Split(chunks[0], " ")
			value = strings.TrimSpace(chunks[0])
		}
	}

	return
}
