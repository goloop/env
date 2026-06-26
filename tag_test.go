package env

import (
	"testing"
)

// TestTagGroupl tests tagGroup.
func TestTagGroupl(t *testing.T) {
	// Incorrect tag group.
	tg := tagGroup{key: defValueIgnored}
	if valid := tg.isValid(); valid {
		t.Error("should be invalid")
	}

	// Correct tag group.
	tg = tagGroup{key: "NORMAL_KEY"}
	if valid := tg.isValid(); !valid {
		t.Error("should be valid")
	}
}
