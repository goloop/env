package env

import (
	"strings"
	"unicode/utf8"
)

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
