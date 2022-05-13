package env

import (
	"bufio"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"
	"unicode/utf8"
)

// The sts converts a slice of any type to a string. If pass a value
// other than a slice or array, an empty string will be returned.
//
// Note: This function is not used as an env function subsystem,
// it is only used to test package functions.
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
//
// Note: This function is not used as an env function subsystem,
// it is only used to test package functions.
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

// The isEmpty returns true if string contains separators or comment only.
func isEmpty(str string) bool {
	return emptyLineRgx.Match([]byte(str))
}

// The readParseStore reads env-file, parses this one by the key and value, and
// stores into environment. It's flexible function that can be used to build
// more specific tools.
//
// Arguments
//
//     filename  path to the env-file;
//     expand    if true replaces ${var} or $var on the values
//               from the current environment variables;
//     update    if true overwrites the value that has already been
//               set in the environment to new one from the env-file;
//     forced    if true ignores wrong entries in env-file and loads
//               all of possible options, without causing an exception.
//
// Examples
//
// There is `.env` env-file that contains:
//
//     # .env file
//     HOST=0.0.0.0
//     PORT=80
//     EMAIL=$USER@goloop.one
//
// Some variables are already exists in the environment:
//
//     $ env | grep -E "USER|HOST"
//     USER=goloop
//     HOST=localhost
//
// To correctly load data from env-file followed by updating the environment:
//
//     env.ReadParseStore(".env", true, true, false)
//
//     // USER=goloop
//     // HOST=0.0.0.0
//     // PORT=80
//     // EMAIL=goloop@goloop.one
//
// Loading new keys to the environment without updating existing ones:
//
//     env.ReadParseStore(".env", true, false, false)
//
//     // USER=goloop
//     // HOST=localhost          <= hasn't been updated
//     // PORT=80
//     // EMAIL=goloop@goloop.one
//
// Don't change values that contain keys:
//
//     env.ReadParseStore(".env", false, true, false)
//
//     // USER=goloop
//     // HOST=0.0.0.0
//     // PORT=80
//     // EMAIL=$USER@goloop.one  <= $USER hasn't been changed to real value
//
// Loading data from a damaged env-file. If the configuration env-file is used
// by other applications and can have incorrect key/value, it can be ignored.
// For example env-file contains incorrect key `1BC` (the variable name can't
// start with a digit):
//
//     # .env file
//     HOST=0.0.0.0
//     PORT=80
//     1BC=NO                     # <= incorrect variable
//     EMAIL=$USER@goloop.one
//
// There will be an error loading this file:
//
//     err := env.ReadParseStore(".env", true, true, false)
//     if err != nil {
//         log.Panic(err) // panic: missing variable name
//     }
//
// but we can use force method to ignore this line:
//
//     // ... set forced as true (last argument)
//     err := env.ReadParseStore(".env", true, true, true)
//
//     // now the err variable is nil and environment has:
//     // USER=goloop
//     // HOST=0.0.0.0
//     // PORT=80
//     // EMAIL=goloop@goloop.one
func readParseStore(filename string, expand, update, forced bool) (err error) {
	var (
		file       *os.File
		key, value string
	)

	// Open env-file.
	file, err = os.Open(filename)
	if err != nil {
		return // unable to open file
	}
	defer file.Close()

	// Parse file.
	// TODO: use goroutines.
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		// Get current line and ignore empty string or comments.
		str := scanner.Text()
		if isEmpty(str) {
			continue
		}

		// Parse expression.
		// The string containing the expression must be of the
		// format as: [export] KEY=VALUE [# Comment]
		key, value, err = parseExpression(str)
		if err != nil {
			if forced {
				continue // ignore wrong entry
			}
			return // incorrect expression
		}

		// Overwrite or add new value.
		if _, ok := os.LookupEnv(key); update || !ok {
			if expand {
				value = Expand(value)
			}
			err = Set(key, value)
			if err != nil {
				return
			}
		}
	}

	return scanner.Err()
}

// The splitN splits the string at the specified rune-marker ignoring the
// position of the marker inside of the group: `...`, '...', "..."
// and (...), {...}, [...].
//
// Arguments:
//    str  data;
//    sep  element separator;
//    n    the number of strings to be returned by the function.
//         It can be any of the following:
//         - n is equal to zero (n == 0): The result is nil, i.e, zero
//           sub strings. An empty list is returned;
//         - n is greater than zero (n > 0): At most n sub strings will be
//           returned and the last string will be the unsplit remainder;
//         - n is less than zero (n < 0): All possible substring
//           will be returned.
//
// Examples:
//    splitN("a,b,c,d", ',', -1)     // ["a", "b", "c", "d"]
//    splitN("a,(b,c),d", ',', -1)   // ["a", "(b,c)", "d"]
//    splitN("'a,b',c,d", ',', -1)   // ["'a,b'", "c", "d"]
//    splitN("a,\"b,c\",d", ',', -1) // ["a", "\"b,c\"", "d"]
func splitN(str, sep string, n int) (r []string) {
	var (
		level int
		host  rune
		char  rune
		tmp   string

		flips    = map[rune]rune{'}': '{', ']': '[', ')': '('}
		quotes   = "\"'`"
		brackets = "({["
	)

	if n == 0 {
		return r
	} else if n == 1 {
		return []string{str}
	}

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
	for i := 0; i < utf8.RuneCountInString(str); i++ {
		char = rune(str[i])
		if level == 0 && contains(quotes+brackets, char) {
			host, level = char, level+1
		} else if contains(quotes, host, char) {
			level, host = 0, 0
		} else if contains(brackets, host, flips[char]) {
			level--
			if level <= 0 {
				level, host = 0, 0
			}
		} else if level == 0 {
			endpoint := i + utf8.RuneCountInString(sep)
			if endpoint > len(str) {
				endpoint = len(str)
			}

			if sep == str[i:endpoint] {
				i += utf8.RuneCountInString(sep) - 1
				r = append(r, tmp)
				tmp = ""
				if n > 0 && n == len(r)+1 {
					tmp = str[endpoint:]
					break
				}
				continue
			}
		}

		tmp += string(char)
	}

	// Add last piece to the result.
	if len(tmp) != 0 || string(char) == sep {
		r = append(r, tmp)
	}

	return
}

// The removeInlineComment removes the comment in the string.
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

// The parseExpression breaks expression into key and value, ignore
// comments and any spaces.
//
// Note: value must be an expression.
func parseExpression(exp string) (key, value string, err error) {
	var (
		quote  = "\""
		marker = fmt.Sprintf("<::%d::>", time.Now().Unix())
	)

	// Get key.
	// Remove `export` prefix, `=` suffix and trim spaces.
	tmp := keyRgx.FindStringSubmatch(exp)
	if len(tmp) < 2 {
		err = fmt.Errorf("missing variable name for: %s (`%v`)", exp, tmp)
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
