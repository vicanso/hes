package hes

import (
	"errors"
	"fmt"
	"testing"
)

func TestNewHTTPError(t *testing.T) {
	he := New("my error")
	he.Category = "custom"
	he.Code = "1-101"
	he.StatusCode = 400
	t.Run("error", func(t *testing.T) {
		if he.Error() != "category=custom, code=1-101, message=my error" {
			t.Fatalf("get error message fail")
		}
	})

	t.Run("format", func(t *testing.T) {
		he := New("my error")
		if fmt.Sprintf("%s", he) != "message=my error" {
			t.Fatalf("format s fail")
		}
		if fmt.Sprintf("%q", he) != `"my error"` {
			t.Fatalf("format q fail")
		}
	})

	t.Run("set caller", func(t *testing.T) {
		he.SetCaller(1)
		if he.File == "" ||
			he.Line == 0 {
			t.Fatalf("set caller fail")
		}
	})

	t.Run("new with status code", func(t *testing.T) {
		message := "abc"
		he := NewWithStatusCode(message, 403)
		if he.Message != message ||
			he.StatusCode != 403 {
			t.Fatalf("new with status code fail")
		}
	})

	t.Run("new with error", func(t *testing.T) {
		err := errors.New("abcd")
		he := NewWithError(err)
		if he.StatusCode != defaultStatusCode ||
			he.Err != err ||
			he.Message != err.Error() {
			t.Fatalf("new with error fail")
		}
	})
}

func TestNewWithCaller(t *testing.T) {
	he := NewWithCaller("my error")
	if he.File == "" ||
		he.Line == 0 {
		t.Fatalf("new with caller fail")
	}
}

func TestToJSON(t *testing.T) {
	he := NewWithCaller("my error")
	he.Category = "cat"
	he.Code = "code-001"
	he.StatusCode = 500
	he.Exception = true
	he.Extra = map[string]interface{}{
		"a": 1,
		"b": "2",
	}
	str := fmt.Sprintf(`{"statusCode":500,"code":"code-001","category":"cat","message":"my error","exception":true,"file":"%s","line":%d,"extra":{"a":1,"b":"2"}}`, he.File, he.Line)
	if string(he.ToJSON()) != str {
		t.Fatalf("to json fail")
	}
}

func TestABC(t *testing.T) {
	he := &Error{
		Message:  "error message",
		Code:     "cus-validate-fail",
		Category: "common",
	}
	if fmt.Sprintf("%s", he) != "category=common, code=cus-validate-fail, message=error message" {
		t.Fatalf("format fail")
	}
}

func TestWrap(t *testing.T) {
	he := &Error{
		Message: "error message",
	}
	if Wrap(he) != he {
		t.Fatalf("wrap http error fail")
	}

	err := errors.New("abcd")
	he = Wrap(err)
	if he.Err != err || he.Message != err.Error() {
		t.Fatalf("wrap original error fail")
	}
}
