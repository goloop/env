package env

import (
	"regexp"
)

// The keyRgx is regular expression to verify
// the correctness name of the key.
var keyRgx = regexp.MustCompile(`^[A-Za-z_]{1}\w*$`)

// The tagGroup is the tag group of a field.
type tagGroup struct {
	key   string
	value string
	sep   string
}

// The isValid returns true if key name is valid.
func (tg tagGroup) isValid() bool {
	return keyRgx.Match([]byte(tg.key))
}

// The isIgnored returns true if key name is ignoreKeyValue or incorrect.
func (tg tagGroup) isIgnored() bool {
	return !tg.isValid() || tg.key == ignoreKeyValue
}
