package env

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"unicode"
)

// The sts function converts a slice or an array of any type to a string.
// If the value is not a slice or array, an error message will be returned.
//
// The second argument to the function specifies the separator to be
// inserted between the elements of the sequence in the result string.
//
// Examples:
//
//	sts([]int{1, 2, 3}, ",")          // "1,2,3"
//	sts([]string{"1", "2", "3"}, ";") // "1;2;3"
//
// Note: This function is not used as an environment function subsystem,
// it is only used to test package functions.
func sts(seq interface{}, sep string) (string, error) {
	// Create a string builder to concatenate strings.
	var sb strings.Builder

	// Check the type of the input data.
	kind := reflect.TypeOf(seq).Kind()
	if kind != reflect.Array && kind != reflect.Slice {
		return "", errors.New("input is not a slice or array")
	}

	// Convert the sequence to a string.
	s := reflect.ValueOf(seq)
	for i := 0; i < s.Len(); i++ {
		if i > 0 {
			sb.WriteString(sep)
		}
		fmt.Fprintf(&sb, "%v", s.Index(i))
	}

	return sb.String(), nil
}

// The fts function returns data as a string from the struct or pointer
// on struct by field name. If the name gets the name of the key-like
// (with '_' separator, such as delimiter used in environment variables),
// for example KEY_A, it will be converted to a Go-style name - KeyA.
//
// If the specified field is missing from the structure,
// an empty string will be returned.
//
// Note: This function is not used as an environment function subsystem,
// it is only used to test package functions.
func fts(v interface{}, name, sep string) string {
	// Check if v is a struct. And if v is a pointer to a structure,
	// we need to get the structure it refers to.
	r := reflect.ValueOf(v)
	if r.Kind() == reflect.Ptr {
		r = r.Elem()
	}

	if r.Kind() != reflect.Struct {
		return ""
	}

	// Correct the field name to go-style.
	if strings.Contains(name, "_") {
		// Split words, capitalize words and join it.
		name = strings.NewReplacer("_", " ").Replace(name)
		name = strings.Title(strings.ToLower(name))
		name = strings.ReplaceAll(name, " ", "")
	}

	// Check if the struct has a field with the given name.
	f := reflect.Indirect(r).FieldByName(name)
	if !f.IsValid() {
		return ""
	}

	// Convert the field value to a string.
	var value string
	switch f.Kind() {
	case reflect.Slice, reflect.Array:
		if sep == "" {
			value = fmt.Sprintf("%v", f)
		} else {
			value = strings.Join(strings.Fields(fmt.Sprint(f)), sep)
		}
		value = strings.Trim(value, "[]")
	default:
		value = fmt.Sprintf("%v", f)
	}

	return value
}

// The isEmpty function returns true if the string from the environment file
// contains separators or comments only.
func isEmpty(str string) bool {
	// If string is empty, return true.
	if len(str) == 0 {
		return true
	}

	// Check the first character in the string:
	//  - if it is the hash character '#' - the string is a comment;
	//  - if the first character is not a separator (the string is not empty);
	//  - if the first character is a separator, check the string with
	//    using a regular expression.

	// Get first rune from string.
	firstRune := []rune(str)[0]

	// If first character is a comment - string is empty.
	if firstRune == '#' {
		return true
	}

	// If first character is not a separator - string is not empty.
	if !unicode.IsSpace(firstRune) {
		return false
	}

	// A complex string to be tested using a regular expression.
	return emptyLineRgx.MatchString(str)
}

// The readParseStore reads env-file, parses this one by the key and value,
// and stores in environment. It's flexible function that can be used to
// build more specific tools.
//
// Arguments:
//
//   - filename path to the env-file;
//   - expand   if true, replaces ${key} or $key on the values
//     from the current environment variables;
//   - update   if true, overwrites the value that has already been
//     set in the environment to new one from the env-file;
//   - forced   if true, ignores wrong entries in the env-file and
//     loads all correct options, without file parsing exception.
//
// Examples:
//
// There is `.env` env-file that contains:
//
//	# .env file
//	HOST=0.0.0.0
//	PORT=80
//	EMAIL=$USER@goloop.one
//
// Some variables are already exists in the environment:
//
//	$ env | grep -E "USER|HOST"
//	USER=goloop
//	HOST=localhost
//
// To correctly load data from env-file followed by updating the environment:
//
//	env.ReadParseStore(".env", true, true, false)
//
//	// USER=goloop
//	// HOST=0.0.0.0
//	// PORT=80
//	// EMAIL=goloop@goloop.one
//
// Loading new keys to the environment without updating existing ones:
//
//	env.ReadParseStore(".env", true, false, false)
//
//	// USER=goloop
//	// HOST=localhost          <= hasn't been updated
//	// PORT=80
//	// EMAIL=goloop@goloop.one
//
// Don't change values that contain keys:
//
//	env.ReadParseStore(".env", false, true, false)
//
//	// USER=goloop
//	// HOST=0.0.0.0
//	// PORT=80
//	// EMAIL=$USER@goloop.one  <= $USER hasn't been changed to real value
//
// Loading data from a damaged env-file. If the configuration env-file is used
// by other applications and can have incorrect key/value, it can be ignored.
// For example env-file contains incorrect key `1BC` (the variable name can't
// start with a digit):
//
//	# .env file
//	HOST=0.0.0.0
//	PORT=80
//	1BC=NO                     # <= incorrect variable
//	EMAIL=$USER@goloop.one
//
// There will be an error loading this file:
//
//	err := env.ReadParseStore(".env", true, true, false)
//	if err != nil {
//	    log.Panic(err) // panic: missing variable name
//	}
//
// But we can use force method to ignore this line:
//
//	// ... set forced as true (last argument)
//	err := env.ReadParseStore(".env", true, true, true)
//
//	// Now the err variable is nil
//	// and environment has:
//	// USER=goloop
//	// HOST=0.0.0.0
//	// PORT=80
//	// EMAIL=goloop@goloop.one
func readParseStore(filename string, expand, update, forced bool) error {
	// A parsed key/value entry from the env-file.
	type entry struct {
		key      string
		value    string
		expanded bool // value may contain ${var}/$var to expand
	}

	// Try to open env-file in read only mode.
	file, err := os.OpenFile(filename, os.O_RDONLY, 0)
	if err != nil {
		return err
	}
	defer file.Close()

	// Read and parse the file sequentially. Parsing a small env-file is a
	// few string operations per line, so goroutines/channels would only add
	// coordination overhead; the apply step below must stay ordered anyway
	// (expansion depends on earlier keys already being set).
	var entries []entry

	// Read and parse line by line.
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		text := scanner.Text()

		// Multiline values: if the value opens a quote that is not closed
		// on this physical line, keep reading and join the lines with "\n"
		// until the matching closing quote is found.
		if q := multilineQuote(text); q != 0 {
			var b strings.Builder
			b.WriteString(text)
			for scanner.Scan() {
				cont := scanner.Text()
				b.WriteByte('\n')
				b.WriteString(cont)
				if countUnescapedQuote(cont, q)%2 != 0 {
					break // closing quote found
				}
			}
			text = b.String()
		}

		// Ignore empty strings and comments.
		if isEmpty(text) {
			continue
		}

		// Parse expression of the form: [export] KEY=VALUE [# comment].
		key, value, quote, err := parseExpression(text)
		if err != nil {
			if forced {
				continue // ignore the wrong line
			}
			return err
		}

		// Variable expansion (${var}/$var) applies to unquoted and
		// double-quoted values only. Single quotes and backticks are
		// literal per the dotenv specification, so `$` stays as-is.
		expanded := expand && quote != '\'' && quote != '`' &&
			strings.Contains(value, "$")

		entries = append(entries, entry{key: key, value: value, expanded: expanded})
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	// Apply entries to the environment in file order. Order matters in
	// expand mode: KEY_1=${KEY_0} must see the value KEY_0 had at that
	// point (e.g. before a later KEY_0 override):
	//
	//	KEY_0=VALUE_0
	//	KEY_1=${KEY_0}7   // -> VALUE_07
	//	KEY_0=VALUE_1     // overrides KEY_0 afterwards
	for _, e := range entries {
		if _, ok := os.LookupEnv(e.key); update || !ok {
			value := e.value
			if e.expanded {
				value = os.ExpandEnv(value)
			}
			if err := os.Setenv(e.key, value); err != nil {
				return err
			}
		}
	}

	return nil
}

// The splitN function splits the string at the specified rune separator,
// ignoring the position of the separator inside of the group:
// `...`, '...', "..." and (...), {...}, [...].
//
// Arguments:
//
//   - str data;
//   - sep element separator;
//   - n   the number of strings to be returned by the function.
//
// The n can be any of the following:
//  1. n is equal to zero (n == 0): The result is nil, i.e, zero
//     sub strings. An empty list is returned;
//  2. n is greater than zero (n > 0): At most n sub strings will be
//     returned and the last string will be the unsplit remainder;
//  3. n is less than zero (n < 0): All possible substring
//     will be returned.
//
// Examples:
//
//	splitN("a,b,c,d", ',', -1)     // ["a", "b", "c", "d"]
//	splitN("a,(b,c),d", ',', -1)   // ["a", "(b,c)", "d"]
//	splitN("'a,b',c,d", ',', -1)   // ["'a,b'", "c", "d"]
//	splitN("a,\"b,c\",d", ',', -1) // ["a", "\"b,c\"", "d"]
func splitN(str, sep string, n int) (r []string) {
	var (
		level int
		host  rune
		char  rune

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
		last := -1
		for _, sep := range separators {
			ir := strings.IndexRune(str, sep)
			if ir < 0 || (last >= 0 && last != ir) {
				return false
			}
			last = ir
		}

		return true
	}

	// Work on runes, not bytes: indexing str[i] by byte corrupts multi-byte
	// values and separators (e.g. Cyrillic, emoji). Converting to []rune once
	// also keeps the loop O(n) instead of O(n^2).
	runes := []rune(str)
	sepRunes := []rune(sep)
	sepLen := len(sepRunes)

	// The matchSep reports whether the separator occurs at position i.
	matchSep := func(i int) bool {
		if sepLen == 0 || i+sepLen > len(runes) {
			return false
		}
		for j := 0; j < sepLen; j++ {
			if runes[i+j] != sepRunes[j] {
				return false
			}
		}
		return true
	}

	// Pre-allocate based on the actual separator (not a hard-coded comma).
	r = make([]string, 0, strings.Count(str, sep)+1)

	// Split value.
	var sb strings.Builder
	for i := 0; i < len(runes); i++ {
		char = runes[i]
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
		case level == 0 && matchSep(i):
			r = append(r, sb.String())
			sb.Reset()
			if n > 0 && n == len(r)+1 {
				// The last element is the unsplit remainder.
				return append(r, string(runes[i+sepLen:]))
			}
			i += sepLen - 1
			continue
		}

		sb.WriteRune(char)
	}

	// Add last piece to the result (including a trailing empty
	// field when the string ends with the separator).
	if sb.Len() != 0 || string(char) == sep {
		r = append(r, sb.String())
	}

	return
}

// The expandEscapes interprets backslash escape sequences inside
// double-quoted values: \n, \t, \r become the corresponding control
// characters and \\ becomes a single backslash (\" is already handled
// during quote processing). Unknown escapes are left untouched.
func expandEscapes(s string) string {
	if !strings.Contains(s, "\\") {
		return s
	}

	var sb strings.Builder
	sb.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case 'n':
				sb.WriteByte('\n')
			case 't':
				sb.WriteByte('\t')
			case 'r':
				sb.WriteByte('\r')
			case '\\':
				sb.WriteByte('\\')
			case '"':
				sb.WriteByte('"')
			default:
				// Keep unknown escapes verbatim (backslash + char).
				sb.WriteByte('\\')
				sb.WriteByte(s[i+1])
			}
			i++
			continue
		}
		sb.WriteByte(s[i])
	}

	return sb.String()
}

// The countUnescapedQuote counts occurrences of the quote byte q in s that
// are not escaped by a preceding backslash (an even number of backslashes
// before the quote means it is not escaped).
func countUnescapedQuote(s string, q byte) int {
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] == q {
			bs := 0
			for j := i - 1; j >= 0 && s[j] == '\\'; j-- {
				bs++
			}
			if bs%2 == 0 {
				n++
			}
		}
	}

	return n
}

// The multilineQuote reports the opening quote byte (", ' or `) when the
// given line starts a quoted value whose quote is not closed on the same
// line (i.e. the start of a multiline value). It returns 0 otherwise.
func multilineQuote(line string) byte {
	// Comments and empty lines never start a value.
	if isEmpty(line) {
		return 0
	}

	pos := strings.IndexByte(line, '=')
	if pos == -1 {
		return 0
	}

	value := strings.TrimLeft(line[pos+1:], " \t")
	if value == "" {
		return 0
	}

	q := value[0]
	if q != '"' && q != '\'' && q != '`' {
		return 0
	}

	// The opening quote is unterminated when the number of unescaped
	// quotes on the line is odd.
	if countUnescapedQuote(value, q)%2 != 0 {
		return q
	}

	return 0
}

// The removeInlineComment function removes the comment in the env-string.
// It removes comments starting with the hash symbol (#) if they are not
// enclosed in quotes (single, double, or backquote).
//
// The value for quote can be as: single quote ('),
// double quote ("), and backquote (`).
func removeInlineComment(str string, q rune) string {
	// If the comment isn't in the string.
	// The environment file uses the lattice symbol (#) as a comment.
	if !strings.Contains(str, "#") {
		return str
	}

	var (
		quote  = string(q)     // quote as string
		escape = "\\" + quote  // escaped quote
		inside bool            // inside of the quote
		result strings.Builder // result string
	)

	// Remove the comment in the string.
	for i := 0; i < len(str); i++ {
		ch := str[i]

		switch {
		case ch == byte(q):
			if inside {
				// Check if the quote is escaped.
				if i > 0 && str[i-1] != '\\' {
					inside = false
				}
			} else {
				inside = true
			}
			result.WriteByte(ch)
		case ch == '#' && !inside:
			return strings.TrimSpace(result.String())
		case ch == '\\' && inside && i+1 < len(str) && str[i+1] == byte(q):
			// Escaping quotes inside a quoted line.
			result.WriteString(escape)
			i++
		default:
			result.WriteByte(ch)
		}
	}

	return result.String()
}

// The parseExpression function breaks an expression into a key and value,
// ignoring comments and any spaces. The value must be an env-expression.
//
// The returned quote is the kind of quote that wrapped the value (', " or `),
// or 0 if the value was unquoted. Callers use it to decide whether variable
// expansion applies (single quotes and backticks are literal).
func parseExpression(exp string) (key, value string, quote rune, err error) {

	// Get key name.
	// Remove `export` prefix, `=` suffix and trim spaces.
	tmp := keyRgx.FindStringSubmatch(exp)
	if len(tmp) < 2 {
		err = fmt.Errorf("missing variable name for: %s (`%v`)", exp, tmp)
		return
	}

	key = tmp[1]

	// Get value of the key.
	// Everything after the first `=` is the value. An empty value (`KEY=`)
	// is valid and yields an empty string; surrounding whitespace of an
	// unquoted value is trimmed (`KEY= value` -> `value`). Whitespace inside
	// quotes is preserved later during quote processing.
	pos := strings.IndexRune(exp, '=')
	if pos == -1 {
		err = fmt.Errorf("missing `=` sign in the expression: %s", exp)
		return
	}

	value = strings.TrimSpace(exp[pos+1:])

	// Check the value for quotes.
	if strings.HasPrefix(value, "'") {
		quote = '\''
	} else if strings.HasPrefix(value, "\"") {
		quote = '"'
	} else if strings.HasPrefix(value, "`") {
		quote = '`'
	}

	if quote == 0 && strings.Contains(value, "#") {
		// Split by sharp sign and for string without quotes -
		// the first element has the meaning only.
		chunks := strings.Split(value, "#")
		chunks = strings.Split(chunks[0], " ")
		value = strings.TrimSpace(chunks[0])
	} else if quote != 0 {
		// A unique marker for temporary replacement of quotation marks.
		buffer := make([]byte, 8)
		rand.Read(buffer)
		marker := "<::" + hex.EncodeToString(buffer) + "::>"

		// Replace escaped quotes, remove comment in the string,
		// check begin- and end- quotes and back escaped quotes.
		value = strings.Replace(value, fmt.Sprintf("\\%c", quote), marker, -1)
		value = removeInlineComment(value, quote)

		// Check begin- and end- quotes.
		if strings.Count(value, string(quote))%2 != 0 {
			err = fmt.Errorf("incorrect value: %s", value)
			return
		}

		// Remove begin- and end- quotes
		// ... change `\"` and `\'` to `"` and `'`.
		value = value[1 : len(value)-1]
		value = strings.Replace(value, marker, string(quote), -1)

		// In double-quoted values interpret escape sequences
		// (\n, \t, \r, \\). Single quotes and backticks stay literal.
		if quote == '"' {
			value = expandEscapes(value)
		}
	}

	return
}
