# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [2.0.0] - 2026-06-25

Version 2.0.0 is a major rewrite. The API is reorganized around two familiar
shapes — a small file-loading API and an `encoding/json`-style struct API —
and the `.env` parser is brought in line with the de-facto `.env` format.
Several long-standing bugs are fixed. See the
[migration guide](#migration-from-v1) below.

### Added

- File loading family: `Load`, `Overload`, `LoadRaw`, `OverloadRaw` (variadic,
  default to `.env`, "first value wins" across files).
- Reader loading: `LoadReader(io.Reader)` for `embed.FS`, the network or a
  string.
- Side-effect-free parsing into a map: `Read`, `ReadRaw`, `Parse`, `ParseRaw`.
- Struct mapping variants: `UnmarshalMap`, `UnmarshalFile`, `MarshalMap`,
  `MarshalFile`.
- Functional options: `WithPrefix`, `WithSeparator`, `WithTimeLayout`
  (precedence: field tag > option > built-in default).
- `time.Duration` (`30s`, `1h30m`) and `time.Time` support; a `layout` tag and
  `WithTimeLayout` option (Go layout or a constant name such as `DateOnly`).
- Inline `required` flag (`env:"KEY,required"`) and the `env:"-"` ignore tag
  (the ignore behaviour previously did not work).
- Multi-line quoted values (the long-documented feature is now implemented).
- Typed sentinel errors: `ErrNilObject`, `ErrNotPointer`, `ErrNotStruct`,
  `ErrEmptyStruct`, `ErrInvalidObject`, `ErrRequired` (testable with
  `errors.Is`); conversion errors are wrapped with `%w`.
- Field-level support for `encoding.TextMarshaler` and
  `encoding.TextUnmarshaler`, so types such as `net.IP`, `netip.Addr`,
  `big.Int`, `slog.Level` and your own enums work automatically (including
  slices, arrays and pointers of them).
- `MustLoad` (panics on error, for `init`/`main`), `All` (an `iter.Seq2`
  iterator over a file's pairs) and the `WithFileMode` option for `MarshalFile`
  (default `0o644`; use `0o600` for secrets).
- `WithParser[T]` and `WithEncoder[T]` options register a decoder/encoder for a
  type you do not control and that does not implement
  `encoding.TextUnmarshaler`/`TextMarshaler`; they apply to the type and to
  slices, arrays and pointers of it, and take precedence over the built-ins.
- `MarshalWriter(w io.Writer, …)` and `UnmarshalReader(r io.Reader, …)` complete
  the reader/writer symmetry (counterparts of `LoadReader` and `UnmarshalFile`).
- Full reference documentation: `DOC.md` (English) and `DOC.UK.md` (Ukrainian),
  plus runnable `Example` functions.

### Changed

- `Unmarshal` and `Marshal` are options-based; the positional `prefix` argument
  becomes the `WithPrefix` option. `Marshal` now returns only an `error`.
- `Marshaler`/`Unmarshaler` mirror `encoding/json`: `MarshalEnv` returns a
  `map[string]string` and `UnmarshalEnv` receives the resolved source map.
- Parsing is now spec-compliant: empty values (`KEY=`), trimmed unquoted
  values, single quotes and backticks are literal (no expansion), and escape
  sequences (`\n`, `\t`, `\r`, `\\`, `\"`) are interpreted in double quotes.
- Decoding follows `encoding/json` presence rules: an absent key leaves the
  field untouched (so in-code defaults survive), a present but empty value
  (`KEY=`) sets the zero value, and a slice is replaced rather than appended to.
- The default list separator is now a comma (was a space), which avoids data
  loss for values that contain spaces. Override it with the `sep` tag or
  `WithSeparator`.
- `bool` fields accept `yes`/`no` and `on`/`off` (case-insensitive) in addition
  to the `strconv.ParseBool` literals; the previous "float greater than 0.7"
  heuristic is removed.
- A nil pointer field is optional: it decodes as nil when its key is absent and
  is omitted on encode, so optional values round-trip.
- Conversion errors now include the offending key (e.g. `PORT: ...`).
- Struct field tags are parsed once per type and cached (like `encoding/json`),
  which speeds up repeated `Unmarshal`/`Marshal` of the same type and reduces
  allocations.
- `go.mod` requires Go 1.24; the package has no third-party dependencies.

### Removed

- `LoadSafe`, `Update`, `UpdateSafe` (replaced by `LoadRaw`/`Overload`/
  `OverloadRaw`).
- `Save` (replaced by `MarshalFile`).
- `ParallelTasks` and the concurrent parser — parsing is now sequential, which
  is simpler, race-free and faster on real files. The `golang.org/x/sync`
  dependency is dropped.

### Fixed

- `splitN` corrupted non-ASCII values and separators (and invalid UTF-8); it is
  rewritten to be rune-correct, byte-preserving and O(n) (also far fewer
  allocations).
- Empty values (`KEY=`, `KEY=""`) and a space after `=` are now valid.
- Floats marshal to their shortest round-tripping form instead of the `%f`
  six-decimal form (`3.14` no longer becomes `3.140000`).
- A data race on the global parallel-tasks state and a dead `break` in the
  reader loop are gone with the sequential parser.
- Quoted values are parsed in a single escape-aware pass; the previous
  `crypto/rand` marker (whose error was ignored) is removed.
- The struct mapper no longer panics on nil pointer fields, a nil object passed
  to `Marshal`, decoding into a nil scalar pointer, or unexported fields
  (unexported fields are skipped, like `encoding/json`).
- An unquoted `#` starts an inline comment only when preceded by whitespace, so
  values such as `pass#word`, `#fff` and URL fragments are no longer silently
  truncated.
- `[]*T` and `[N]*T` (slices/arrays of pointers) now decode, with an empty
  element becoming a nil element.
- `MarshalFile` quotes values that would otherwise produce an invalid `.env`
  (a newline, leading/trailing whitespace or an inline-comment `#`), so the
  file round-trips through `UnmarshalFile`.
- A pointer-receiver `MarshalText` is now honoured on encode (it used to be
  ignored, encoding the default kind representation).
- Values larger than `bufio.Scanner`'s default ~64 KiB limit (PEM chains, keys,
  JWTs, base64 blobs) now parse.
- `WithPrefix` is applied to the keys of a custom `Marshaler`, the same as for
  reflective structs.

### Migration from v1

| v1 | v2 |
|----|----|
| `Load(file)` | `Load(files...)` — same defaults, now variadic |
| `LoadSafe(file)` | `LoadRaw(files...)` |
| `Update(file)` | `Overload(files...)` |
| `UpdateSafe(file)` | `OverloadRaw(files...)` |
| `Unmarshal(prefix, &v)` | `Unmarshal(&v, env.WithPrefix(prefix))` |
| `Marshal(prefix, v) ([]string, error)` | `Marshal(v, env.WithPrefix(prefix)) error` |
| `Save(file, prefix, v)` | `MarshalFile(file, v, env.WithPrefix(prefix))` |
| `MarshalEnv() ([]string, error)` | `MarshalEnv() (map[string]string, error)` |
| `UnmarshalEnv() error` | `UnmarshalEnv(map[string]string) error` |
| `ParallelTasks(n)` | removed (parsing is sequential) |

Notes:

- `WithPrefix` normalizes the namespace: `WithPrefix("APP")` and
  `WithPrefix("APP_")` are equivalent and both map `PORT` to `APP_PORT`. In v1
  the trailing `_` had to be written by hand.
- For custom `Marshaler` types, return a `map[string]string` from `MarshalEnv`
  instead of calling `env.Set` yourself — the library writes the values.
- For custom `Unmarshaler` types, read from the `data map[string]string`
  argument instead of calling `env.Get`/`os.Getenv`.
- Values that legitimately contain `$` should be loaded with the `Raw` variants
  (or wrapped in single quotes in the `.env` file) to avoid expansion.
