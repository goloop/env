package env

// The tagGroup represents the tag group of a field.
type tagGroup struct {
	key   string // key name
	value string // key value
	sep   string // separator between value items (for sequences)
}

// The isValid method returns true if the key name is valid.
func (tg tagGroup) isValid() bool {
	return validKeyRgx.MatchString(tg.key)
}

// The isIgnored method returns true if the key name is
// defValueIgnored or incorrect.
func (tg tagGroup) isIgnored() bool {
	return !tg.isValid() || tg.key == defValueIgnored
}
