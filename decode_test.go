package env

import (
	"fmt"
	"net/url"
	"strings"
	"testing"
)

// The dataUnmarshalENV structure with custom UnmarshalENV method.
type dataUnmarshalENV struct {
	Host         string   `env:"HOST"`
	Port         int      `env:"PORT"`
	AllowedHosts []string `env:"ALLOWED_HOSTS,:"`
}

// UnmarshalENV the custom method for unmarshalling data from the environment.
func (c *dataUnmarshalENV) UnmarshalENV() error {
	// You can use different methods to get data from the environment
	// like os.Getenv or env.Get and process the result according
	// to custom requirements.
	c.Host = "192.168.0.3"
	c.Port = 80
	c.AllowedHosts = []string{"192.168.0.1", "localhost"}
	return nil
}

// TestUnmarshalENVNotPointer tests unmarshalENV for the correct handling
// of an exception for a non-pointer value.
func TestUnmarshalENVNotPointer(t *testing.T) {
	type data struct{}
	if err := unmarshalENV("", data{}); err == nil {
		t.Error("an error is expected for non-pointer value")
	}
}

// TestUnmarshalENVNotInitialized tests unmarshalENV for the correct handling
// of an exception for a not initialized value.
func TestUnmarshalENVNotInitialized(t *testing.T) {
	type data struct{}
	var d *data
	if err := unmarshalENV("", d); err == nil {
		t.Error("an error is expected for not initialized value")
	}
}

// TestUnmarshalENVNotStruct tests unmarshalENV for the correct handling
// of an exception for a value that isn't a struct.
func TestUnmarshalENVNotStruct(t *testing.T) {
	var d = new(int)
	if err := unmarshalENV("", d); err == nil {
		t.Error("an error is expected for a pointer not to a struct")
	}
}

// TestUnmarshalENVCustom tests unmarshalENV function
// with custom UnmarshalENV method.
func TestUnmarshalENVCustom(t *testing.T) {
	var (
		c     = &dataUnmarshalENV{}
		err   error
		tests = [][]string{
			{"HOST", "0.0.0.1"},
			{"PORT", "8080"},
			{"ALLOWED_HOSTS", "localhost:127.0.0.1"},
		}
	)

	// Set test data.
	Clear()
	for _, item := range tests {
		err = Set(item[0], item[1])
		if err != nil {
			t.Error(err)
		}
	}

	err = unmarshalENV("", c)
	if err != nil {
		t.Error(err)
	}

	// Test marshalling.
	if c.Host != "192.168.0.3" {
		t.Errorf("HOST: expected `192.168.0.3` but `%v`", c.Host)
	}

	if c.Port != 80 {
		t.Errorf("PORT: expected `80` but `%v`", c.Port)
	}

	if value := sts(c.AllowedHosts, ":"); value != "192.168.0.1:localhost" {
		t.Errorf("ALLOWED_HOSTS: expected `%v` but `%v`",
			"192.168.0.1:localhost", value)
	}
}

// TestUnmarshalENVNumbers tests unmarshalENV for Int, Uint and Float types.
func TestUnmarshalENVNumbers(t *testing.T) {
	type data struct {
		KeyInt     int     `env:"KEY_INT"`
		KeyInt8    int8    `env:"KEY_INT8"`
		KeyInt16   int16   `env:"KEY_INT16"`
		KeyInt32   int32   `env:"KEY_INT32"`
		KeyInt64   int64   `env:"KEY_INT64"`
		KeyUint    uint    `env:"KEY_UINT"`
		KeyUint8   uint8   `env:"KEY_UINT8"`
		KeyUint16  uint16  `env:"KEY_UINT16"`
		KeyUint32  uint32  `env:"KEY_UINT32"`
		KeyUint64  uint64  `env:"KEY_UINT64"`
		KeyFloat32 float32 `env:"KEY_FLOAT32"`
		KeyFloat64 float64 `env:"KEY_FLOAT64"`
	}

	var (
		overflow = "9999999999999999999999999999999999999999999999999999999999"
		tests    = map[string][]string{
			"KEY_INT":     {"2", "-2", overflow},
			"KEY_INT8":    {"8", "-8", overflow},
			"KEY_INT16":   {"16", "-16", overflow},
			"KEY_INT32":   {"32", "-32", overflow},
			"KEY_INT64":   {"64", "-64", overflow},
			"KEY_UINT":    {"2", "-2", overflow},
			"KEY_UINT8":   {"8", "-8", overflow},
			"KEY_UINT16":  {"16", "-16", overflow},
			"KEY_UINT32":  {"32", "-32", overflow},
			"KEY_UINT64":  {"64", "-64", overflow},
			"KEY_FLOAT32": {"32.3", "-32.5", overflow},
			"KEY_FLOAT64": {"64.3", "-64.5", overflow},
		}
	)

	// Testing.
	for i := 0; i < 3; i++ {
		var err error
		for key, value := range tests {
			var d = &data{}

			// Set test data.
			Clear()
			err = Set(key, value[i])
			if err != nil {
				t.Error(err)
			}

			// Unmarshaling.
			err = unmarshalENV("", d)

			// Check error of the unmarshalling.
			switch i {
			case 0: // the value is correct for all types
				// Should not cause an error.
				if err != nil {
					t.Error(err)
					continue
				}
			case 1: // value is not valid for some types
				if !strings.Contains(key, "UINT") {
					// For int and float types should not cause an error.
					if err != nil {
						t.Error(err)
						continue
					}
				} else {
					// Uint cannot be negative.
					if err == nil {
						t.Errorf("uint cannot be negative: %s", value[i])
					}
					continue
				}
			case 2:
				// Ignore Float64 to check for `value out of range`.
				if !strings.Contains(key, "FLOAT64") && err == nil {
					t.Errorf("for %s must be `value out of "+
						"range` exception", key)
				}
				continue
			}

			// Check the correctness of the result.
			if v := fts(d, key, ""); v != value[i] {
				t.Errorf("%s is incorrect `%s`!=`%s`", key, v, value[i])
			}
		}
	}
}

// TestUnmarshalENVBoll tests unmarshalENV function for bool types.
func TestUnmarshalENVBool(t *testing.T) {
	type data struct {
		KeyBool bool `env:"KEY_BOOL"`
	}

	var (
		correct = map[string]bool{
			"true":  true,
			"false": false,
			"0":     false,
			"1":     true,
			"":      false,
			"True":  true,
			"TRUE":  true,
			"False": false,
			"FALSE": false,
		}
		incorrect = []string{
			"ok",
			"yes",
			"no",
			"0xff",
			"true/false",
		}
	)

	// Test correct values.
	for value, test := range correct {
		var (
			d   = &data{}
			err error
		)

		Clear()
		err = Set("KEY_BOOL", value)
		if err != nil {
			t.Error(err)
		}

		err = unmarshalENV("", d)
		if err != nil {
			t.Error(err)
		}

		if d.KeyBool != test {
			t.Errorf("KeyBool is incorrect `%t`!=`%t`", d.KeyBool, test)
		}
	}

	// Test incorrect values.
	for _, test := range incorrect {
		var (
			d   = &data{}
			err error
		)

		Clear()
		err = Set("KEY_BOOL", test)
		if err != nil {
			t.Error(err)
		}

		err = unmarshalENV("", d)
		if err == nil {
			t.Error("didn't handle the error")
		}
	}
}

// TestUnmarshalENVString tests unmarshalENV function for string type.
func TestUnmarshalENVString(t *testing.T) {
	type data struct {
		KeyString string `env:"KEY_STRING"`
	}

	var tests = []interface{}{
		8080,
		"Hello World",
		"true",
		true,
		3.14,
	}

	// Test correct values.
	for _, test := range tests {
		var (
			d   = &data{}
			s   = fmt.Sprintf("%v", test)
			err error
		)

		Clear()
		err = Set("KEY_STRING", s)
		if err != nil {
			t.Error(err)
		}

		err = unmarshalENV("", d)
		if err != nil {
			t.Error(err)
		}

		if d.KeyString != s {
			t.Errorf("KeyString is incorrect `%s`!=`%s`", d.KeyString, s)
		}
	}
}

// TestUnmarshalENVSlice tests unmarshalENV function for slice.
func TestUnmarshalENVSlice(t *testing.T) {
	// Use `#` as separator for items.
	type data struct {
		KeyInt   []int   `env:"KEY_INT,,#"`
		KeyInt8  []int8  `env:"KEY_INT8,,#"`
		KeyInt16 []int16 `env:"KEY_INT16,,#"`
		KeyInt32 []int32 `env:"KEY_INT32,,#"`
		KeyInt64 []int64 `env:"KEY_INT64,,#"`

		KeyUint   []uint   `env:"KEY_UINT,,#"`
		KeyUint8  []uint8  `env:"KEY_UINT8,,#"`
		KeyUint16 []uint16 `env:"KEY_UINT16,,#"`
		KeyUint32 []uint32 `env:"KEY_UINT32,,#"`
		KeyUint64 []uint64 `env:"KEY_UINT64,,#"`

		KeyFloat32 []float32 `env:"KEY_FLOAT32,,#"`
		KeyFloat64 []float64 `env:"KEY_FLOAT64,,#"`

		KeyString []string `env:"KEY_STRING,,#"`
		KeyGroup  []string `env:"KEY_GROUP,,#"`
		KeyBool   []bool   `env:"KEY_BOOL,,#"`
	}

	var (
		corretc = map[string]string{
			"KEY_INT":   "-30#-20#-10#0#10#20#30",
			"KEY_INT8":  "-30#-20#-10#0#10#20#30",
			"KEY_INT16": "-30#-20#-10#0#10#20#30",
			"KEY_INT32": "-30#-20#-10#0#10#20#30",
			"KEY_INT64": "-30#-20#-10#0#10#20#30",

			"KEY_UINT":   "0#10#20#30",
			"KEY_UINT8":  "0#10#20#30",
			"KEY_UINT16": "0#10#20#30",
			"KEY_UINT32": "0#10#20#30",
			"KEY_UINT64": "0#10#20#30",

			"KEY_FLOAT32": "-3.1#-1.27#0#1.27#3.3",
			"KEY_FLOAT64": "-3.1#-1.27#0#1.27#3.3",

			"KEY_STRING": "one#two#three#four#five",
			"KEY_GROUP":  "one#\"two#three\"#four#five",
			"KEY_BOOL":   "1#true#True#TRUE#0#false#False#False",
		}
		incorrect = map[string]string{
			"KEY_INT":   "-30#-20#-10#A#10#20#30",
			"KEY_INT8":  "-30#-20#-10#A#10#20#30",
			"KEY_INT16": "-30#-20#-10#A#10#20#30",
			"KEY_INT32": "-30#-20#-10#A#10#20#30",
			"KEY_INT64": "-30#-20#-10#A#10#20#30",

			"KEY_UINT":   "0#10#-20#30",
			"KEY_UINT8":  "0#10#-20#30",
			"KEY_UINT16": "0#10#-20#30",
			"KEY_UINT32": "0#10#-20#30",
			"KEY_UINT64": "0#10#-20#30",

			"KEY_FLOAT32": "-3.1#-1.27#A#1.27#3.3",
			"KEY_FLOAT64": "-3.1#-1.27#A#1.27#3.3",
		}
	)

	// Testing correct values.
	for key, value := range corretc {
		var (
			d   = &data{}
			err error
		)

		Clear()
		err = Set(key, value)
		if err != nil {
			t.Error(err)
		}

		err = unmarshalENV("", d)
		if err != nil {
			t.Error(err)
		}

		if key == "KEY_BOOL" {
			// Bool string.
			value = "true#true#true#true#false#false#false#false"
		} else if key == "KEY_GROUP" {
			// Checking if the group has been split correctly.
			value = "one#\"two:three\"#four#five"
			for i := 0; i < len(d.KeyGroup); i++ {
				d.KeyGroup[i] = strings.Replace(d.KeyGroup[i], "#", ":", 1)
			}
		}

		if v := fts(d, key, "#"); v != value {
			t.Errorf("%s is incorrect `%s` != `%s` %v", key, v, value, d)
		}
	}

	// Testing incorrect values.
	for key, value := range incorrect {
		var (
			d   = &data{}
			err error
		)

		Clear()
		err = Set(key, value)
		if err != nil {
			t.Error(err)
		}

		err = unmarshalENV("", d)
		if err == nil {
			t.Error("must be error for", value)
		}
	}
}

// TestUnmarshalENVArray tests unmarshalENV function with array.
func TestUnmarshalENVArray(t *testing.T) {
	// Use default separator for items (i.e. `:` symbol).
	type data struct {
		KeyInt     [5]int     `env:"KEY_INT"`
		KeyUint    [4]uint    `env:"KEY_UINT"`
		KeyFloat64 [5]float64 `env:"KEY_FLOAT64"`
		KeyString  [5]string  `env:"KEY_STRING"`
		KeyGroup   [4]string  `env:"KEY_GROUP"`
		KeyBool    [8]bool    `env:"KEY_BOOL"`
	}

	var (
		corretc = map[string]string{
			"KEY_INT":     "-20:-10:0:10:20",
			"KEY_UINT":    "0:10:20:30",
			"KEY_FLOAT64": "-3.1:-1.27:0:1.27:3.3",
			"KEY_STRING":  "one:two:three:four:five",
			"KEY_GROUP":   "one:\"two:three\":four:five",
			"KEY_BOOL":    "1:true:True:TRUE:0:false:False:False",
		}
		incorrect = map[string]string{
			"KEY_INT":     "-30:-20:-10:A:10:20:30",
			"KEY_UINT":    "0:10:-20:30",
			"KEY_FLOAT64": "-3.1:-1.27:A:1.27:3.3",
		}
		overflow = map[string]string{
			"KEY_INT":     "-30:-20:-10:0:10:20:30:100",
			"KEY_UINT":    "0:10:20:30:100",
			"KEY_FLOAT64": "-3.1:-1.27:0:1.27:3.3:100.0",
			"KEY_STRING":  "one:two:three:four:five:one hundred",
			"KEY_BOOL":    "1:true:True:TRUE:0:false:False:False:true",
		}
	)

	// Test correct values.
	for key, value := range corretc {
		var (
			d   = &data{}
			err error
		)

		Clear()
		err = Set(key, value)
		if err != nil {
			t.Error(err)
		}

		err = unmarshalENV("", d)
		if err != nil {
			t.Error(err)
		}

		if key == "KEY_BOOL" {
			value = "true:true:true:true:false:false:false:false"
		} else if key == "KEY_GROUP" {
			// Checking if the group has been split correctly.
			value = "one:\"two#three\":four:five"
			for i := 0; i < len(d.KeyGroup); i++ {
				d.KeyGroup[i] = strings.Replace(d.KeyGroup[i], ":", "#", 1)
			}
		}

		if v := fts(d, key, ":"); v != value {
			t.Errorf("%s is incorrect `%s` != `%s` %v", key, v, value, d)
		}
	}

	// Test incorrect values.
	for key, value := range incorrect {
		var (
			d   = &data{}
			err error
		)

		Clear()
		err = Set(key, value)
		if err != nil {
			t.Error(err)
		}

		err = unmarshalENV("", d)
		if err == nil {
			t.Error("There should be an exception due to an invalid value.")
		}
	}

	// Test array overflow.
	for key, value := range overflow {
		var (
			d   = &data{}
			err error
		)

		Clear()
		err = Set(key, value)
		if err != nil {
			t.Error(err)
		}

		err = unmarshalENV("", d)
		if err == nil {
			t.Error("There should be an exception due to array overflow.")
		}
	}
}

// TestUnmarshalURL tests unmarshalENV for url.URL type.
func TestUnmarshalURL(t *testing.T) {
	type data struct {
		KeyURLPlain      url.URL     `env:"KEY_URL_PLAIN"`
		KeyURLPoint      *url.URL    `env:"KEY_URL_POINT"`
		KeyURLPlainSlice []url.URL   `env:"KEY_URL_PLAIN_SLICE,,!"`
		KeyURLPointSlice []*url.URL  `env:"KEY_URL_POINT_SLICE,,!"`
		KeyURLPlainArray [2]url.URL  `env:"KEY_URL_PLAIN_ARRAY,,!"`
		KeyURLPointArray [2]*url.URL `env:"KEY_URL_POINT_ARRAY,,!"`
	}

	var (
		slice []string
		str   string
		err   error
		d     = data{}

		defaults = [][]string{
			{
				"KEY_URL_PLAIN",
				"http://plain.goloop.one",
			},
			{
				"KEY_URL_POINT",
				"http://point.goloop.one",
			},
			{
				"KEY_URL_PLAIN_SLICE",
				"http://a.plain.goloop.one!http://b.plain.goloop.one",
			},
			{
				"KEY_URL_POINT_SLICE",
				"http://a.point.goloop.one!http://b.point.goloop.one",
			},
			{
				"KEY_URL_PLAIN_ARRAY",
				"http://c.plain.goloop.one!http://d.plain.goloop.one",
			},
			{
				"KEY_URL_POINT_ARRAY",
				"http://c.point.goloop.one!http://d.point.goloop.one",
			},
		}
	)

	// Set tests data.
	for _, item := range defaults {
		err = Set(item[0], item[1])
		if err != nil {
			t.Error(err)
		}
	}

	// Unmarshaling.
	err = unmarshalENV("", &d)
	if err != nil {
		t.Error(err)
	}

	// Tests results.
	if v := d.KeyURLPlain.String(); v != "http://plain.goloop.one" {
		t.Errorf("Incorrect unmarshaling plain url.URL: %s", v)
	}

	if v := d.KeyURLPoint.String(); v != "http://point.goloop.one" {
		t.Errorf("Incorrect unmarshaling point url.URL: %s", v)
	}

	// Plain slice.
	slice = []string{}
	for _, v := range d.KeyURLPlainSlice {
		slice = append(slice, v.String())
	}
	str = strings.Trim(strings.Replace(fmt.Sprint(slice), " ", "!", -1), "[]")
	if str != "http://a.plain.goloop.one!http://b.plain.goloop.one" {
		t.Errorf("Incorrect unmarshaling plain slice []url.URL: %s", str)
	}

	// Point slice.
	slice = []string{}
	for _, v := range d.KeyURLPointSlice {
		slice = append(slice, v.String())
	}
	str = strings.Trim(strings.Replace(fmt.Sprint(slice), " ", "!", -1), "[]")
	if str != "http://a.point.goloop.one!http://b.point.goloop.one" {
		t.Errorf("Incorrect unmarshaling point alice []*url.URL: %s", str)
	}

	// Plain array.
	slice = []string{}
	for _, v := range d.KeyURLPlainArray {
		slice = append(slice, v.String())
	}

	str = strings.Trim(strings.Replace(fmt.Sprint(slice), " ", "!", -1), "[]")
	if str != "http://c.plain.goloop.one!http://d.plain.goloop.one" {
		t.Errorf("Incorrect unmarshaling plain array [2]url.URL: %s", str)
	}

	// Point array.
	slice = []string{}
	for _, v := range d.KeyURLPointArray {
		slice = append(slice, v.String())
	}

	str = strings.Trim(strings.Replace(fmt.Sprint(slice), " ", "!", -1), "[]")
	if str != "http://c.point.goloop.one!http://d.point.goloop.one" {
		t.Errorf("Incorrect unmarshaling point array [2]*url.URL: %s", str)
	}
}

// TestUnmarshalStruct tests unmarshalENV for the struct.
func TestUnmarshalStruct(t *testing.T) {
	type Address struct {
		Country string `env:"COUNTRY"`
	}

	type User struct {
		Name    string  `env:"NAME"`
		Address Address `env:"ADDRESS"`
	}

	type Client struct {
		User     User    `env:"USER"`
		HomePage url.URL `env:"HOME_PAGE"`
	}

	var (
		err   error
		c     = Client{}
		tests = [][]string{
			{"USER_NAME", "Jerry"},
			{"USER_ADDRESS_COUNTRY", "UK"},
			{"HOME_PAGE", "http://example.org"},
		}
	)

	for _, item := range tests {
		err = Set(item[0], item[1])
		if err != nil {
			t.Error(err)
		}
	}

	// Unmarshaling.
	err = unmarshalENV("", &c)
	if err != nil {
		t.Error("Incorrect ummarshaling.")
	}

	// Tests.
	if c.User.Address.Country != "UK" {
		t.Errorf("Incorrect ummarshaling User.Address: %v", c.User.Address)
	}

	if c.User.Name != "Jerry" {
		t.Errorf("Incorrect ummarshaling User: %v", c.User)
	}

	if c.HomePage.String() != "http://example.org" {
		t.Errorf("Incorrect ummarshaling url.URL: %v", c.HomePage)
	}
}

// TestUnmarshalStructPtr tests unmarshalENV for the pointerf of the struct.
func TestUnmarshalStructPtr(t *testing.T) {
	type Address struct {
		Country string `env:"COUNTRY"`
	}

	type User struct {
		Name    string   `env:"NAME"`
		Address *Address `env:"ADDRESS"`
	}

	type Client struct {
		User     *User    `env:"USER"`
		HomePage *url.URL `env:"HOME_PAGE"`
	}

	var (
		err   error
		c     = Client{}
		tests = [][]string{
			{"USER_NAME", "Lucy"},
			{"USER_ADDRESS_COUNTRY", "UA"},
			{"HOME_PAGE", "http://example.net"},
		}
	)

	for _, item := range tests {
		err = Set(item[0], item[1])
		if err != nil {
			t.Error(err)
		}
	}

	// Unmarshaling.
	err = unmarshalENV("", &c)
	if err != nil {
		t.Error("Incorrect ummarshaling.")
	}

	// Tests.
	if c.User.Address.Country != "UA" {
		t.Errorf("Incorrect ummarshaling User.Address: %v", c.User.Address)
	}

	if c.User.Name != "Lucy" {
		t.Errorf("Incorrect ummarshaling User: %v", c.User)
	}

	if c.HomePage.String() != "http://example.net" {
		t.Errorf("Incorrect ummarshaling url.URL: %v", c.HomePage)
	}
}

// TestUnmarshalENVStringPtr tests unmarshalENV function
// for pointer on the string type.
func TestUnmarshalENVStringPtr(t *testing.T) {
	type data struct {
		KeyString *string `env:"KEY_STRING"`
	}

	var (
		err error
		s   string
		d   = data{KeyString: &s}
	)

	err = Set("KEY_STRING", "Hello World")
	if err != nil {
		t.Error(err)
	}

	err = unmarshalENV("", &d)
	if err != nil {
		t.Error(err)
	}

	if *d.KeyString != "Hello World" {
		t.Errorf("Incorrect value set for KEY_STRING: %v", *d.KeyString)
	}

}

// TestUnmarshalDefaultValue tests unmarshalENV for default value.
func TestUnmarshalDefaultValue(t *testing.T) {
	type data struct {
		Host         string    `env:"HOST,0.0.0.0"`
		AllowedHosts []string  `env:"ALLOWED_HOSTS,{localhost:0.0.0.0},:"`
		Names        [3]string `env:"NAME_LIST,'John,Bob,Smit',,"` // sep `,`
	}

	var (
		d     data
		err   error
		tests = [][]string{
			{"HOST", "localhost"},
			{"ALLOWED_HOSTS", "127.0.0.1:localhost"},
			{"NAME_LIST", "John"},
		}
	)

	Clear() // make empty environment

	// Unmarshaling wit default values.
	d = data{}
	err = unmarshalENV("", &d)
	if err != nil {
		t.Error("Incorrect ummarshaling.")
	}

	if d.Host != "0.0.0.0" {
		t.Errorf("incorrect Host %s", d.Host)
	}

	if v := sts(d.AllowedHosts, ":"); v != "localhost:0.0.0.0" {
		t.Errorf("incorrect AllowedHosts %s", v)
	}

	if v := sts(d.Names, ":"); v != "John:Bob:Smit" {
		t.Errorf("incorrect Names %s", d)
	}

	// Set any values.
	for _, item := range tests {
		err = Set(item[0], item[1])
		if err != nil {
			t.Error(err)
		}
	}

	// Unmarshaling wit environment values.
	d = data{}
	err = unmarshalENV("", &d)
	if err != nil {
		t.Error("Incorrect ummarshaling.")
	}

	if d.Host == "0.0.0.0" {
		t.Errorf("Host sets as default %s", d.Host)
	}

	if sts(d.AllowedHosts, ":") == "localhost:0.0.0.0" {
		t.Errorf("AllowedHosts sets as default %s", d.AllowedHosts)
	}

	if sts(d.Names, ":") == "John:Bob:Smit" {
		t.Errorf("Names setas as default %s", d.Names)
	}
}
