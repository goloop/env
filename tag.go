package env

// The tagGroup is the tag group of a field.
type tagGroup struct {
	key   string
	value string
	sep   string
}

// The isValid returns true if key name is valid.
func (tg tagGroup) isValid() bool {
	return validKeyRgx.Match([]byte(tg.key))
}

// The isIgnored returns true if key name is defValueIgnored or incorrect.
func (tg tagGroup) isIgnored() bool {
	return !tg.isValid() || tg.key == defValueIgnored
}
