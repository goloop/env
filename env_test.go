package env

import (
	"testing"
)

// TestReadParseStoreOpen tests function to open a nonexistent file.
func TestLoadReadParseStoreOpen(t *testing.T) {
	err := ReadParseStore("./fixtures/nonexist.env", false, false, false)
	if err == nil {
		t.Error("expected an error for open a nonexistent file")
	}
}

// TestReadParseStoreExported checks the parsing of the
// env-file with the `export` command.
func TestReadParseStoreExported(t *testing.T) {
	var tests = map[string]string{
		"KEY_0": "value 0",
		"KEY_1": "value 1",
		"KEY_2": "value_2",
		"KEY_3": "value_0:value_1:value_2:value_3",
	}

	// Load env-file.
	Clear()
	err := ReadParseStore("./fixtures/exported.env", false, false, false)
	if err != nil {
		t.Error(err.Error())
	}

	// Compare with sample.
	for key, value := range tests {
		if v := Get(key); value != v {
			t.Errorf("incorrect value for `%s` key: `%s`!=`%s`", key, value, v)
		}
	}
}

// TestReadParseStoreComments checks the parsing of the
// env-file with the comments and empty strings.
func TestReadParseStoreComments(t *testing.T) {
	var tests = map[string]string{
		"KEY_0": "value 0",
		"KEY_1": "value 1",
		"KEY_2": "value_2",
		"KEY_3": "value_3",
		"KEY_4": "value_4:value_4:value_4",
		"KEY_5": `some text with # sharp sign and "escaped quotation" mark`,
	}

	// Load env-file.
	Clear()
	err := ReadParseStore("./fixtures/comments.env", false, false, false)
	if err != nil {
		t.Error(err.Error())
	}

	// Compare with sample.
	for key, value := range tests {
		if v := Get(key); value != v {
			t.Errorf("incorrect value for `%s` key: `%s`!=`%s`", key, value, v)
		}
	}
}

// TestReadParseStoreWorngEqualKey tests problem with
// spaces before the equal sign.
func TestReadParseStoreWorngEqualKey(t *testing.T) {
	err := ReadParseStore("./fixtures/wrongequalkey.env", false, false, false)
	if err == nil {
		t.Error("expected an error")
	}
}

// TestReadParseStoreWorngEqualValue tests problem with
// space after the equal sign.
func TestReadParseStoreWorngEqualValue(t *testing.T) {
	err := ReadParseStore("./fixtures/wrongequalvalue.env", false, true, false)
	if err == nil {
		t.Error("expected an error")
	}
}

// TestReadParseStoreIgnoreWorngEntry tests to force loading with
// the incorrect lines.
func TestReadParseStoreIgnoreWorngEntry(t *testing.T) {
	var forced = true
	var tests = map[string]string{
		"KEY_0": "value_0",
		"KEY_1": "value_1",
		"KEY_4": "value_4",
		"KEY_5": "value",
		"KEY_6": "777",
		"KEY_7": "${KEY_1}",
	}

	// Load env-file.
	Clear()
	err := ReadParseStore("./fixtures/wrongentries.env", false, false, forced)
	if err != nil {
		t.Error(err.Error())
	}

	// Compare with sample.
	for key, value := range tests {
		if v := Get(key); value != v {
			t.Errorf("incorrect value for `%s` key: `%s`!=`%s`", key, value, v)
		}
	}
}

// TestReadParseStoreVariables tests replacing variables on real values.
func TestReadParseStoreVariables(t *testing.T) {
	var expand = true
	var tests = map[string]string{
		"KEY_0": "value_0",
		"KEY_1": "value_1",
		"KEY_2": "value_001",
		"KEY_3": "value_001->correct value",
		"KEY_4": "value_0value_001",
	}

	// Load env-file.
	Clear()
	err := ReadParseStore("./fixtures/variables.env", expand, false, false)
	if err != nil {
		t.Error(err.Error())
	}

	// Compare with sample.
	for key, value := range tests {
		if v := Get(key); value != v {
			t.Errorf("incorrect value for `%s` key: `%s`!=`%s`", key, value, v)
		}
	}
}

// TestReadParseStoreNotUpdate tests variable update protection.
func TestReadParseStoreNotUpdate(t *testing.T) {
	var (
		update = false
		err    error
	)

	// Set test data.
	Clear()
	err = Set("KEY_0", "") // set empty string
	if err != nil {
		t.Error(err)
	}

	// Read simple env-file with KEY_0.
	err = ReadParseStore("./fixtures/simple.env", false, update, false)
	if err != nil {
		t.Error(err.Error())
	}

	// Tests.
	if v := Get("KEY_0"); v != "" {
		t.Error("error: the value has been updated")
	}
}

// TestReadParseStoreUpdate tests variable update.
func TestReadParseStoreUpdate(t *testing.T) {
	var (
		update = true
		err    error
	)

	// Set test data.
	Clear()
	err = Set("KEY_0", "") // set empty string
	if err != nil {
		t.Error(err)
	}

	// Read simple env-file with KEY_0.
	err = ReadParseStore("./fixtures/simple.env", false, update, false)
	if err != nil {
		t.Error(err.Error())
	}

	// Tests.
	if v := Get("KEY_0"); v != "value 0" {
		t.Error("error: variable won't update")
	}
}

// TestLoad tests Load function.
func TestLoad(t *testing.T) {
	var err error

	Clear()
	err = Set("KEY_0", "default")
	if err != nil {
		t.Error(err)
	}

	// Load env-file.
	err = Load("./fixtures/variables.env")
	if err != nil {
		t.Error(err)
	}

	// Variable update protection.
	if Get("KEY_0") != "default" {
		t.Error("the existing variable has been overwritten")
	}

	// Setting a new variable.
	if Get("KEY_1") != "value_1" {
		t.Error("data wasn't loaded")
	}

	// Expand test.
	if v := Get("KEY_2"); v != "default01" { // KEY_0 not overwritten
		t.Errorf("expected value `default01` but `%s`.", v)
	}
}

// TestLoadSafe tests LoadSafe function.
func TestLoadSafe(t *testing.T) {
	var err error

	Clear()
	err = Set("KEY_0", "default")
	if err != nil {
		t.Error(err)
	}

	// Load env-file.
	err = LoadSafe("./fixtures/variables.env")
	if err != nil {
		t.Error(err)
	}

	// Expand test.
	if v := Get("KEY_2"); v != "${KEY_0}01" { // LoadSafe don't expand vars
		t.Errorf("expected value `${KEY_0}01` but `%s`.", v)
	}
}

// TestUpdate tests Update function.
func TestUpdate(t *testing.T) {
	var err error

	Clear()
	err = Set("KEY_0", "default")
	if err != nil {
		t.Error(err)
	}

	// Load env-file.
	err = Update("./fixtures/variables.env")
	if err != nil {
		t.Error(err)
	}

	// Variable update protection.
	if Get("KEY_0") == "default" {
		t.Error("existing variable has not overwritten")
	}

	// Setting a new variable.
	if Get("KEY_1") != "value_1" {
		t.Error("data didn't loaded")
	}

	// Expand test.
	if v := Get("KEY_2"); v != "value_001" { // KEY_0 not overwritten
		t.Errorf("expected value `value_001` but `%s`.", v)
	}
}

// TestUpdateSafe tests UpdateSafe function.
func TestUpdateSafe(t *testing.T) {
	var err error

	Clear()
	err = Set("KEY_0", "default")
	if err != nil {
		t.Error(err)
	}

	// Load env-file.
	err = UpdateSafe("./fixtures/variables.env")
	if err != nil {
		t.Error(err)
	}

	// Expand test.
	if v := Get("KEY_2"); v != "${KEY_0}01" { // UpdateSafe don't expand vars
		t.Errorf("expected value `${KEY_0}01` but `%s`.", v)
	}
}

// TestExists tests Exists function.
func TestExist(t *testing.T) {
	var (
		err   error
		tests = [][]string{
			{"KEY_0", "default"},
			{"KEY_1", "default"},
		}
	)

	Clear()
	for _, item := range tests {
		err = Set(item[0], item[1])
		if err != nil {
			t.Error(err)
		}
	}

	// Variables is exists.
	if !Exists("KEY_0") || !Exists("KEY_0", "KEY_1") {
		t.Error("expected value `ture` but `false`")
	}

	// Variables doesn't exists.
	if Exists("KEY_2") || Exists("KEY_0", "KEY_1", "KEY_2") {
		t.Error("expected value `false` but `true`")
	}
}
