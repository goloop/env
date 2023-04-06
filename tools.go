package env

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/sync/errgroup"
)

// The sts converts a sequence (slice or array) of any type to a string.
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
// Note: This function is not used as an env function subsystem,
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

// The fts returns data as string from the struct or pointer
// on struct by field name. If name gets the name of the key-like
// (with '_' separator, such a delimiter is used in environment variables),
// for example KEY_A it will be converted to go-like name - KeyA.
//
// If the specified field is missing from the structure,
// an empty string will be returned.
//
// Note: This function is not used as an env function subsystem,
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
		name = strings.ReplaceAll(
			strings.Title(
				strings.ToLower(
					strings.ReplaceAll(name, "_", " "),
				),
			), " ", "")
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

// The isEmpty returns true if string from environment file
// contains separators or comment only.
func isEmpty(str string) bool {
	// If string is empty, return true.
	if len(str) == 0 {
		return true
	}

	// Check the first character in the string:
	// - if it is the hash character '#' - the string is a comment;
	// - if the first character is not a separator (the string is not empty);
	// - if the first character is a separator, check the string with
	//   using a regular expression.

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
	return emptyLineRgx.Match([]byte(str))
}

// The readParseStore reads env-file, parses this one by the key and value,
// and stores in environment. It's flexible function that can be used to
// build more specific tools.
//
// Arguments:
//
//	filename path to the env-file;
//	expand   if true, replaces ${key} or $key on the values
//	         from the current environment variables;
//	update   if true, overwrites the value that has already been
//	         set in the environment to new one from the env-file;
//	forced   if true, ignores wrong entries in the env-file and
//	         loads all correct options, without file parsing exception.
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
	// Define a structure for the line
	// that is read from the env-file.
	type line struct {
		text   string // raw string from env-file
		number int    // number of line in the env-file
	}

	// Define a structure for the result,
	// which is a parsed line from the env-file.
	type output struct {
		expanded bool   // true if the value can be expanded
		value    string // key value
		line     line   // raw line object
		key      string // key name
	}

	var (
		outputs []output   // slice of parsed lines
		mu      sync.Mutex // mutex for outputs
	)

	// Try to open env-file in read only mode.
	file, err := os.OpenFile(filename, os.O_RDONLY, 0)
	if err != nil {
		return err
	}
	defer file.Close()

	// Parse env-file using goroutines.
	// We use errgroup as a better way to group goroutines and context to
	// stop all goroutines from executing if an error is detected in a file.
	lines := make(chan line) // channel for lines from env-file
	ctx, cancel := context.WithCancel(context.Background())
	eg, ctx := errgroup.WithContext(ctx)
	defer cancel()

	// Create some goroutines (parallelTasks)
	// to parsing lines from an env-file.
	for i := 0; i < parallelTasks; i++ {
		eg.Go(func() error {
			for line := range lines {
				// Ignore empty string or comments.
				if isEmpty(line.text) {
					continue
				}

				// Parse expression.
				// The string containing the expression must be of the
				// format as: [export] KEY=VALUE [# Comment]
				key, value, err := parseExpression(line.text)
				if err != nil {
					if forced {
						continue // ignore error in the line
					} else {
						cancel() // stop other goroutines too
						return err
					}
				}

				// Check whether to execute os.Expand only in expand mode,
				// otherwise set false for all exceptions.
				expanded := false
				if expand {
					expanded = strings.Contains(value, "$")
				}

				// Save the result.
				mu.Lock()
				outputs = append(outputs, output{
					expanded: expanded,
					value:    value,
					line:     line,
					key:      key,
				})
				mu.Unlock()
			}

			return nil
		})
	}

	// Read the file line by line and send it to the channel.
	// The done channel informs about the end of reading
	// the file and the existing reading error.
	done := make(chan error)
	go func() {
		number := 0 // file line number
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			lines <- line{text: scanner.Text(), number: number}
			number++
		}
		done <- scanner.Err()
	}()

	// Wait for the end of reading the file and check for errors.
	err = <-done
	close(lines)
	if err != nil {
		return err
	}

	// Wait for all goroutines to finish and check for errors.
	err = eg.Wait()
	if err != nil && !errors.Is(err, context.Canceled) {
		return err
	}

	// Sort the results by line number.
	if expand {
		sort.Slice(outputs, func(i, j int) bool {
			return outputs[i].line.number < outputs[j].line.number
		})
	}

	// Set the environment variables.
	for _, o := range outputs {
		if _, ok := os.LookupEnv(o.key); update || !ok {
			if expand && o.expanded {
				o.value = os.ExpandEnv(o.value)
			}

			err := os.Setenv(o.key, o.value)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// The splitN splits the string at the specified rune-marker ignoring the
// position of the marker inside of the group: `...`, '...', "..."
// and (...), {...}, [...].
//
// Arguments:
//
//	str  data;
//	sep  element separator;
//	n    the number of strings to be returned by the function.
//	     It can be any of the following:
//	     - n is equal to zero (n == 0): The result is nil, i.e, zero
//	       sub strings. An empty list is returned;
//	     - n is greater than zero (n > 0): At most n sub strings will be
//	       returned and the last string will be the unsplit remainder;
//	     - n is less than zero (n < 0): All possible substring
//	       will be returned.
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

// The removeInlineComment removes the comment in the env-string.
// Only in strings where the value is enclosed in quotes.
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

// The parseExpression breaks expression into key and value, ignore
// comments and any spaces.
//
// Note: value must be an env-expression.
func parseExpression(exp string) (key, value string, err error) {
	// Get key name.
	// Remove `export` prefix, `=` suffix and trim spaces.
	tmp := keyRgx.FindStringSubmatch(exp)
	if len(tmp) < 2 {
		err = fmt.Errorf("missing variable name for: %s (`%v`)", exp, tmp)
		return
	}

	key = tmp[1]

	// Get value of the key.
	// ... the `=` sign in the string.
	if pos := strings.IndexByte(exp, '='); pos != -1 {
		value = exp[pos:]
		if !valueRgx.MatchString(value) {
			err = fmt.Errorf("incorrect value: %s", value)
			return
		}
	} else {
		err = fmt.Errorf("missing `=` sign in the expression: %s", exp)
		return
	}

	value = strings.TrimSpace(value[1:])

	// Check the value for quotes.
	var quote rune
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
		marker := fmt.Sprintf("<::%d::>", time.Now().Unix())

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
	}

	return
}
