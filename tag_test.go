package env

import (
	"testing"
)

// TestTagGroupl tests tagGroup.
func TestTagGroupl(t *testing.T) {
	// Incorrect tag group.
	tg := tagGroup{key: defValueIgnored}
	if ignored := tg.isIgnored(); !ignored {
		t.Error("should be ignored")
	}

	if valid := tg.isValid(); valid {
		t.Error("should be invalid")
	}

	// Correct tag group.
	tg = tagGroup{key: "NORMAL_KEY"}
	if ignored := tg.isIgnored(); ignored {
		t.Error("should be normal")
	}

	if valid := tg.isValid(); !valid {
		t.Error("should be valid")
	}
}
