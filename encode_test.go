package env

import (
	"net/url"
	"strings"
	"testing"
)

// CustomMarshal structure with custom MarshalENV method.
type CustomMarshal struct {
	Host         string   `env:"HOST"`
	Port         int      `env:"PORT"`
	AllowedHosts []string `env:"ALLOWED_HOSTS,:"`
}

// MarshalENV the custom method for marshalling.
func (c *CustomMarshal) MarshalENV() ([]string, error) {
	var tests = [][]string{
		{"HOST", "192.168.0.1"},
		{"PORT", "80"},
		{"ALLOWED_HOSTS", "localhost"},
	}

	for _, item := range tests {
		err := Set(item[0], item[1])
		if err != nil {
			return []string{}, err
		}
	}

	return []string{
		"HOST=192.168.0.1",
		"PORT=80",
		"ALLOWED_HOSTS=localhost",
	}, nil
}

// TestMarshalENVNilPointer tests marshalENV function
// for uninitialized pointer.
func TestMarshalENVNilPointer(t *testing.T) {
	type Empty struct{}
	var value *Empty
	if _, err := marshalENV(value, ""); err == nil {
		t.Error("exception expected for an uninitialized object")
	}
}

// TestMarshalENVNotStruct tests marshalENV function for not struct.
func TestMarshalNotStruct(t *testing.T) {
	var value string
	if _, err := marshalENV(value, ""); err == nil {
		t.Error("exception expected for an object other than structure")
	}
}

// TestMarshalENV tests marshalENV function with struct value.
func TestMarshalENV(t *testing.T) {
	type data struct {
		Host         string    `env:"HOST"`
		Port         int       `env:"PORT"`
		AllowedHosts []string  `env:"ALLOWED_HOSTS,,!"`
		AllowedUsers [2]string `env:"ALLOWED_USERS,,:"`
	}

	var value = data{
		"192.168.0.5",
		8080,
		[]string{"localhost", "127.0.0.1"},
		[2]string{"John", "Bob"},
	}

	Clear()
	_, err := marshalENV(value, "")
	if err != nil {
		t.Error(err)
	}

	// Test marshalling.
	if v := Get("HOST"); v != "192.168.0.5" {
		t.Errorf("Incorrect value set for HOST: %s", v)
	}

	if v := Get("PORT"); v != "8080" {
		t.Errorf("Incorrect value set for PORT: %s", v)
	}

	if v := Get("ALLOWED_HOSTS"); v != "localhost!127.0.0.1" {
		t.Errorf("Incorrect value set for ALLOWED_HOSTS: %s", v)
	}

	if v := Get("ALLOWED_USERS"); v != "John:Bob" {
		t.Errorf("Incorrect value set for ALLOWED_USERS: %s", v)
	}
}

// TestMarshalENVPtr tests marshalENV function for pointer of the struct value.
func TestMarshalENVPtr(t *testing.T) {
	type Struct struct {
		Host         string    `env:"HOST"`
		Port         int       `env:"PORT"`
		AllowedHosts []string  `env:"ALLOWED_HOSTS,,!"`
		AllowedUsers [2]string `env:"ALLOWED_USERS,,:"`
	}
	var value = &Struct{
		"192.168.0.5",
		8080,
		[]string{"localhost", "127.0.0.1"},
		[2]string{"John", "Bob"},
	}

	Clear()
	_, err := marshalENV(value, "")
	if err != nil {
		t.Error(err)
	}

	// Test marshalling.
	if v := Get("HOST"); v != "192.168.0.5" {
		t.Errorf("Incorrect value set for HOST: %s", v)
	}

	if v := Get("PORT"); v != "8080" {
		t.Errorf("Incorrect value set for PORT: %s", v)
	}

	if v := Get("ALLOWED_HOSTS"); v != "localhost!127.0.0.1" {
		t.Errorf("Incorrect value set for ALLOWED_HOSTS: %s", v)
	}

	if v := Get("ALLOWED_USERS"); v != "John:Bob" {
		t.Errorf("Incorrect value set for ALLOWED_USERS: %s", v)
	}
}

// TestMarshalENVCustom tests marshalENV function for object
// with custom MarshalENV method.
func TestMarshalENVCustom(t *testing.T) {
	var scope = CustomMarshal{
		"localhost",                        // default: 192.168.0.1
		8080,                               // default: 80
		[]string{"localhost", "127.0.0.1"}, // default: 192.168.0.1
	}

	Clear()
	_, err := marshalENV(scope, "")
	if err != nil {
		t.Error(err)
	}

	// Test marshalling.
	if v := Get("HOST"); v != "192.168.0.1" {
		t.Errorf("Incorrect value set for HOST: %s", v)
	}

	if v := Get("PORT"); v != "80" {
		t.Errorf("Incorrect value set for PORT: %s", v)
	}

	if v := Get("ALLOWED_HOSTS"); v != "localhost" {
		t.Errorf("Incorrect value set for ALLOWED_HOSTS: %s", v)
	}
}

// TestMarshalENVCustomPtr tests marshalENV function for pointer
// with custom MarshalENV method.
func TestMarshalENVCustomPtr(t *testing.T) {
	var scope = &CustomMarshal{
		"localhost",                        // default: 192.168.0.1
		8080,                               // default: 80
		[]string{"localhost", "127.0.0.1"}, // default: 192.168.0.1
	}

	Clear()
	_, err := marshalENV(scope, "")
	if err != nil {
		t.Error(err)
	}

	// Test marshalling.
	if v := Get("HOST"); v != "192.168.0.1" {
		t.Errorf("Incorrect value set for HOST: %s", v)
	}

	if v := Get("PORT"); v != "80" {
		t.Errorf("Incorrect value set for PORT: %s", v)
	}

	if v := Get("ALLOWED_HOSTS"); v != "localhost" {
		t.Errorf("Incorrect value set for ALLOWED_HOSTS: %s", v)
	}
}

// TestMarshalENVURL tests marshaling of the URL.
func TestMarshalENVURL(t *testing.T) {
	type URLTestType struct {
		KeyURLPlain      url.URL     `env:"KEY_URL_PLAIN"`
		KeyURLPoint      *url.URL    `env:"KEY_URL_POINT"`
		KeyURLPlainSlice []url.URL   `env:"KEY_URL_PLAIN_SLICE,,!"`
		KeyURLPointSlice []*url.URL  `env:"KEY_URL_POINT_SLICE,,!"`
		KeyURLPlainArray [2]url.URL  `env:"KEY_URL_PLAIN_ARRAY,,!"`
		KeyURLPointArray [2]*url.URL `env:"KEY_URL_POINT_ARRAY,,!"`
	}

	var test string
	var data = URLTestType{
		KeyURLPlain: url.URL{Scheme: "http", Host: "plain.goloop.one"},
		KeyURLPoint: &url.URL{Scheme: "http", Host: "point.goloop.one"},
		KeyURLPlainSlice: []url.URL{
			{Scheme: "http", Host: "a.plain.goloop.one"},
			{Scheme: "http", Host: "b.plain.goloop.one"},
		},
		KeyURLPointSlice: []*url.URL{
			{Scheme: "http", Host: "a.point.goloop.one"},
			{Scheme: "http", Host: "b.point.goloop.one"},
		},
		KeyURLPlainArray: [2]url.URL{
			{Scheme: "http", Host: "c.plain.goloop.one"},
			{Scheme: "http", Host: "d.plain.goloop.one"},
		},
		KeyURLPointArray: [2]*url.URL{
			{Scheme: "http", Host: "c.point.goloop.one"},
			{Scheme: "http", Host: "d.point.goloop.one"},
		},
	}

	_, err := marshalENV(data, "")
	if err != nil {
		t.Error(err)
	}

	// Tests results.
	if v := Get("KEY_URL_PLAIN"); v != "http://plain.goloop.one" {
		t.Errorf("Incorrect marshaling plain url.URL: %s", v)
	}

	if v := Get("KEY_URL_POINT"); v != "http://point.goloop.one" {
		t.Errorf("Incorrect marshaling poin url.URL: %s", v)
	}

	// Plain slice.
	test = "http://a.plain.goloop.one!http://b.plain.goloop.one"
	if v := Get("KEY_URL_PLAIN_SLICE"); v != test {
		t.Errorf("Incorrect marshaling poin slice []url.URL: %s", v)
	}

	// Point slice.
	test = "http://a.point.goloop.one!http://b.point.goloop.one"
	if v := Get("KEY_URL_POINT_SLICE"); v != test {
		t.Errorf("Incorrect marshaling point slice []*url.URL: %s", v)
	}

	// Plain array.
	test = "http://c.plain.goloop.one!http://d.plain.goloop.one"
	if v := Get("KEY_URL_PLAIN_ARRAY"); v != test {
		t.Errorf("Incorrect marshaling plain array []url.URL: %s", v)
	}

	// Point array.
	test = "http://c.point.goloop.one!http://d.point.goloop.one"
	if v := Get("KEY_URL_POINT_ARRAY"); v != test {
		t.Errorf("Incorrect marshaling point array []*url.URL: %s", v)
	}
}

// TestMarshalENVStruct tests marshaling of the struct.
func TestMarshalENVStruct(t *testing.T) {
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

	var data = Client{
		User: User{
			Name: "John",
			Address: Address{
				Country: "USA",
			},
		},
		HomePage: url.URL{Scheme: "http", Host: "goloop.one"},
	}

	// Marshaling.
	result, _ := marshalENV(data, "")

	// Tests.
	if v := Get("USER_NAME"); v != "John" {
		t.Errorf("Incorrect marshaling (Name): %s\n%v", v, result)
	}

	if v := Get("USER_ADDRESS_COUNTRY"); v != "USA" {
		t.Errorf("Incorrect marshaling (Cuontry): %s\n%v", v, result)
	}

	if v := Get("HOME_PAGE"); v != "http://goloop.one" {
		t.Errorf("Incorrect marshaling url.URL (HomePage):%s", v)
	}
}

// TestMarshalENVStructPtr tests marshaling of the pointer on the struct.
func TestMarshalENVStructPtr(t *testing.T) {
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

	var data = Client{
		User: &User{
			Name: "John",
			Address: &Address{
				Country: "USA",
			},
		},
		HomePage: &url.URL{Scheme: "http", Host: "goloop.one"},
	}

	// Marshaling.
	result, _ := marshalENV(data, "")

	// Tests.
	if v := Get("USER_NAME"); v != "John" {
		t.Errorf("Incorrect marshaling (Name): %s\n%v", v, result)
	}

	if v := Get("USER_ADDRESS_COUNTRY"); v != "USA" {
		t.Errorf("Incorrect marshaling (Cuontry): %s\n%v", v, result)
	}

	if v := Get("HOME_PAGE"); v != "http://goloop.one" {
		t.Errorf("Incorrect marshaling url.URL (HomePage):%s", v)
	}
}

// TestMarshalENVNumberPtr tests marshalENV for pointer
// of Int, Uint and Float types.
func TestMarshalENVNumberPtr(t *testing.T) {
	type Struct struct {
		KeyInt     *int     `env:"KEY_INT"`
		KeyInt8    *int8    `env:"KEY_INT8"`
		KeyInt16   *int16   `env:"KEY_INT16"`
		KeyInt32   *int32   `env:"KEY_INT32"`
		KeyInt64   *int64   `env:"KEY_INT64"`
		KeyUint    *uint    `env:"KEY_UINT"`
		KeyUint8   *uint8   `env:"KEY_UINT8"`
		KeyUint16  *uint16  `env:"KEY_UINT16"`
		KeyUint32  *uint32  `env:"KEY_UINT32"`
		KeyUint64  *uint64  `env:"KEY_UINT64"`
		KeyFloat32 *float32 `env:"KEY_FLOAT32"`
		KeyFloat64 *float64 `env:"KEY_FLOAT64"`
	}

	var (
		keyInt     int     = 7
		keyInt8    int8    = 7
		keyInt16   int16   = 7
		keyInt32   int32   = 7
		keyInt64   int64   = 7
		keyUint    uint    = 7
		keyUint8   uint8   = 7
		keyUint16  uint16  = 7
		keyUint32  uint32  = 7
		keyUint64  uint64  = 7
		keyFloat32 float32 = 7.0
		keyFloat64 float64 = 7.0

		value = Struct{
			KeyInt:     &keyInt,
			KeyInt8:    &keyInt8,
			KeyInt16:   &keyInt16,
			KeyInt32:   &keyInt32,
			KeyInt64:   &keyInt64,
			KeyUint:    &keyUint,
			KeyUint8:   &keyUint8,
			KeyUint16:  &keyUint16,
			KeyUint32:  &keyUint32,
			KeyUint64:  &keyUint64,
			KeyFloat32: &keyFloat32,
			KeyFloat64: &keyFloat64,
		}
	)

	// ...
	Clear()
	_, err := marshalENV(value, "")
	if err != nil {
		t.Error(err)
	}

	// Tests.
	keys := []string{"KEY_INT", "KEY_INT8", "KEY_INT16", "KEY_INT32",
		"KEY_INT64", "KEY_UINT", "KEY_UINT8", "KEY_UINT16",
		"KEY_UINT32", "KEY_UINT64", "KEY_FLOAT32", "KEY_FLOAT64"}

	for _, key := range keys {
		tmp := strings.Split(Get(key), ".") // recline fractional part
		if tmp[0] != "7" {
			t.Errorf("Incorrect value set for %s: %s", key, tmp[0])
		}
	}
}

// TestMarshalENVBoolPtr tests marshalENV for pointer of bool.
func TestMarshalENVBoolPtr(t *testing.T) {
	type Struct struct {
		KeyBool *bool `env:"KEY_BOOL"`
	}

	var (
		keyBool bool = true

		value = Struct{
			KeyBool: &keyBool,
		}
	)

	// ...
	Clear()
	_, err := marshalENV(value, "")
	if err != nil {
		t.Error(err)
	}

	// Tests.
	if v := Get("KEY_BOOL"); v != "true" {
		t.Errorf("Incorrect value set for KEY_BOOL: %s", v)
	}
}

// TestMarshalENVStringPtr tests marshalENV for pointer of bool.
func TestMarshalENVStringPtr(t *testing.T) {
	type Struct struct {
		KeyString *string `env:"KEY_STRING"`
	}

	var (
		keyString string = "Hello World"

		value = Struct{
			KeyString: &keyString,
		}
	)

	// ...
	Clear()
	_, err := marshalENV(value, "")
	if err != nil {
		t.Error(err)
	}

	// Tests.
	if v := Get("KEY_STRING"); v != "Hello World" {
		t.Errorf("Incorrect value set for KEY_STRING: %s", v)
	}
}

// TestMarshalENVSlice tests marshalENV function for slice.
func TestMarshalENVSlice(t *testing.T) {
	type Slice struct {
		KeyInt   []*int   `env:"KEY_INT,:"`
		KeyInt8  []*int8  `env:"KEY_INT8,:"`
		KeyInt16 []*int16 `env:"KEY_INT16,:"`
		KeyInt32 []*int32 `env:"KEY_INT32,:"`
		KeyInt64 []*int64 `env:"KEY_INT64,:"`

		KeyUint   []*uint   `env:"KEY_UINT,:"`
		KeyUint8  []*uint8  `env:"KEY_UINT8,:"`
		KeyUint16 []*uint16 `env:"KEY_UINT16,:"`
		KeyUint32 []*uint32 `env:"KEY_UINT32,:"`
		KeyUint64 []*uint64 `env:"KEY_UINT64,:"`

		KeyFloat32 []*float32 `env:"KEY_FLOAT32,:"`
		KeyFloat64 []*float64 `env:"KEY_FLOAT64,:"`

		KeyString []*string `env:"KEY_STRING,:"`
		KeyBool   []*bool   `env:"KEY_BOOL,:"`
	}

	var (
		a, b, c int = 1, 2, 3

		keyInt = []*int{&a, &b, &c}
		s      = Slice{KeyInt: keyInt}
	)

	// // Convert slice into string.
	// toStr := func(v interface{}) string {
	// 	return strings.Trim(strings.Replace(fmt.Sprint(v), " ", ":", -1), "[]")
	// }

	_, err := marshalENV(s, "")
	if err != nil {
		t.Error(err)
	}
}
