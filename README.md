# hes

[![Build Status](https://github.com/vicanso/hes/workflows/Test/badge.svg)](https://github.com/vicanso/hes/actions)

Structured HTTP errors for Go.

`hes` v1 is a clean break from v0: standard `errors` chain support, options-based construction, immutable helpers, and safe JSON responses.

## Install

```bash
go get github.com/vicanso/hes@v1.0.0
```

Requires Go 1.22+.

## Quick start

```go
package main

import (
	"fmt"
	"net/http"

	"github.com/vicanso/hes"
)

func main() {
	err := hes.New("invalid email",
		hes.WithCode("user.invalid_email"),
		hes.WithCategory("user"),
		hes.WithStatus(http.StatusUnprocessableEntity),
		hes.WithExtra("field", "email"),
	)

	fmt.Println(err)
	// statusCode=422, category=user, code=user.invalid_email, message=invalid email

	buf, _ := err.MarshalPublic()
	fmt.Println(string(buf))
}
```

## Construct

| Function | Default status | Notes |
|----------|----------------|-------|
| `New(msg, opts...)` | 400 | Client-facing errors |
| `Wrap(err, opts...)` | 500 + `Exception` | Unknown errors; **uses `errors.As`** — chain `*Error` is extracted, not re-wrapped |
| `From(err)` | — | `errors.As` then `Wrap` |
| `BadRequest` | 400 | Alias of `New` |
| `Unauthorized` / `Forbidden` / `NotFound` / `RequestTimeout` / `Conflict` / `UnsupportedMediaType` / `UnprocessableEntity` / `TooManyRequests` | 4xx | Thin helpers |
| `Internal` / `BadGateway` / `ServiceUnavailable` / `GatewayTimeout` | 5xx + `Exception` | Server failures |

### Options

```go
hes.WithStatus(403)
hes.WithCode("auth.forbidden")
hes.WithCategory("auth")
hes.WithSubCategory("token")
hes.WithTitle("Forbidden")
hes.WithException()
hes.WithExtra("userId", 1)
hes.WithExtras(map[string]any{"a": 1})
hes.WithCause(err)
hes.WithCaller(0) // capture call site
```

### Immutable updates

Prefer deriving new values over mutating fields:

```go
base := hes.NotFound("user not found", hes.WithCode("user.not_found"))
err := base.WithExtra("id", userID).WithMessage("user 42 not found")
```

### Aggregate validation errors

```go
err := hes.BadRequest("validation failed", hes.WithCode("validation")).
	Append(
		hes.New("email required", hes.WithCode("field.email")),
		hes.New("name too short", hes.WithCode("field.name")),
	)
```

## Standard library interop

```go
cause := errors.New("db timeout")
err := hes.Wrap(cause, hes.WithCode("db.timeout"))

errors.Is(err, cause)  // true via Unwrap
hes.Is(err)            // true — chain contains *hes.Error
he, ok := hes.As(err)  // extract *hes.Error

// Match by application code (both sides need non-empty Code)
sentinel := &hes.Error{Code: "db.timeout"}
errors.Is(err, sentinel) // true

// Code-less: match when client-facing fields are equal
a := &hes.Error{StatusCode: 404, Category: "user", Message: "not found"}
b := &hes.Error{StatusCode: 404, Category: "user", Message: "not found"}
errors.Is(a, b) // true — same data
```

`errors.Is` / `(*Error).Is` rules (also used by `Append` dedup):

1. Pointer identity (handled by the standard library first)
2. Both have non-empty `Code` → codes equal
3. Otherwise → same `StatusCode`, `Category`, `SubCategory`, `Code`, `Title`, `Message`

## JSON responses

- `ToJSON()` — full error (may include `file` / `line`)
- `MarshalPublic()` — client view, **no tree clone** (strips caller)
- `Public()` — deep clone with caller stripped (when you need an `*Error`)
- `WriteJSON(w)` / `WriteJSONTo(w)` — public JSON; nil receiver returns `hes.ErrNil`

```go
func handle(w http.ResponseWriter, r *http.Request) {
	err := doWork()
	if err != nil {
		he := hes.From(err)
		_ = he.WriteJSON(w)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
```

Example public payload:

```json
{
  "statusCode": 400,
  "code": "user.invalid_email",
  "category": "user",
  "message": "invalid email",
  "extra": {"field": "email"}
}
```

## Caller capture

```go
// Per error
err := hes.New("boom", hes.WithCaller(0))

// Global default for New/Wrap
hes.SetDefaultCaller(true)
hes.SetFileConverter(func(file string) string {
	return strings.TrimPrefix(file, "/app/")
})
```

## Migration from v0

v1 removes the combinatorial `NewWith*` API and mutex helpers.

| v0 | v1 |
|----|----|
| `New(msg, category)` | `New(msg, WithCategory(category))` |
| `NewWithStatusCode(msg, code, cat)` | `New(msg, WithStatus(code), WithCategory(cat))` |
| `NewWithError(err)` | `Wrap(err)` (default **500**, not 400) |
| `NewWithErrorStatusCode(err, code)` | `Wrap(err, WithStatus(code))` |
| `NewWithCaller(msg)` | `New(msg, WithCaller(0))` |
| `NewWithException(msg)` | `New(msg, WithException())` or `Internal(msg)` |
| `NewMutex(...)` | removed — treat errors as immutable |
| `he.Add(err)` | `he = he.Append(err)` |
| `he.AddExtra(k, v)` | `he = he.WithExtra(k, v)` |
| `he.CloneWithMessage(m)` | `he.WithMessage(m)` |
| `IsError(err)` | `Is(err)` / `errors.As` |
| `he.Err` | `he.Cause` |
| `he.ToJSON()` → `[]byte` | `he.ToJSON()` → `([]byte, error)` |
| `EnableCaller(true)` | `SetDefaultCaller(true)` |
| `SetFileConvertor(fn)` | `SetFileConverter(fn)` |

### Semantic changes

1. **`Wrap` defaults to 500 + Exception** (v0 used 400).
2. **`Wrap` / `From` walk the error chain** via `errors.As` (do not double-wrap `*Error`).
3. **`Clone` / `With*` deep-copy** `Extra` and `Errs`; `Append` shares leaf pointers under the immutability convention.
4. **No mutex API** — do not mutate errors across goroutines; derive new values.
5. **`errors.Is` matches by `Code`, or by structural fields** when Code is empty (aligned with Append dedup).
6. **`ToJSON()` returns `([]byte, error)`**; prefer `MarshalPublic` / `WriteJSON` for clients.

## License

Apache-2.0
