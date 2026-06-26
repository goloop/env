# env — довідник

Повний довідник пакета `github.com/goloop/env/v2`. Стислий огляд — у
[README](README.md). English version: [DOC.md](DOC.md).

## Зміст

- [Ментальна модель](#ментальна-модель)
- [Завантаження файлів в оточення](#завантаження-файлів-в-оточення)
- [Парсинг у map](#парсинг-у-map)
- [Декодування у структуру](#декодування-у-структуру)
- [Кодування структури](#кодування-структури)
- [Опції](#опції)
- [Теги структур](#теги-структур)
- [Підтримувані типи](#підтримувані-типи)
- [Кастомний маршалінг](#кастомний-маршалінг)
- [Помічники оточення](#помічники-оточення)
- [Помилки](#помилки)
- [Формат .env](#формат-env)
- [Рецепти й поради](#рецепти-й-поради)

## Ментальна модель

Пакет переміщує конфігурацію між трьома місцями:

```
файл .env / io.Reader  ──►  процесне оточення (os.Environ)  ──►  Go-структура
       │                              ▲                               ▲
       └──────────────►  map[string]string  ──────────────────────────┘
```

- **Завантаження** (`Load`, `Overload`, …) пише дані файлу в **процесне
  оточення**. Значення стають рядками, видимими для `os.Getenv`, дочірніх
  процесів та інших бібліотек.
- **Парсинг** (`Read`, `Parse`) перетворює дані файлу/reader на звичайну
  **`map[string]string`** без жодних побічних ефектів.
- **Декодування** (`Unmarshal`, `UnmarshalMap`, `UnmarshalFile`) заповнює
  **типізовану структуру** з оточення, map або файлу.
- **Кодування** (`Marshal`, `MarshalMap`, `MarshalFile`) серіалізує структуру
  в оточення, map або файл.

Ключове правило: голі `Marshal`/`Unmarshal` працюють із **глобальним**
процесним оточенням; варіанти `*Map` і `*File` — **ні**. У тестах,
конкурентному чи multi-tenant коді бери чисті варіанти.

## Завантаження файлів в оточення

Ці функції читають один або кілька `.env`-файлів (варіадик; без аргументів —
дефолт `.env`) і пишуть результат у процесне оточення. Якщо файлів кілька,
**перше задане значення ключа виграє**.

| Функція | Розгортання | Наявні ключі |
|---------|-------------|--------------|
| `Load(filenames ...string) error`        | `${VAR}` розгортається | зберігаються |
| `Overload(filenames ...string) error`    | `${VAR}` розгортається | перезаписуються |
| `LoadRaw(filenames ...string) error`     | літерально | зберігаються |
| `OverloadRaw(filenames ...string) error` | літерально | перезаписуються |

```go
// Завантажити .env, зберігаючи те, що вже є в оточенні.
if err := env.Load(".env"); err != nil {
	log.Fatal(err)
}

// Нашарування файлів: спершу базовий, потім локальні перевизначення.
if err := env.Overload(".env", ".env.local"); err != nil {
	log.Fatal(err)
}
```

Варіанти `Raw` пропускають розгортання `${VAR}`/`$VAR` і зберігають значення
дослівно. Використовуй їх, коли значення законно містять `$` і не мають
інтерполюватися.

`MustLoad(filenames ...string)` — як `Load`, але панікує при помилці; зручно в
`init`/`main`, де відсутній або битий конфіг має зупинити програму:

```go
func init() { env.MustLoad(".env") }
```

### LoadReader

```go
func LoadReader(r io.Reader) error
```

Завантажує `.env`-дані з будь-якого reader в оточення (розгортання увімкнено,
наявні ключі зберігаються). Зручно для вбудованих файлів, мережі або рядка.

```go
//go:embed config.env
var configFS embed.FS

f, _ := configFS.Open("config.env")
defer f.Close()
if err := env.LoadReader(f); err != nil {
	log.Fatal(err)
}
```

## Парсинг у map

Ці функції повертають `map[string]string` і **ніколи не чіпають оточення**.

| Функція | Джерело | Розгортання |
|---------|---------|-------------|
| `Read(filenames ...string) (map[string]string, error)`    | файли  | так |
| `ReadRaw(filenames ...string) (map[string]string, error)` | файли  | ні  |
| `Parse(r io.Reader) (map[string]string, error)`           | reader | так |
| `ParseRaw(r io.Reader) (map[string]string, error)`        | reader | ні  |

```go
m, err := env.Read(".env")
if err != nil {
	log.Fatal(err)
}
fmt.Println(m["HOST"])

// Парс із рядка.
m, _ = env.Parse(strings.NewReader("HOST=localhost\nPORT=8080\n"))
```

`All(filenames ...string) iter.Seq2[string, string]` — зручний ітератор пар
файлу, щоб перебирати без побудови map. **Помилки читання/парсингу мовчки
ігноруються** — відсутній чи битий файл просто нічого не видає, незрізниме від
порожнього. **Якщо треба обробляти збої — бери `Read` (повертає `error`).**

```go
for key, value := range env.All(".env") {
	fmt.Println(key, value)
}
```

Розгортання в `Read`/`Parse` резолвить `${VAR}` проти раніших ключів того ж
джерела і, як запасний варіант, проти поточного процесного оточення — нічого
не записуючи назад.

## Декодування у структуру

```go
func Unmarshal(v any, opts ...Option) error                          // з os.Environ
func UnmarshalMap(m map[string]string, v any, opts ...Option) error  // з map
func UnmarshalFile(filename string, v any, opts ...Option) error     // з файлу
func UnmarshalReader(r io.Reader, v any, opts ...Option) error        // з reader
```

`v` має бути ненульовим вказівником на структуру. Поля зіставляються за тегом
`env` (або іменем поля, якщо тег порожній). `UnmarshalMap` і `UnmarshalFile`
не чіпають оточення.

```go
type Config struct {
	Host string `env:"HOST"`
	Port int    `env:"PORT" def:"80"`
}

// З оточення.
var c Config
if err := env.Unmarshal(&c); err != nil {
	log.Fatal(err)
}

// З map (оточення не задіяне).
_ = env.UnmarshalMap(map[string]string{"HOST": "localhost", "PORT": "9000"}, &c)

// З файлу напряму (парсить файл, оточення не задіяне).
_ = env.UnmarshalFile(".env", &c)
```

`Unmarshal(&cfg)` і `UnmarshalFile(".env", &cfg)` схожі, але різняться в
одному важливому: `Unmarshal` читає **процесне оточення** (тож файл треба
спершу `Load`-нути, щоб його значення там опинились), а `UnmarshalFile` читає
**файл напряму** й лишає оточення недоторканим.

### Дефолти й відсутні ключі

Декодування дотримується тих самих presence-правил, що й `encoding/json`:

- Ключ **присутній** у джерелі — встановлює поле, навіть коли значення порожнє
  (`KEY=` ставить нульове значення й очищає слайс/масив).
- Ключ **відсутній** — лишає поле **недоторканим**, тож in-code дефолт зберігається:

  ```go
  cfg := Config{Port: 8080}
  env.Unmarshal(&cfg) // PORT не задано -> cfg.Port лишається 8080
  ```

- Тег `def` дає значення, що вживається **лише коли ключ відсутній**. Присутнє
  порожнє значення (`KEY=`) — це явний нуль; воно **не** відкочується до `def`.
- Слайс **замінюється**, а не доповнюється, тож декодування ідемпотентне й
  перекриває будь-який in-code дефолт. Вкладена структура декодується «на місці»,
  тож її під-поля, чиїх ключів нема, зберігають значення.

### Списки

Поле-слайс/масив кодується склеюванням елементів через роздільник (тег `sep` чи
`WithSeparator`, дефолт — кома) і декодується розбивкою по ньому. Елемент, що
містить роздільник (чи лапку/дужку), **автоматично береться в лапки**, тож
список round-trip-иться навіть із такими значеннями:

```
[]string{"a,b", "c"}   <->   "a,b",c
```

У рукописному файлі можна взяти елемент у лапки самому: `TAGS="a,b",c`.

При записі у файл чи `io.Writer` значення, яке інакше прочиталося б невірно,
береться в лапки автоматично: newline, край-пробіли й inline-`#` екрануються, а
значення з `$` пишеться в одинарних лапках, щоб **не** експандитись при читанні.
Тож `MarshalFile`/`MarshalWriter` round-trip-ляться через
`UnmarshalFile`/`UnmarshalReader`.

## Кодування структури

```go
func Marshal(v any, opts ...Option) error                         // в os.Environ
func MarshalMap(v any, opts ...Option) (map[string]string, error) // в map
func MarshalFile(filename string, v any, opts ...Option) error    // у файл
func MarshalWriter(w io.Writer, v any, opts ...Option) error      // у writer
```

`Marshal` пише кожне поле в процесне оточення (з перезаписом). Варіанти `*Map`
і `*File` оточення не змінюють. `MarshalMap` парний до `UnmarshalMap` для
round-trip.

```go
type Config struct {
	Host  string   `env:"HOST"`
	Port  int      `env:"PORT"`
	Hosts []string `env:"HOSTS" sep:":"`
}
cfg := Config{Host: "localhost", Port: 8080, Hosts: []string{"a", "b"}}

env.Marshal(cfg)                 // HOST/PORT/HOSTS у оточенні
m, _ := env.MarshalMap(cfg)      // m["HOSTS"] == "a:b"
_ = env.MarshalFile(".env", cfg) // пише рядки KEY=value у .env
```

## Опції

Опції задають **дефолти рівня виклику**, які per-field тег може перебити.
Старшинство завжди: **тег поля > опція > вшитий дефолт**.

```go
func WithPrefix(prefix string) Option
func WithSeparator(sep string) Option
func WithTimeLayout(layout string) Option
func WithFileMode(mode os.FileMode) Option
func WithParser[T any](parse func(string) (T, error)) Option
func WithEncoder[T any](encode func(T) (string, error)) Option
```

### WithPrefix

Іменує неймспейс ключів. Префікс — це рівень; рівні з'єднуються через `_`.
Кінцевий `_` додається автоматично, якщо його нема; порожній префікс не додає
провідного `_`. `WithPrefix("APP")` і `WithPrefix("APP_")` еквівалентні.

```go
type Service struct {
	Port int `env:"PORT"`
}
var app, db Service
env.Unmarshal(&app, env.WithPrefix("APP")) // читає APP_PORT
env.Unmarshal(&db, env.WithPrefix("DB"))   // читає DB_PORT
```

> Порада: для фіксованих неймспейсів краще **вкладені структури** — вони
> читають ті самі значення одним викликом:
>
> ```go
> type Config struct {
>     App Service `env:"APP"` // читає APP_PORT
>     DB  Service `env:"DB"`  // читає DB_PORT
> }
> env.Unmarshal(&cfg)
> ```
>
> `WithPrefix` лишай для рантайм/динамічних префіксів (напр. multi-tenant).

### WithSeparator

Задає дефолтний роздільник для полів-слайсів/масивів без тега `sep`. Вшитий
дефолт — кома.

```go
type Config struct {
	Hosts []string `env:"HOSTS"` // нема тега sep -> бере опцію
}
var c Config
env.UnmarshalMap(map[string]string{"HOSTS": "a,b,c"}, &c, env.WithSeparator(","))
```

### WithTimeLayout

Задає дефолтний layout для полів `time.Time` без тега `layout`. Приймає
Go-формат reference-time або ім'я стандартної константи. Вшитий дефолт —
RFC3339.

```go
env.Unmarshal(&c, env.WithTimeLayout("DateOnly"))
```

### WithFileMode

Задає права, з якими `MarshalFile` створює файл. Дефолт — `0o644`; для файлів
із секретами використовуй `0o600`.

```go
env.MarshalFile(".env", cfg, env.WithFileMode(0o600))
```

### WithParser / WithEncoder

Реєструє декодер (і енкодер) для типу, який ти не контролюєш і який не реалізує
`encoding.TextUnmarshaler`/`TextMarshaler`. Зареєстрована функція має пріоритет
над вбудованою обробкою цього типу й діє на сам тип та на слайси/масиви/
вказівники на нього.

```go
// Money — тип з іншого пакета, зі своїми ParseMoney/String.
opts := []env.Option{
	env.WithParser(func(s string) (Money, error) { return ParseMoney(s) }),
	env.WithEncoder(func(m Money) (string, error) { return m.String(), nil }),
}
env.Unmarshal(&cfg, opts...)
m, _ := env.MarshalMap(cfg, opts...)
```

Можна зареєструвати лише парсер — тоді кодування впаде на вбудовану обробку;
для чистого round-trip реєструй обидва.

## Теги структур

```go
type Config struct {
	Host    string        `env:"HOST"`
	Port    int           `env:"PORT" def:"8080"`
	Hosts   []string      `env:"HOSTS" sep:","`
	Started time.Time      `env:"STARTED_AT" layout:"2006-01-02"`
	Token   string        `env:"TOKEN,required"`
	Secret  string        `env:"-"`
}
```

| Тег | Опис |
|-----|------|
| `env` | Ім'я ключа. `-` повністю ігнорує поле. Inline-прапорці йдуть після імені через кому, напр. `env:"KEY,required"`. |
| `def` | Дефолтне значення, коли ключа нема в джерелі. |
| `sep` | Роздільник для слайсів/масивів (перебиває `WithSeparator`). |
| `layout` | Layout для `time.Time` (перебиває `WithTimeLayout`). Go-формат або ім'я константи: `RFC3339`, `RFC1123`, `DateTime`, `DateOnly`, `TimeOnly`, `Kitchen`, `ANSIC`, `UnixDate`, `Stamp`. |

### required

`env:"KEY,required"` робить поле обов'язковим. Якщо ключа нема в джерелі **і**
не задано `def`, декодування повертає помилку, що загортає `ErrRequired`:

```go
type Config struct {
	Token string `env:"TOKEN,required"`
}
err := env.UnmarshalMap(map[string]string{}, &Config{})
// err: "env: required key is not set: TOKEN"
errors.Is(err, env.ErrRequired) // true
```

`def` задовольняє вимогу (дефолт — це свідома обробка відсутності), тож
`required` разом із `def` ніколи не помиляється.

### Ігнорування поля

`env:"-"` пропускає поле і при декодуванні, і при кодуванні — корисно для
обчислюваних чи секретних полів, які не треба мапити.

## Підтримувані типи

| Категорія | Типи |
|-----------|------|
| Рядки     | `string` |
| Булеві    | `bool` (`true`/`false`, `1`/`0`, `t`/`f` за `strconv.ParseBool`, а також `yes`/`no` та `on`/`off`, регістронезалежно) |
| Цілі      | `int`, `int8`, `int16`, `int32`, `int64` (з перевіркою діапазону) |
| Беззнакові| `uint`, `uint8`, `uint16`, `uint32`, `uint64` (з перевіркою діапазону) |
| Дробові   | `float32`, `float64` (найкоротше зворотне представлення) |
| URL       | `url.URL` |
| Час       | `time.Duration` (`30s`, `1h30m`), `time.Time` (за layout) |
| Складені  | вкладені структури, вказівники, слайси й масиви з вищезгаданого |

```go
type Config struct {
	Debug    bool          `env:"DEBUG"`
	Workers  uint8         `env:"WORKERS" def:"4"`
	Ratio    float64       `env:"RATIO"`
	Endpoint url.URL       `env:"ENDPOINT"`
	Timeout  time.Duration `env:"TIMEOUT" def:"30s"`
	StartAt  time.Time     `env:"START_AT"` // RFC3339 за замовчуванням
	Ports    []int         `env:"PORTS" sep:","`
	Limits   [3]int        `env:"LIMITS" sep:":"` // довжина перевіряється
}
```

Масиви перевіряють довжину: декодувати більше елементів, ніж вміщає масив, —
помилка. Декодування порожнього значення дає порожній слайс (а масив лишає з
нульовими значеннями).

### Кастомні типи полів

Будь-яке поле, чий тип реалізує `encoding.TextUnmarshaler` (а для кодування —
`encoding.TextMarshaler`), підтримується автоматично: значення парситься через
`UnmarshalText` і форматується через `MarshalText`. Це покриває багато
стандартних і сторонніх типів (`net.IP`, `netip.Addr`, `big.Int`,
`slog.Level`…) та твої власні enum-и, включно зі слайсами, масивами й
вказівниками на них.

```go
type Config struct {
	BindIP net.IP   `env:"BIND_IP"`       // "0.0.0.0" -> net.IP
	Level  Level    `env:"LOG_LEVEL"`     // твій enum через UnmarshalText
	Peers  []net.IP `env:"PEERS" sep:","` // і слайси теж
}
```

Спецкейс `time.Time` зберігає свій тег `layout` (його обробляють до шляху
`TextUnmarshaler`), а `url.URL` парситься напряму.

> Потрібен тип, який ти не контролюєш і який не має `TextUnmarshaler`? Зареєструй
> парсер через `WithParser` (і енкодер через `WithEncoder`) — див. Опції. Або
> загорни його в тонкий іменований тип із методом `UnmarshalText`.

### Опціональні поля (вказівники)

Поле-вказівник моделює *опціональне* значення: `nil` означає «відсутнє». Пакет
обробляє відсутність однаково в обидва боки, тож опціональні значення
зберігаються при round-trip:

- **Декодування** алокує вказівник лише коли є що присвоїти — ключ присутній
  (навіть порожній) або задано `def`. Якщо ключа нема й нема дефолту — вказівник
  лишається `nil`.
- **Кодування** пропускає nil-вказівник (ключ не виводиться).
- **nil-елемент слайсу вказівників** виражається позиційно: порожнім значенням
  на своїй позиції (`[]*int{a, nil, b}` → `"1,,3"`).
- Для **nil-вказівника на вкладену структуру** декодування алокує його лише
  коли в джерелі є хоча б один ключ під його префіксом; інакше — `nil`.

```go
type Config struct {
	Port *int `env:"PORT"` // nil коли PORT не задано, *значення коли задано
}
```

Чому саме так:

- **У `.env` немає `null`.** На відміну від JSON, у `.env`-файлі нема літерала
  null — ключ або заданий рядком, або відсутній. Тому чесне відображення
  «не задано» — це відсутній ключ, який і виводить кодування й споживає
  декодування. Як наслідок `MarshalMap` → `UnmarshalMap` повертає nil-вказівники
  назад у `nil`.
- **Це дзеркало `encoding/json`.** `json.Unmarshal` алокує вказівник лише коли
  ключ присутній, інакше лишає nil — те саме правило тут.
- **Це зберігає опціональність.** Сенс вказівника в конфіг-структурі — відрізнити
  «не задано» (`nil`) від «задано нульове значення» (вказівник на `0`/`""`).
  Завжди алокувати означало б стерти цю різницю.

## Кастомний маршалінг

Реалізуй ці інтерфейси, щоб узяти повний контроль над типом, точно як у
`encoding/json`:

```go
type Marshaler interface {
	MarshalEnv() (map[string]string, error)
}

type Unmarshaler interface {
	UnmarshalEnv(data map[string]string) error
}
```

`MarshalEnv` повертає пари ключ/значення; бібліотека вирішує, куди їх покласти
(оточення, map чи файл). `UnmarshalEnv` отримує вже зрезолвлену (розгорнуту)
map джерела й заповнює значення сам — рефлексивна обробка тегів повністю
пропускається.

```go
type Config struct {
	Host string
	Port int
}

func (c *Config) UnmarshalEnv(data map[string]string) error {
	c.Host = data["HOST"]
	c.Port, _ = strconv.Atoi(data["PORT"])
	return nil
}

func (c Config) MarshalEnv() (map[string]string, error) {
	return map[string]string{
		"HOST": c.Host,
		"PORT": strconv.Itoa(c.Port),
	}, nil
}
```

## Помічники оточення

Тонкі обгортки над стандартним пакетом `os`, без залежностей:

| Функція | Еквівалент |
|---------|------------|
| `Get(key) string`            | `os.Getenv` |
| `Set(key, value) error`      | `os.Setenv` |
| `Unset(key) error`           | `os.Unsetenv` |
| `Clear()`                    | `os.Clearenv` |
| `Environ() []string`         | `os.Environ` |
| `Expand(value) string`       | `os.Expand` з `os.Getenv` |
| `Lookup(key) (string, bool)` | `os.LookupEnv` |
| `Exists(keys ...string) bool`| true, якщо задано кожен ключ |

## Помилки

Помилки валідації — типізовані sentinel-и, перевіряються через `errors.Is`:

| Помилка | Значення |
|---------|----------|
| `ErrNilObject`    | об'єкт, переданий у `Unmarshal`/`Marshal`, дорівнює `nil` |
| `ErrNotPointer`   | об'єкт не є ненульовим вказівником на структуру |
| `ErrNotStruct`    | об'єкт не вказує на структуру |
| `ErrEmptyStruct`  | структура без полів |
| `ErrInvalidObject`| `Marshal` отримав не структуру й не вказівник на неї |
| `ErrRequired`     | обов'язкове поле без значення й без дефолту |

```go
if err := env.Unmarshal(&cfg); errors.Is(err, env.ErrRequired) {
	// бракувало обов'язкового ключа
}
```

Помилки конвертації від `strconv` і `time` повертаються як є, тож
`errors.Is(err, strconv.ErrSyntax)` тощо теж працюють.

## Формат .env

Парсер відповідає де-факто специфікації `.env`.

### Ключі

Ключ має відповідати `^[A-Za-z_][A-Za-z0-9_]*$` (POSIX-імена змінних
оточення): починається з літери або підкреслення й містить літери, цифри та
підкреслення. Дозволено опційний префікс `export`: `export KEY=value`.

### Значення й пробіли

```ini
KEY=value          # звичайне значення
EMPTY=             # порожнє значення валідне -> ""
SPACED=  trimmed   # пробіли навколо незакавиченого значення обрізаються
```

### Лапки

```ini
DOUBLE="value"     # обробляються змінні та escape
SINGLE='value'     # літерально: без розгортання, без escape
BACKTICK=`value`   # літерально, всередині можна ' і "
```

Закавичені значення зберігають внутрішні пробіли. Одинарні лапки й бектики
повністю літеральні, тож `'$USER'` лишається `$USER`.

### Escape-послідовності (лише подвійні лапки)

`\n`, `\t`, `\r`, `\\` і `\"` інтерпретуються в подвійних лапках. Одинарні
лапки й бектики зберігають бекслеші дослівно.

```ini
MESSAGE="рядок один\nрядок два"
```

### Багаторядкові значення

Закавичене значення може займати кілька фізичних рядків:

```ini
KEY="рядок один
рядок два
рядок три"
```

### Коментарі

```ini
# Повнорядковий коментар.
KEY=value          # inline-коментар (зверни увагу на пробіл перед #)
PASSWORD=p@ss#word # без пробілу перед #: символ # є частиною значення
COLOR=#fff         # # на початку значення є частиною значення
NOTE="символ # у лапках є частиною значення"
```

Поза лапками `#` починає коментар **лише коли йому передує пробіл/таб**. `#` на
початку значення або всередині токена — літеральний, тож hex-кольори (`#fff`),
URL-фрагменти (`http://x/#a`) і значення на кшталт `pass#word` зберігаються. У
лапках `#` завжди літеральний.

| Рядок | Значення |
|-------|----------|
| `K=abc#123`       | `abc#123` |
| `K=abc #123`      | `abc` |
| `K=a b # коментар` | `a b` |
| `K=#fff`          | `#fff` |

### Розгортання змінних

`${VAR}` і `$VAR` розгортаються в незакавичених і подвійно-закавичених
значеннях, резолвлячись проти раніших ключів того ж джерела й наявного
оточення. Одинарні лапки й бектики — літеральні.

```ini
USER=goloop
EMAIL="${USER}@example.com"   # -> goloop@example.com
LITERAL='${USER}'             # -> ${USER}
```

Варіанти `Raw`/`ParseRaw` повністю вимикають розгортання.

## Рецепти й поради

**Падай рано на неповній конфігурації.** Познач обов'язкові ключі `required`
і перевір помилку раз:

```go
if err := env.Unmarshal(&cfg); err != nil {
	log.Fatalf("config: %v", err)
}
```

**Тримай тести чистими.** Бери чисті варіанти, щоб тести не мутували
глобальний стан:

```go
m := map[string]string{"HOST": "localhost", "PORT": "8080"}
var cfg Config
env.UnmarshalMap(m, &cfg)
```

**Нашаровуй конфігурацію.** Завантаж базовий файл, потім перекрий локальним:

```go
env.Load(".env")            // база, не перезаписує оточення
env.Overload(".env.local")  // локальні перевизначення
```

**Round-trip.** `MarshalMap` і `UnmarshalMap` — взаємно зворотні:

```go
m, _ := env.MarshalMap(cfg)
var back Config
env.UnmarshalMap(m, &back) // back == cfg
```

**Вбудовані дефолти.** Поклади дефолтний `.env` через `embed.FS` і завантаж
його перед справжнім оточенням:

```go
//go:embed defaults.env
var defaults embed.FS
f, _ := defaults.Open("defaults.env")
env.LoadReader(f) // не перезаписує те, що вже задано
```
