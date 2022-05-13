package env

import (
	"fmt"
	"os"
	"testing"
)

// TestLoad tests Load function.
func TestLoad(t *testing.T) {
	os.Clearenv()
	if err := os.Setenv("KEY_0", "default"); err != nil {
		t.Error(err)
	}

	// Load env-file.
	if err := Load("./fixtures/variables.env"); err != nil {
		t.Error(err)
	}

	// Variable update protection.
	if os.Getenv("KEY_0") != "default" {
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
	os.Clearenv()
	if err := Set("KEY_0", "default"); err != nil {
		t.Error(err)
	}

	// Load env-file.
	if err := LoadSafe("./fixtures/variables.env"); err != nil {
		t.Error(err)
	}

	// Expand test.
	// LoadSafe don't expand vars.
	if v := os.Getenv("KEY_2"); v != "${KEY_0}01" {
		t.Errorf("expected value `${KEY_0}01` but `%s`.", v)
	}
}

// TestUpdate tests Update function.
func TestUpdate(t *testing.T) {
	os.Clearenv()
	if err := Set("KEY_0", "default"); err != nil {
		t.Error(err)
	}

	// Load env-file.
	if err := Update("./fixtures/variables.env"); err != nil {
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
	// KEY_0 not overwritten.
	if v := Get("KEY_2"); v != "value_001" {
		t.Errorf("expected value `value_001` but `%s`.", v)
	}
}

// TestUpdateSafe tests UpdateSafe function.
func TestUpdateSafe(t *testing.T) {
	os.Clearenv()
	if err := Set("KEY_0", "default"); err != nil {
		t.Error(err)
	}

	// Load env-file.
	if err := UpdateSafe("./fixtures/variables.env"); err != nil {
		t.Error(err)
	}

	// Expand test.
	// UpdateSafe don't expand vars.
	if v := Get("KEY_2"); v != "${KEY_0}01" {
		t.Errorf("expected value `${KEY_0}01` but `%s`.", v)
	}
}

// TestExists tests Exists function.
func TestExist(t *testing.T) {
	var tests = [][]string{
		{"KEY_0", "default"},
		{"KEY_1", "default"},
	}

	os.Clearenv()
	for _, item := range tests {
		if err := os.Setenv(item[0], item[1]); err != nil {
			t.Error(err)
		}
	}

	// Variables is exists.
	if !Exists("KEY_0") || !Exists("KEY_0", "KEY_1") {
		t.Error("expected value `true` but `false`")
	}

	// Variables doesn't exists.
	if Exists("KEY_2") || Exists("KEY_0", "KEY_1", "KEY_2") {
		t.Error("expected value `false` but `true`")
	}
}

// TestSave tests Save function.
func TestSave(t *testing.T) {
	var data = struct {
		Host string `env:"HOST"`
		Port int    `env:"PORT"`
	}{
		Host: "localhost",
		Port: 8080,
	}

	// Save object.
	os.Clearenv()
	Save("/tmp/.env", "", data)

	// Not chanage environment.
	if h, p := os.Getenv("HOST"), os.Getenv("PORT"); h != "" || p != "" {
		t.Error("doesn't have to change the environment")
	}

	// Load object.
	if err := Load("/tmp/.env"); err != nil {
		t.Error(err)
	}

	h, p := os.Getenv("HOST"), os.Getenv("PORT")
	if h != data.Host {
		t.Errorf("expected `%s` but `%s`", data.Host, h)
	}

	if p != fmt.Sprint(data.Port) {
		t.Errorf("expected `%d` but `%s`", data.Port, fmt.Sprint(data.Port))
	}
}
