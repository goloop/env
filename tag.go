package env

import "strings"

// The tagGroup represents the tag group of a field.
type tagGroup struct {
	key      string // key name
	value    string // key value
	sep      string // separator between value items (for sequences)
	layout   string // time layout for time.Time fields
	required bool   // field must be present in the source
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

// The parseEnvTag splits the value of the env tag into the key name and the
// inline flags. The first comma-separated item is the name; the rest are
// flags (currently only "required").
func parseEnvTag(tag string) (name string, required bool) {
	parts := strings.Split(tag, ",")
	name = strings.TrimSpace(parts[0])
	for _, flag := range parts[1:] {
		if strings.TrimSpace(flag) == tagFlagRequired {
			required = true
		}
	}

	return name, required
}
