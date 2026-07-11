package hes

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	he := New("bad input", WithCategory("auth"), WithCode("invalid"))
	if he.Message != "bad input" {
		t.Fatalf("Message = %q", he.Message)
	}
	if he.StatusCode != http.StatusBadRequest {
		t.Fatalf("StatusCode = %d", he.StatusCode)
	}
	if he.Category != "auth" || he.Code != "invalid" {
		t.Fatalf("category/code = %s/%s", he.Category, he.Code)
	}
	if he.Exception {
		t.Fatal("New should not set Exception")
	}
}

func TestErrorString(t *testing.T) {
	he := &Error{
		StatusCode:  400,
		Category:    "custom",
		SubCategory: "input",
		Code:        "1-101",
		Title:       "Bad Request",
		Message:     "my error",
		Exception:   true,
	}
	want := "statusCode=400, category=custom, subCategory=input, code=1-101, title=Bad Request, message=my error, exception=true"
	if got := he.Error(); got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

func TestFormat(t *testing.T) {
	cause := errors.New("root cause")
	he := Wrap(cause, WithCode("db"))
	if got := fmt.Sprintf("%s", he); !strings.Contains(got, "message=root cause") {
		t.Fatalf("%s = %q", "%s", got)
	}
	if got := fmt.Sprintf("%q", he); got != `"root cause"` {
		t.Fatalf("%q = %s", "%q", got)
	}
	verbose := fmt.Sprintf("%+v", he)
	if !strings.Contains(verbose, "root cause") || !strings.Contains(verbose, "cause:") {
		t.Fatalf("%+v = %q", "%+v", verbose)
	}
}

func TestWrap(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		if Wrap(nil) != nil {
			t.Fatal("expected nil")
		}
	})

	t.Run("plain error", func(t *testing.T) {
		err := errors.New("boom")
		he := Wrap(err)
		if he.StatusCode != http.StatusInternalServerError {
			t.Fatalf("status = %d", he.StatusCode)
		}
		if !he.Exception {
			t.Fatal("expected Exception")
		}
		if he.Cause != err {
			t.Fatal("cause mismatch")
		}
		if he.Message != "boom" {
			t.Fatalf("message = %q", he.Message)
		}
	})

	t.Run("identity for *Error without opts", func(t *testing.T) {
		he := New("x")
		if Wrap(he) != he {
			t.Fatal("Wrap(*Error) should return same pointer")
		}
	})

	t.Run("identity through error chain", func(t *testing.T) {
		he := New("x", WithCode("c"))
		outer := fmt.Errorf("layer: %w", he)
		if Wrap(outer) != he {
			t.Fatal("Wrap should unwrap *Error from chain")
		}
		// must not re-wrap as 500 exception
		if Wrap(outer).Exception {
			t.Fatal("chained *Error should keep original flags")
		}
	})

	t.Run("clone when opts provided", func(t *testing.T) {
		he := New("x")
		wrapped := Wrap(he, WithCode("c1"))
		if wrapped == he {
			t.Fatal("expected clone")
		}
		if wrapped.Code != "c1" || he.Code != "" {
			t.Fatalf("code clone failed: wrapped=%q original=%q", wrapped.Code, he.Code)
		}
	})

	t.Run("opts on chained *Error", func(t *testing.T) {
		he := New("x")
		outer := fmt.Errorf("layer: %w", he)
		got := Wrap(outer, WithStatus(418))
		if got == he || got.StatusCode != 418 || he.StatusCode != 400 {
			t.Fatalf("clone+opts on chain failed: got=%+v he=%+v", got, he)
		}
	})
}

func TestFromAndIsAs(t *testing.T) {
	root := New("root", WithCode("root"))
	wrapped := fmt.Errorf("outer: %w", root)

	if !Is(wrapped) {
		t.Fatal("Is should find *Error in chain")
	}
	if Is(errors.New("nope")) {
		t.Fatal("Is should be false for plain error")
	}

	got, ok := As(wrapped)
	if !ok || got != root {
		t.Fatalf("As = %v, %v", got, ok)
	}
	if From(wrapped) != root {
		t.Fatal("From should return same *Error from chain")
	}

	plain := errors.New("x")
	he := From(plain)
	if he.Cause != plain || !he.Exception {
		t.Fatal("From plain error should Wrap")
	}
}

func TestUnwrapAndErrorsIs(t *testing.T) {
	cause := errors.New("db down")
	he := Wrap(cause, WithCode("db"))

	if !errors.Is(he, cause) {
		t.Fatal("errors.Is should match cause via Unwrap")
	}

	sentinel := &Error{Code: "db", Message: "ignored"}
	if !errors.Is(he, sentinel) {
		t.Fatal("errors.Is should match by Code")
	}

	outer := fmt.Errorf("wrap: %w", he)
	if !errors.Is(outer, sentinel) {
		t.Fatal("errors.Is should walk chain")
	}
}

func TestIsMatching(t *testing.T) {
	// Code match (Message may differ)
	a := &Error{Code: "x", Message: "a"}
	b := &Error{Code: "x", Message: "b"}
	if !errors.Is(a, b) {
		t.Fatal("same Code should match")
	}

	// Code-less: structural match on client-facing fields
	s := &Error{StatusCode: 404, Category: "x", Message: "m"}
	other := &Error{StatusCode: 404, Category: "x", Message: "m"}
	diff := &Error{StatusCode: 404, Category: "x", Message: "other"}
	if !errors.Is(s, other) {
		t.Fatal("same data should match")
	}
	if errors.Is(s, diff) {
		t.Fatal("different Message should not match")
	}
	if !errors.Is(s, s) {
		t.Fatal("pointer identity should match")
	}

	// Append dedup uses the same same() rules
	parent := New("p").Append(s, other)
	if len(parent.Errs) != 1 {
		t.Fatalf("dedup by same data: got %d errs", len(parent.Errs))
	}
}

func TestCloneDeepCopy(t *testing.T) {
	he := New("msg", WithExtra("k", "v"))
	he = he.Append(New("sub"))

	clone := he.Clone()
	clone.Extra["k"] = "changed"
	clone.Errs[0].Message = "mutated"
	clone.Message = "clone-msg"

	if he.Extra["k"] != "v" {
		t.Fatal("Extra map must be deep-copied")
	}
	if he.Errs[0].Message != "sub" {
		t.Fatal("Errs must be deep-copied")
	}
	if he.Message != "msg" {
		t.Fatal("Message must not be shared by assignment confusion")
	}
}

func TestWithMethods(t *testing.T) {
	base := New("base", WithCategory("c"))
	next := base.
		WithMessage("next").
		WithStatus(404).
		WithCode("nf").
		WithSubCategory("users").
		WithTitle("Missing").
		WithException(true).
		WithExtra("a", 1)

	if base.Message != "base" || base.StatusCode != 400 || base.Code != "" {
		t.Fatal("With* must not mutate original")
	}
	if next.Message != "next" || next.StatusCode != 404 || next.Code != "nf" || !next.Exception {
		t.Fatalf("unexpected next: %+v", next)
	}
	if next.SubCategory != "users" || next.Title != "Missing" {
		t.Fatalf("title/sub = %s/%s", next.Title, next.SubCategory)
	}
	if next.Extra["a"] != 1 {
		t.Fatal("extra missing")
	}
}

func TestAppend(t *testing.T) {
	he := New("parent")
	he2 := he.Append(New("child1"), nil, New("child2"))
	if he.IsNotEmpty() {
		t.Fatal("original should stay empty")
	}
	if he2.IsEmpty() || len(he2.Errs) != 2 {
		t.Fatalf("errs = %d", len(he2.Errs))
	}

	// empty append returns same pointer
	if he.Append() != he {
		t.Fatal("Append() with no args should return receiver")
	}

	// flatten nested
	nested := New("n").Append(New("leaf"))
	he3 := New("p").Append(nested)
	if len(he3.Errs) != 1 || he3.Errs[0].Message != "leaf" {
		t.Fatalf("flatten failed: %+v", he3.Errs)
	}

	// dedup
	he4 := New("p").Append(New("x"), New("x"))
	if len(he4.Errs) != 1 {
		t.Fatalf("dedup failed: %d", len(he4.Errs))
	}

	// share leaf pointer under immutability convention
	leaf := New("shared")
	he5 := New("p").Append(leaf)
	if he5.Errs[0] != leaf {
		t.Fatal("Append should share immutable leaf pointers")
	}

	str := he2.Error()
	if !strings.Contains(str, "errs:(") || !strings.Contains(str, "child1") {
		t.Fatalf("Error() sub list = %q", str)
	}
}

func TestHelpers(t *testing.T) {
	cases := []struct {
		fn     func(string, ...Option) *Error
		status int
		exc    bool
	}{
		{BadRequest, 400, false},
		{Unauthorized, 401, false},
		{Forbidden, 403, false},
		{NotFound, 404, false},
		{RequestTimeout, 408, false},
		{Conflict, 409, false},
		{UnsupportedMediaType, 415, false},
		{UnprocessableEntity, 422, false},
		{TooManyRequests, 429, false},
		{Internal, 500, true},
		{BadGateway, 502, true},
		{ServiceUnavailable, 503, true},
		{GatewayTimeout, 504, true},
	}
	for _, tc := range cases {
		he := tc.fn("m")
		if he.StatusCode != tc.status || he.Exception != tc.exc {
			t.Fatalf("%d: status=%d exception=%v", tc.status, he.StatusCode, he.Exception)
		}
	}
}

func TestToJSONAndPublic(t *testing.T) {
	he := New("my error",
		WithStatus(500),
		WithCode("code-001"),
		WithCategory("cat"),
		WithException(),
		WithExtras(map[string]any{"a": 1, "b": "2"}),
		WithCaller(0),
	)
	if he.File == "" || he.Line == 0 {
		t.Fatal("expected caller info")
	}

	buf, err := he.ToJSON()
	if err != nil {
		t.Fatal(err)
	}
	var full map[string]any
	if err := json.Unmarshal(buf, &full); err != nil {
		t.Fatal(err)
	}
	if full["file"] == nil || full["line"] == nil {
		t.Fatalf("full json missing caller: %s", buf)
	}

	pub, err := he.MarshalPublic()
	if err != nil {
		t.Fatal(err)
	}
	var public map[string]any
	if err := json.Unmarshal(pub, &public); err != nil {
		t.Fatal(err)
	}
	if _, ok := public["file"]; ok {
		t.Fatalf("public json should strip file: %s", pub)
	}
	if public["message"] != "my error" {
		t.Fatalf("message = %v", public["message"])
	}
	// original unchanged
	if he.File == "" {
		t.Fatal("MarshalPublic must not clear original caller")
	}

	// Public() still works and strips caller on the clone
	p := he.Public()
	if p.File != "" || he.File == "" {
		t.Fatal("Public clone should strip caller without touching original")
	}
}

func TestWriteJSON(t *testing.T) {
	he := NotFound("missing", WithCode("nf"), WithCaller(0))
	rec := httptest.NewRecorder()
	if err := he.WriteJSON(rec); err != nil {
		t.Fatal(err)
	}
	if rec.Code != 404 {
		t.Fatalf("status = %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("content-type = %q", ct)
	}
	if strings.Contains(rec.Body.String(), "file") {
		t.Fatalf("body should be public: %s", rec.Body.String())
	}

	var nilErr *Error
	if !errors.Is(nilErr.WriteJSON(httptest.NewRecorder()), ErrNil) {
		t.Fatal("nil WriteJSON should return ErrNil")
	}
}

func TestDefaultCallerAndConverter(t *testing.T) {
	SetDefaultCaller(true)
	SetFileConverter(func(file string) string {
		return strings.ReplaceAll(file, "hes", "X")
	})
	t.Cleanup(func() {
		SetDefaultCaller(false)
		SetFileConverter(nil)
	})

	he := New("test")
	if he.File == "" || he.Line == 0 {
		t.Fatal("default caller should capture")
	}
	if strings.Contains(he.File, "/hes/") {
		t.Fatalf("converter not applied: %s", he.File)
	}
	if !strings.Contains(he.File, "error_test.go") {
		t.Fatalf("file = %s", he.File)
	}
}

func TestStatusOr(t *testing.T) {
	he := &Error{Message: "x"}
	if he.StatusOr(418) != 418 {
		t.Fatal("StatusOr fallback")
	}
	if he.StatusOrInternal() != 500 {
		t.Fatal("StatusOrInternal")
	}
	he.StatusCode = 404
	if he.StatusOrInternal() != 404 {
		t.Fatal("StatusOrInternal should keep status")
	}
}

func TestNilReceivers(t *testing.T) {
	var he *Error
	if he.Error() != "<nil>" {
		t.Fatalf("nil Error() = %q", he.Error())
	}
	if he.Clone() != nil || he.Public() != nil || he.Unwrap() != nil {
		t.Fatal("nil methods")
	}
	if !he.IsEmpty() || he.IsNotEmpty() {
		t.Fatal("nil empty checks")
	}
	buf, err := he.ToJSON()
	if err != nil || string(buf) != "null" {
		t.Fatalf("ToJSON nil = %s, %v", buf, err)
	}
	pub, err := he.MarshalPublic()
	if err != nil || string(pub) != "null" {
		t.Fatalf("MarshalPublic nil = %s, %v", pub, err)
	}
	if fmt.Sprintf("%s", he) != "<nil>" {
		t.Fatal("nil Format")
	}
}

func TestOptionsAndWithCause(t *testing.T) {
	cause := errors.New("root")
	he := New("msg",
		WithTitle("t"),
		WithSubCategory("sub"),
		WithCause(cause),
		WithExtras(nil),
		WithExtras(map[string]any{"x": true}),
		nil, // ignored
	)
	if he.Title != "t" || he.SubCategory != "sub" || he.Cause != cause {
		t.Fatalf("opts not applied: %+v", he)
	}
	if he.Extra["x"] != true {
		t.Fatal("extras")
	}

	next := he.WithCategory("cat2").WithCause(errors.New("other"))
	if he.Category != "" || next.Category != "cat2" {
		t.Fatal("WithCategory")
	}
	if next.Cause.Error() != "other" {
		t.Fatal("WithCause")
	}
}

func TestUnwrapVariants(t *testing.T) {
	a := New("a")
	b := New("b")
	onlySubs := New("p").Append(a, b)
	joined := onlySubs.Unwrap()
	if joined == nil || !errors.Is(joined, a) || !errors.Is(joined, b) {
		t.Fatalf("joined subs: %v", joined)
	}

	single := New("p").Append(a)
	if single.Unwrap() != single.Errs[0] {
		t.Fatal("single sub unwrap")
	}

	cause := errors.New("c")
	both := Wrap(cause).Append(a)
	u := both.Unwrap()
	if !errors.Is(u, cause) || !errors.Is(u, a) {
		t.Fatalf("cause+subs unwrap: %v", u)
	}

	if New("x").Unwrap() != nil {
		t.Fatal("no unwrap")
	}
}

func TestWriteJSONTo(t *testing.T) {
	var buf strings.Builder
	he := New("m", WithCaller(0)).Append(New("c", WithCaller(0)))
	if err := he.WriteJSONTo(&buf); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "file") {
		t.Fatalf("public encode should strip file: %s", buf.String())
	}

	var nilErr *Error
	if !errors.Is(nilErr.WriteJSONTo(&strings.Builder{}), ErrNil) {
		t.Fatal("nil WriteJSONTo should return ErrNil")
	}
}

func TestErrorWithFile(t *testing.T) {
	he := New("m", WithCaller(0))
	if !strings.Contains(he.Error(), "file=") || !strings.Contains(he.Error(), "line=") {
		t.Fatalf("Error() should include caller: %s", he.Error())
	}
}

func TestAppendNilReceiver(t *testing.T) {
	var he *Error
	if he.Append(New("x")) != nil {
		t.Fatal("nil Append")
	}
}
