package env

import "strings"

// Option configures a marshal/unmarshal call. Options set call-level defaults
// that a per-field tag can override (a field's sep tag overrides WithSeparator).
type Option func(*settings)

// WithPrefix sets the key namespace for the call. The prefix names a level and
// levels are joined with "_": WithPrefix("APP") maps the field PORT to APP_PORT.
// A trailing "_" is added automatically when missing, and an empty prefix adds
// no leading "_".
func WithPrefix(prefix string) Option {
	return func(s *settings) { s.prefix = prefix }
}

// WithSeparator sets the default separator for slice/array values. A per-field
// sep tag still takes precedence. The built-in default is a single space.
func WithSeparator(sep string) Option {
	return func(s *settings) { s.separator = sep }
}

// newSettings builds the resolved settings for the public API from the given
// options, applying defaults and normalizing the prefix (appending "_" when
// it is non-empty and does not already end with it).
func newSettings(opts ...Option) settings {
	s := settings{separator: defValueSep}
	for _, opt := range opts {
		opt(&s)
	}

	if s.prefix != "" && !strings.HasSuffix(s.prefix, "_") {
		s.prefix += "_"
	}

	return s
}
