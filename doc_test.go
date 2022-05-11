package env

import (
	"strings"
	"testing"
)

// TestVersion tests the package version.
//
// Note: Each time when you change the major version,
// you need to fix the expected const in the test.
func TestVersion(t *testing.T) {
	const expected = "v1." // change it for major version

	version := Version()
	if strings.HasPrefix(version, expected) != true {
		t.Error("incorrect version")
	}

	if len(strings.Split(version, ".")) != 3 {
		t.Error("version format should be as " +
			"v{major_version}.{minor_version}.{patch_version}")
	}
}
