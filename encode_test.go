package env

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
)

// The configEncode structure with custom MarshalEnv method.
type configEncode struct {
	Host         string   `env:"HOST"`
	Port         int      `env:"PORT"`
	AllowedHosts []string `env:"ALLOWED_HOSTS" sep:":"`
}

// MarshalEnv the custom method for marshalling.
func (c *configEncode) MarshalEnv() ([]string, error) {
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

// TestMarshalEnvNilPointer tests marshalEnv function
// for uninitialized pointer.
func TestMarshalEnvNilPointer(t *testing.T) {
	var data *struct{}
	if _, err := marshalEnv("", data); err == nil {
		t.Error("should be error for an uninitialized object")
	}
}

// TestMarshalEnvNotStruct tests marshalEnv function for not struct.
func TestMarshalNotStruct(t *testing.T) {
	var data string
	if _, err := marshalEnv(data, ""); err == nil {
		t.Error("should be error for an object other than structure")
	}
}

// TestMarshalEnv tests marshalEnv function with struct value.
func TestMarshalEnv(t *testing.T) {
	var data = struct {
		Host         string    `env:"HOST"`
		Port         int       `env:"PORT"`
		AllowedHosts []string  `env:"ALLOWED_HOSTS" sep:"!"`
		AllowedUsers [2]string `env:"ALLOWED_USERS" sep:":"`
	}{
		"192.168.0.5",
		8080,
		[]string{"localhost", "127.0.0.1"},
		[2]string{"John", "Bob"},
	}

	// ...
	os.Clearenv()
	if _, err := marshalEnv("", data); err != nil {
		t.Error(err)
	}

	// Test marshalling.
	expected := data.Host
	if v := Get("HOST"); v != expected {
		t.Errorf("expected %s but %s", expected, v)
	}

	expected = fmt.Sprintf("%d", data.Port)
	if v := Get("PORT"); v != expected {
		t.Errorf("expected %s but %s", expected, v)
	}

	expected = strings.Join(data.AllowedHosts, "!")
	if v := Get("ALLOWED_HOSTS"); v != expected {
		t.Errorf("expected %s but %s", expected, v)
	}

	expected = strings.Join(data.AllowedUsers[:], ":")
	if v := Get("ALLOWED_USERS"); v != expected {
		t.Errorf("expected %s but %s", expected, v)
	}
}

// TestMarshalEnvPtr tests marshalEnv function for pointer of the struct value.
func TestMarshalEnvPtr(t *testing.T) {
	var data = &struct {
		Host         string    `env:"HOST"`
		Port         int       `env:"PORT"`
		AllowedHosts []string  `env:"ALLOWED_HOSTS" sep:"!"`
		AllowedUsers [2]string `env:"ALLOWED_USERS" sep:":"`
	}{
		"192.168.0.5",
		8080,
		[]string{"localhost", "127.0.0.1"},
		[2]string{"John", "Bob"},
	}

	os.Clearenv()
	if _, err := marshalEnv("", data); err != nil {
		t.Error(err)
	}

	// Test marshalling.
	expected := data.Host
	if v := Get("HOST"); v != expected {
		t.Errorf("expected %s but %s", expected, v)
	}

	expected = fmt.Sprintf("%d", data.Port)
	if v := Get("PORT"); v != expected {
		t.Errorf("expected %s but %s", expected, v)
	}

	expected = strings.Join(data.AllowedHosts, "!")
	if v := Get("ALLOWED_HOSTS"); v != expected {
		t.Errorf("expected %s but %s", expected, v)
	}

	expected = strings.Join(data.AllowedUsers[:], ":")
	if v := Get("ALLOWED_USERS"); v != expected {
		t.Errorf("expected %s but %s", expected, v)
	}
}

// TestMarshalEnvCustom tests marshalEnv function for object
// with custom MarshalEnv method.
func TestMarshalEnvCustom(t *testing.T) {
	var data = configEncode{
		"localhost",                        // default: 192.168.0.1
		8080,                               // default: 80
		[]string{"localhost", "127.0.0.1"}, // default: 192.168.0.1
	}

	os.Clearenv()
	if _, err := marshalEnv("", data); err != nil {
		t.Error(err)
	}

	// Test marshalling.
	expected := "192.168.0.1"
	if v := Get("HOST"); v != expected {
		t.Errorf("expected %s but %s", expected, v)
	}

	expected = "80"
	if v := Get("PORT"); v != expected {
		t.Errorf("expected %s but %s", expected, v)
	}

	expected = "localhost"
	if v := Get("ALLOWED_HOSTS"); v != expected {
		t.Errorf("expected %s but %s", expected, v)
	}
}

// TestMarshalEnvCustomPtr tests marshalEnv function for pointer
// with custom MarshalEnv method.
func TestMarshalEnvCustomPtr(t *testing.T) {
	var scope = &configEncode{
		"localhost",                        // default: 192.168.0.1
		8080,                               // default: 80
		[]string{"localhost", "127.0.0.1"}, // default: 192.168.0.1
	}

	os.Clearenv()
	if _, err := marshalEnv("", scope); err != nil {
		t.Error(err)
	}

	// Test marshalling.
	expected := "192.168.0.1"
	if v := Get("HOST"); v != expected {
		t.Errorf("expected %s but %s", expected, v)
	}

	expected = "80"
	if v := Get("PORT"); v != expected {
		t.Errorf("expected %s but %s", expected, v)
	}

	expected = "localhost"
	if v := Get("ALLOWED_HOSTS"); v != expected {
		t.Errorf("expected %s but %s", expected, v)
	}
}

// TestMarshalEnvURL tests marshaling of the URL.
func TestMarshalEnvURL(t *testing.T) {
	type URLTestType struct {
		KeyURLPlain      url.URL     `env:"KEY_URL_PLAIN"`
		KeyURLPoint      *url.URL    `env:"KEY_URL_POINT"`
		KeyURLPlainSlice []url.URL   `env:"KEY_URL_PLAIN_SLICE" sep:"!"`
		KeyURLPointSlice []*url.URL  `env:"KEY_URL_POINT_SLICE" sep:"!"`
		KeyURLPlainArray [2]url.URL  `env:"KEY_URL_PLAIN_ARRAY" sep:"!"`
		KeyURLPointArray [2]*url.URL `env:"KEY_URL_POINT_ARRAY" sep:"!"`
	}

	//var test string
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

	os.Clearenv()
	if _, err := marshalEnv("", data); err != nil {
		t.Error(err)
	}

	// Tests results.
	expected := "http://plain.goloop.one"
	if v := Get("KEY_URL_PLAIN"); v != expected {
		t.Errorf("expected %s but %s", expected, v)
		//t.Errorf("Incorrect marshaling plain url.URL: %s", v)
	}

	expected = "http://point.goloop.one"
	if v := Get("KEY_URL_POINT"); v != expected {
		t.Errorf("expected %s but %s", expected, v)
	}

	// Plain slice.
	expected = "http://a.plain.goloop.one!http://b.plain.goloop.one"
	if v := Get("KEY_URL_PLAIN_SLICE"); v != expected {
		t.Errorf("expected %s but %s", expected, v)
		//t.Errorf("Incorrect marshaling poin slice []url.URL: %s", v)
	}

	// Point slice.
	expected = "http://a.point.goloop.one!http://b.point.goloop.one"
	if v := Get("KEY_URL_POINT_SLICE"); v != expected {
		t.Errorf("expected %s but %s", expected, v)
	}

	// Plain array.
	expected = "http://c.plain.goloop.one!http://d.plain.goloop.one"
	if v := Get("KEY_URL_PLAIN_ARRAY"); v != expected {
		t.Errorf("expected %s but %s", expected, v)
	}

	// Point array.
	expected = "http://c.point.goloop.one!http://d.point.goloop.one"
	if v := Get("KEY_URL_POINT_ARRAY"); v != expected {
		t.Errorf("expected %s but %s", expected, v)
	}
}

// TestMarshalEnvStruct tests marshaling of the struct.
func TestMarshalEnvStruct(t *testing.T) {
	type address struct {
		Country string `env:"COUNTRY"`
	}

	type user struct {
		Name    string  `env:"NAME"`
		Address address `env:"ADDRESS"`
	}

	type client struct {
		User     user    `env:"USER"`
		HomePage url.URL `env:"HOME_PAGE"`
	}

	var data = client{
		User: user{
			Name: "John",
			Address: address{
				Country: "USA",
			},
		},
		HomePage: url.URL{Scheme: "http", Host: "goloop.one"},
	}

	// Marshaling.
	os.Clearenv()
	_, err := marshalEnv("", data)
	if err != nil {
		t.Error(err)
	}

	// Tests.
	expected := data.User.Name
	if v := os.Getenv("USER_NAME"); v != expected {
		t.Errorf("expected %s but %s", expected, v)
	}

	expected = data.User.Address.Country
	if v := os.Getenv("USER_ADDRESS_COUNTRY"); v != expected {
		t.Errorf("expected %s but %s", expected, v)
	}

	expected = data.HomePage.String()
	if v := os.Getenv("HOME_PAGE"); v != expected {
		t.Errorf("expected %s but %s", expected, v)
	}
}

// TestMarshalEnvStructPtr tests marshaling of the pointer on the struct.
func TestMarshalEnvStructPtr(t *testing.T) {
	type address struct {
		Country string `env:"COUNTRY"`
	}

	type user struct {
		Name    string   `env:"NAME"`
		Address *address `env:"ADDRESS"`
	}

	type client struct {
		User     *user    `env:"USER"`
		HomePage *url.URL `env:"HOME_PAGE"`
	}

	var data = client{
		User: &user{
			Name: "John",
			Address: &address{
				Country: "USA",
			},
		},
		HomePage: &url.URL{Scheme: "http", Host: "goloop.one"},
	}

	// Marshaling.
	os.Clearenv()
	_, err := marshalEnv("", data)
	if err != nil {
		t.Error(err)
	}

	// Tests.
	expected := data.User.Name
	if v := os.Getenv("USER_NAME"); v != expected {
		t.Errorf("expected %s but %s", expected, v)
	}

	expected = data.User.Address.Country
	if v := os.Getenv("USER_ADDRESS_COUNTRY"); v != expected {
		t.Errorf("expected %s but %s", expected, v)
	}

	expected = data.HomePage.String()
	if v := os.Getenv("HOME_PAGE"); v != expected {
		t.Errorf("expected %s but %s", expected, v)
	}
}

// TestMarshalEnvNumberPtr tests marshalEnv for pointer
// of Int, Uint and Float types.
func TestMarshalEnvNumberPtr(t *testing.T) {
	var (
		keyInt     = int(7)
		keyInt8    = int8(7)
		keyInt16   = int16(7)
		keyInt32   = int32(7)
		keyInt64   = int64(7)
		keyUint    = uint(7)
		keyUint8   = uint8(7)
		keyUint16  = uint16(7)
		keyUint32  = uint32(7)
		keyUint64  = uint64(7)
		keyFloat32 = float32(7.0)
		keyFloat64 = float64(7.0)

		data = struct {
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
		}{
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
	os.Clearenv()
	if _, err := marshalEnv("", data); err != nil {
		t.Error(err)
	}

	// Tests.
	keys := []string{
		"KEY_INT", "KEY_INT8", "KEY_INT16", "KEY_INT32",
		"KEY_INT64", "KEY_UINT", "KEY_UINT8", "KEY_UINT16",
		"KEY_UINT32", "KEY_UINT64", "KEY_FLOAT32", "KEY_FLOAT64",
	}

	for _, key := range keys {
		tmp := strings.Split(os.Getenv(key), ".") // recline fractional part
		if tmp[0] != "7" {
			t.Errorf("incorrect value set for %s: %s", key, tmp[0])
		}
	}
}

// TestMarshalEnvBoolPtr tests marshalEnv for pointer of bool.
func TestMarshalEnvBoolPtr(t *testing.T) {
	var (
		keyBool = true
		data    = struct {
			KeyBool *bool `env:"KEY_BOOL"`
		}{
			KeyBool: &keyBool,
		}
	)

	// ...
	os.Clearenv()
	if _, err := marshalEnv("", data); err != nil {
		t.Error(err)
	}

	// Tests.
	if v := Get("KEY_BOOL"); v != "true" {
		t.Errorf("incorrect value set for KEY_BOOL: %s", v)
	}
}

// TestMarshalEnvStringPtr tests marshalEnv for pointer of bool.
func TestMarshalEnvStringPtr(t *testing.T) {
	var (
		keyString = "Hello World"
		value     = struct {
			KeyString *string `env:"KEY_STRING"`
		}{
			KeyString: &keyString,
		}
	)

	// ...
	os.Clearenv()
	if _, err := marshalEnv("", value); err != nil {
		t.Error(err)
	}

	// Tests.
	if v := os.Getenv("KEY_STRING"); v != "Hello World" {
		t.Errorf("incorrect value set for KEY_STRING: %s", v)
	}
}

// TestMarshalEnvSlice tests marshalEnv function for slice.
func TestMarshalEnvSlice(t *testing.T) {
	type chunk struct {
		KeyInt   []*int   `env:"KEY_INT" sep:":"`
		KeyInt8  []*int8  `env:"KEY_INT8" sep:":"`
		KeyInt16 []*int16 `env:"KEY_INT16" sep:":"`
		KeyInt32 []*int32 `env:"KEY_INT32" sep:":"`
		KeyInt64 []*int64 `env:"KEY_INT64" sep:":"`

		KeyUint   []*uint   `env:"KEY_UINT" sep:":"`
		KeyUint8  []*uint8  `env:"KEY_UINT8" sep:":"`
		KeyUint16 []*uint16 `env:"KEY_UINT16" sep:":"`
		KeyUint32 []*uint32 `env:"KEY_UINT32" sep:":"`
		KeyUint64 []*uint64 `env:"KEY_UINT64" sep:":"`

		KeyFloat32 []*float32 `env:"KEY_FLOAT32" sep:":"`
		KeyFloat64 []*float64 `env:"KEY_FLOAT64" sep:":"`

		KeyString []*string `env:"KEY_STRING" sep:":"`
		KeyBool   []*bool   `env:"KEY_BOOL" sep:":"`
	}

	var (
		a, b, c int = 1, 2, 3
		keyInt      = []*int{&a, &b, &c}
		s           = chunk{KeyInt: keyInt}
	)

	if _, err := marshalEnv("", s); err != nil {
		t.Error(err)
	}
}

// TestUnmarshalMultiService tests unmarshaling of the
// data of environment by the specified prefix.
func TestUnmarshalMultiService(t *testing.T) {
	type server struct {
		Name string `env:"NAME"`
		Host string `env:"HOST"`
		Port int    `env:"PORT"`
	}

	var (
		serverA = server{}
		serverB = server{}
	)

	os.Clearenv()
	err := readParseStore("./fixtures/multiservice.env", true, true, true)
	if err != nil {
		t.Error(err)
	}

	Unmarshal("SERVICE_A_", &serverA)
	Unmarshal("SERVICE_B_", &serverB)

	if v := serverA.Name; v != "A" {
		t.Errorf("expected `A` but `%s`", v)
	}

	if v := serverB.Name; v != "B" {
		t.Errorf("expected `B` but `%s`", v)
	}
}
