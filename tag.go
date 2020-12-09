package env

import (
	"regexp"
	"strings"
)

// The correctKeyRgx is regular expression
// to verify the correctness of the key name.
var nameRgx = regexp.MustCompile(`^[A-Za-z_]{1}\w*$`)

// The tagArgs is struct of the fields of the tag for env's package.
// Tag example: "env:[kye,[value,[sep]]]" where:
//    key - key name in the environment;
//    value - default value for this key;
//    sep - list separator (default ":").
type tagArgs struct {
	Key   string
	Value string
	Sep   string
}

// The isValid returns true if key name is valid.
func (args tagArgs) IsValid() bool {
	return nameRgx.Match([]byte(args.Key))
}

// The isIgnored returns true if key name is "-" or incorrect.
func (args tagArgs) IsIgnored() bool {
	return !args.IsValid() || args.Key == "-"
}

// The getTagValues returns field valueas as array: [key, value, sep].
func getTagValues(tag string) (r [3]string) {
	var chunks = splitN(tag, ",", 3)
	for i, c := range chunks {
		// Save the last piece without changed.
		if i == len(r)-1 {
			if v := strings.Join(chunks[i:], ","); v != "" {
				r[i] = v
			}
			break
		}

		r[i] = c
	}

	return
}

// The getTagArgs returns tagArgs object for tag.
// If key isn't sets in the tag, it will be assigned from the second argument.
//
// Examples:
//    getTagArgs("a,b,c", "default") // [a b c]
//    getTagArgs(",b,c", "default") // [default b c]
//    getTagArgs(",b", "default") // [default b :]
func getTagArgs(tag string, key string) *tagArgs {
	var args, v = &tagArgs{}, getTagValues(tag)

	// Key.
	args.Key = strings.Trim(v[0], " ")
	if len(args.Key) == 0 {
		args.Key = key
	}

	// Value.
	args.Value = strings.TrimRight(strings.TrimLeft(v[1], "({[\"'`"), ")}]\"'`")

	// Separator of the list.
	args.Sep = v[2]
	if len(args.Sep) == 0 {
		args.Sep = ":"
	}

	return args
}
