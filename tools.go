package env

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"
)

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
	file, err := os.OpenFile(filename, os.O_RDONLY, 0)
	if err != nil {
		return err
	}
	defer file.Close()

	entries, err := scanEntries(file, forced)
	if err != nil {
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
			if expand && e.expandable() && strings.Contains(value, "$") {
				value = os.ExpandEnv(value)
			}
			if err := os.Setenv(e.key, value); err != nil {
				return err
			}
		}
	}

	return nil
}

// The rawEntry is a parsed but not-yet-expanded key/value entry. The quote is
// the kind of quote that wrapped the value (0 if unquoted); it decides whether
// variable expansion applies.
type rawEntry struct {
	key   string
	value string
	quote rune
}

// expandable reports whether the value may have ${var}/$var expanded: only
// unquoted and double-quoted values; single quotes and backticks are literal.
func (e rawEntry) expandable() bool {
	return e.quote != '\'' && e.quote != '`'
}

// The scanEntries reads r and parses it into ordered raw entries, honouring
// multiline quoted values. When forced is true malformed lines are skipped
// instead of returning an error.
func scanEntries(r io.Reader, forced bool) ([]rawEntry, error) {
	var entries []rawEntry

	scanner := bufio.NewScanner(r)
	// Start small (like the default) but raise the ceiling so long values
	// (PEM chains, keys, JWTs, base64 blobs) are not rejected with
	// "token too long"; the buffer grows on demand up to the ceiling.
	scanner.Buffer(make([]byte, 0, 4096), 10*1024*1024)
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
			return nil, err
		}

		entries = append(entries, rawEntry{key: key, value: value, quote: quote})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}

// The parse reads r into a map of key/value pairs. When expand is true,
// ${var}/$var references in unquoted and double-quoted values are resolved
// against the already-parsed keys and, as a fallback, the process environment.
func parse(r io.Reader, expand bool) (map[string]string, error) {
	entries, err := scanEntries(r, false)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string, len(entries))
	lookup := func(key string) string {
		if v, ok := result[key]; ok {
			return v
		}
		return os.Getenv(key)
	}

	for _, e := range entries {
		value := e.value
		if expand && e.expandable() && strings.Contains(value, "$") {
			value = os.Expand(value, lookup)
		}
		result[e.key] = value
	}

	return result, nil
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

		flips    = map[rune]rune{'}': '{', ']': '[', ')': '('}
		quotes   = "\"'`"
		brackets = "({["
	)

	if n == 0 {
		return r
	}
	if n == 1 {
		return []string{str}
	}
	if str == "" {
		return r // an empty value yields no elements
	}
	if sep == "" {
		return []string{str} // an empty separator cannot split
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

	// Work on byte offsets of the original string: runes are decoded only to
	// drive the grouping state machine, but segments are cut from the original
	// bytes. This keeps valid UTF-8 correct, preserves invalid bytes verbatim
	// and matches multi-byte separators exactly.
	r = make([]string, 0, strings.Count(str, sep)+1)

	segStart := 0
	for i := 0; i < len(str); {
		char, size := utf8.DecodeRuneInString(str[i:])

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
		case level == 0 && strings.HasPrefix(str[i:], sep):
			r = append(r, str[segStart:i])
			i += len(sep)
			segStart = i
			if n > 0 && len(r) == n-1 {
				// The last element is the unsplit remainder.
				return append(r, str[segStart:])
			}
			continue
		}

		i += size
	}

	// The final segment (an empty trailing field when the string ends with
	// the separator).
	return append(r, str[segStart:])
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

	if quote == 0 {
		// For an unquoted value a '#' starts an inline comment only when it is
		// preceded by whitespace. A '#' at the start of the value or inside a
		// token is literal (so values like a hex colour #fff, a URL fragment
		// or pass#word are preserved). Only the comment is removed.
		if i := inlineCommentIndex(value); i >= 0 {
			value = strings.TrimRight(value[:i], " \t")
		}
	} else if quote != 0 {
		// Extract the quoted content with a single escape-aware pass:
		// find the matching closing quote (a backslash escapes the next
		// character) and drop anything after it (an inline comment).
		content, ok := parseQuoted(value, byte(quote))
		if !ok {
			err = fmt.Errorf("incorrect value: %s", value)
			return
		}

		if quote == '"' {
			// Double quotes interpret escape sequences (\n, \t, \r, \\, \").
			value = expandEscapes(content)
		} else {
			// Single quotes and backticks are literal; only the escaped
			// quote itself is unescaped (\' -> ', \` -> `).
			value = strings.ReplaceAll(content, "\\"+string(quote), string(quote))
		}
	}

	return
}

// The parseQuoted extracts the content of a quoted value. The string s must
// start with the quote byte; the scan is escape-aware (a backslash escapes
// the next character) and stops at the first unescaped closing quote.
// It returns the content between the quotes and true, or false if there is
// no matching closing quote.
func parseQuoted(s string, quote byte) (string, bool) {
	for i := 1; i < len(s); i++ {
		switch s[i] {
		case '\\':
			i++ // skip the escaped character
		case quote:
			return s[1:i], true
		}
	}

	return "", false
}

// The inlineCommentIndex returns the index of the '#' that starts an inline
// comment in an unquoted value - that is, the first '#' preceded by whitespace.
// A '#' at the start of the value or inside a token is literal, so it returns
// -1 in that case.
func inlineCommentIndex(s string) int {
	for i := 1; i < len(s); i++ {
		if s[i] == '#' && (s[i-1] == ' ' || s[i-1] == '\t') {
			return i
		}
	}

	return -1
}
