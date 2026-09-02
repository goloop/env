package env

import "strings"

// Expansion replaces ${NAME} and $NAME with the value of a variable.
//
// A reference is recognised only when NAME is a name this format could
// have defined in the first place - validKeyName, that is
// [A-Za-z_][A-Za-z0-9_]* - and everything else keeps its "$" as written.
//
// The standard library's os.Expand implements the shell's substitution
// language instead, and the shell has positional parameters. A .env file
// has none, but it does have prices, awk and sed one-liners, regular
// expressions and passwords, so under os.Expand a value quietly lost
// part of itself:
//
//	PRICE=cost: $100          became "cost: 00"     ($1 taken as a name)
//	DISCOUNT="save $5 today"  became "save  today"
//	AWK=$1 == "x"             became " == \"x\""
//
// Losing part of a value without a word is the worst thing this package
// can do, so a name here has to look like a name.
//
// There is no escape for a literal "$": the file format already has one,
// which is single quotes, and a second one layered on top would be a
// puzzle. A doubled "$" is left alone as well, which is not an escape -
// nothing is removed - but is what keeps a password like "pa$$word"
// intact without anyone having to know a rule.

// expandVars replaces variable references in s, taking values from
// lookup. A reference to a name lookup does not know becomes empty, as
// it always has.
func expandVars(s string, lookup func(string) string) string {
	if !strings.Contains(s, "$") {
		return s
	}

	var b strings.Builder
	b.Grow(len(s))

	for i := 0; i < len(s); {
		if s[i] != '$' {
			b.WriteByte(s[i])
			i++
			continue
		}

		// A doubled "$" never starts a reference.
		if i+1 < len(s) && s[i+1] == '$' {
			b.WriteString("$$")
			i += 2
			continue
		}

		rest := s[i+1:]

		if strings.HasPrefix(rest, "{") {
			end := strings.IndexByte(rest, '}')
			// A braced group that does not spell a name is text. The
			// alternative, eating it, is how "${" at the end of a value
			// used to take the rest of the line with it.
			if end < 0 || !validKeyName(rest[1:end]) {
				b.WriteByte('$')
				i++
				continue
			}
			b.WriteString(lookup(rest[1:end]))
			i += 1 + end + 1
			continue
		}

		n := 0
		for n < len(rest) && isKeyByte(rest[n], n == 0) {
			n++
		}
		if n == 0 {
			b.WriteByte('$')
			i++
			continue
		}
		b.WriteString(lookup(rest[:n]))
		i += 1 + n
	}

	return b.String()
}
