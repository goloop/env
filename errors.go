package env

import "errors"

// Sentinel errors returned by the package. Test for them with errors.Is.
var (
	// ErrNilObject is returned when the object passed to Unmarshal or
	// Marshal is nil.
	ErrNilObject = errors.New("env: object is nil")

	// ErrNotPointer is returned when the object is not a non-nil pointer
	// to a struct.
	ErrNotPointer = errors.New("env: object must be a non-nil pointer to a struct")

	// ErrNotStruct is returned when the object does not point to a struct.
	ErrNotStruct = errors.New("env: object must be a pointer to a struct")

	// ErrEmptyStruct is returned when the struct has no fields.
	ErrEmptyStruct = errors.New("env: struct has no fields")

	// ErrInvalidObject is returned by Marshal when the object is not a
	// struct or a pointer to a struct.
	ErrInvalidObject = errors.New("env: object must be a struct or a pointer to a struct")
)
