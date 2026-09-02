# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.8.0] - 2026-09-02

Minor release: variable expansion stops reading the shell's positional
parameters. No API change, but every value containing a `$` is read the way it
is written from here on, so this is worth a look before upgrading.

### Fixed
- `${VAR}`/`$VAR` expansion no longer treats `$1` through `$9` (or `$@`, `$$`
  and the rest of the shell's specials) as variable names. `os.Expand`, which
  the loaders used, implements the shell's substitution language, and a `.env`
  file has no positional parameters - but it does have prices, awk and sed
  one-liners, regular expressions and passwords, and each of those quietly lost
  part of itself:

  ```ini
  PRICE=cost: $100          # was "cost: 00"
  DISCOUNT="save $5 today"  # was "save  today"
  AWK=$1 == "x"             # was " == \"x\""
  PASSWORD=pa$$word         # was "pa"
  ```

  A reference is now recognised only when it names a key this format could
  define, that is `[A-Za-z_][A-Za-z0-9_]*`. Anything else keeps its `$` as
  written, a doubled `$$` is never a reference, and single quotes and backticks
  remain the literal forms. A value that always named a real variable behaves
  exactly as before.
- An unterminated `${` is text rather than an instruction to swallow the rest
  of the value, which is what `os.Expand` did with it.

### Changed
- `Expand` was documented as a synonym for `os.Expand` and is no longer one: it
  now follows the same name rule as the loaders. The signature is unchanged.
  Naming one substitution language and implementing another in the same package
  was the underlying mistake; this is the half that had to change.

  This is why the release is minor rather than a patch. The defect is a defect,
  but the documented contract of a public function changed with it, and the
  reading of every value containing a `$` changed across `Load`, `Overload`,
  `Read`, `Parse` and their file and reader variants.

### Documentation
- The reference, the README and the package documentation state the name rule
  and show what keeps its `$`.

## [2.7.1] - 2026-08-11

Patch release.

### Documentation
- The reference covers the `absolute` tag flag and carries the canonical
  `Load` -> `Unmarshal` -> `Validate` recipe, both of which until now lived
  only in the godoc.

## [2.7.0] - 2026-08-11

Minor release: let a nested field name a variable in full.

### Added
- An inline `absolute` flag on the env tag - `env:"DATABASE_URL,absolute"` -
  names the environment variable exactly, ignoring the prefix its enclosing
  structs contribute. Deployments fix some names once and for all, and grouping
  fields in Go should not rename them: without this, moving pool settings into
  a DB struct turns DATABASE_URL into DB_DATABASE_URL and the config has to be
  flattened back out to match reality. The flag drops the whole prefix chain,
  including one set with `WithPrefix`, and `Marshal` writes the same name
  `Unmarshal` reads.

### Documentation
- The package documentation writes down the canonical `Load` -> `Unmarshal` ->
  `Validate` shape, why the third step is the application's own (a tag can
  express presence, not rules spanning several fields), and why `Unmarshal`
  does not call a `Validate` method for you even when one exists: forgetting to
  write the method is exactly as easy as forgetting to call it, and an implicit
  call would put application logic inside a decoder.

## [Unreleased]

## [2.6.0]

### Added
- `LoadSafe` loads .env files like `Load` but skips a file that does not exist
  instead of returning an error, while still reporting parse and other I/O
  errors. It serves the load-if-present pattern (read `.env` in development,
  fall back to the real environment in CI/production) without a manual
  `os.Stat` guard at the call site.

## [2.5.0]

Parser correctness and decode fixes. One behaviour change is noted below.

### Fixed
- A quote inside an inline comment on a closed single-line value
  (`KEY='abc' # don't`) no longer opens a multiline value and silently
  swallows the following lines. Multiline detection now uses the same
  escape-aware scan as the value parser.
- `Load` and `Read` agree on a key repeated within one file: the last
  occurrence wins for both, consistently with the de-facto `.env` behaviour.
- `splitN` counts nested brackets of the same kind, so a value like
  `{"a":{"b":1},"c":2},x` splits into the outer group and `x` instead of
  breaking at the first inner `}`.
- An empty nested struct (a field of type `struct{}` or `*struct{}`) no longer
  fails `Unmarshal` with `ErrEmptyStruct`; it is skipped, matching `Marshal`.
- A parser registered with `WithParser` for a pointer type (`*T`) is now
  honoured for a pointer field, instead of the value being silently dropped.

### Changed
- **Behaviour:** unexpected text after a closing quote (`J="abc"def`) is now a
  parse error rather than being silently discarded. Only whitespace and an
  inline `#` comment may follow the closing quote.

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
- `ReadSeq(files…) (iter.Seq2, error)` is the error-aware counterpart of `All`,
  and the `WithRequiredAll()` option makes every leaf field required during
  decoding (nested structs excluded).
- `MarshalString`/`UnmarshalString` are in-memory string counterparts of
  `MarshalFile`/`UnmarshalFile`.
- `Raw` codec variants — `UnmarshalFileRaw`/`UnmarshalReaderRaw`/
  `UnmarshalStringRaw` and `MarshalFileRaw`/`MarshalWriterRaw`/`MarshalStringRaw`
  — skip `${VAR}`/`$VAR` expansion, so any value (including `$` together with
  single quotes and backticks) round-trips verbatim.
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
- List values now round-trip even when an element contains the separator (or a
  quote/bracket): such elements are quoted on encode and unquoted on decode, so
  `[]string{"a,b","c"}` survives `Marshal`/`Unmarshal`.
- A value containing `$` written to a file or `io.Writer` is single-quoted so
  it is not expanded as `${VAR}`/`$VAR` when read back by `UnmarshalFile`/
  `UnmarshalReader`.
- `MarshalWriter` now reports a short write (via `io.Copy`) instead of silently
  dropping part of the output.
- A slice with a single empty element (`[]string{""}`) is distinguished from an
  empty slice (`[]string{}`) on round-trip — the former is written as `""`.

### Migration from v1

First, update the import path (v2 uses semantic import versioning):

    go get github.com/goloop/env/v2

```go
import "github.com/goloop/env/v2" // was "github.com/goloop/env"
```

Then adjust the calls:

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
