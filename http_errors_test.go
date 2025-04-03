package hes

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewHTTPError(t *testing.T) {
	he := New("my error")
	he.Category = "custom"
	he.Code = "1-101"
	he.StatusCode = 400
	t.Run("error", func(t *testing.T) {
		assert := assert.New(t)
		assert.Equal("statusCode=400, category=custom, code=1-101, message=my error", he.Error(), "get error message fail")
	})

	t.Run("format", func(t *testing.T) {
		assert := assert.New(t)
		he := New("my error")
		assert.Equal("statusCode=400, message=my error", fmt.Sprintf("%s", he))
		assert.Equal(`"my error"`, fmt.Sprintf("%q", he))
	})

	t.Run("set caller", func(t *testing.T) {
		assert := assert.New(t)
		he.SetCaller(1)
		assert.NotEmpty(he.File)
		assert.NotEqual(0, he.Line)
	})

	t.Run("new with status code", func(t *testing.T) {
		assert := assert.New(t)
		message := "abc"
		category := "category"
		he := NewWithStatusCode(message, 403, category)
		assert.Equal(message, he.Message)
		assert.Equal(403, he.StatusCode)
		assert.Equal(category, he.Category)
	})

	t.Run("new with error", func(t *testing.T) {
		assert := assert.New(t)
		err := errors.New("abcd")
		he := NewWithError(err)
		assert.Equal(defaultStatusCode, he.StatusCode)
		assert.Equal(err, he.Err)
		assert.Equal(err.Error(), he.Message)
	})

	t.Run("check error", func(t *testing.T) {
		assert := assert.New(t)
		assert.True(IsError(he))
		assert.False(IsError(errors.New("abcd")))
	})

	t.Run("error list", func(t *testing.T) {
		assert := assert.New(t)
		he := New("message")
		he1 := New("messsage1")
		he.Add(he1, nil)
		assert.False(he.IsEmpty())
		assert.True(he.IsNotEmpty())
		assert.Equal("{\"statusCode\":400,\"message\":\"message\",\"errs\":[{\"statusCode\":400,\"message\":\"messsage1\"}]}", string(he.ToJSON()))
		assert.Equal("statusCode=400, message=message, errs:(statusCode=400, message=messsage1)", he.Error())

		he2 := New("message2")
		he2.Add(he)
		assert.Equal("statusCode=400, message=message2, errs:(statusCode=400, message=messsage1)", he2.Error())
	})

	t.Run("add extra", func(t *testing.T) {
		assert := assert.New(t)

		he := New("message")
		he.AddExtra("title", "test")
		assert.Equal("test", he.Extra["title"])
	})
}

func TestMutexError(t *testing.T) {
	assert := assert.New(t)
	he := NewMutex("my error")
	he.SetCaller(1)
	assert.NotEmpty(he.File)
	assert.NotEmpty(he.Line)

	he.Add(errors.New("abc"))

	he1 := he.Clone()
	assert.Equal(he.Message, he1.Message)
}

func TestNewWithCaller(t *testing.T) {
	assert := assert.New(t)
	he := NewWithCaller("my error")
	assert.NotEmpty(he.File)
	assert.NotEqual(0, he.Line)
}

func TestNewWithExcpetion(t *testing.T) {
	assert := assert.New(t)
	he := NewWithException("abc")
	assert.True(he.Exception)
}

func TestEnableCaller(t *testing.T) {
	assert := assert.New(t)
	EnableCaller(true)
	SetFileConvertor(func(file string) string {
		return strings.ReplaceAll(file, "hes", "")
	})
	defer EnableCaller(false)

	he := New("test")
	assert.NotEmpty(he.File)
	assert.False(strings.Contains(he.File, "hes"))
	assert.True(strings.Contains(he.File, "http_errors_test.go"))
	assert.NotEmpty(he.Line)
}

func TestToJSON(t *testing.T) {
	assert := assert.New(t)
	he := NewWithCaller("my error")
	he.Category = "cat"
	he.Code = "code-001"
	he.StatusCode = 500
	he.Exception = true
	he.Err = errors.New("abcd")
	he.Extra = map[string]interface{}{
		"a": 1,
		"b": "2",
	}
	str := fmt.Sprintf(`{"statusCode":500,"code":"code-001","category":"cat","message":"my error","exception":true,"file":"%s","line":%d,"extra":{"a":1,"b":"2"}}`, he.File, he.Line)
	assert.Equal(str, string(he.ToJSON()))
}

func TestClone(t *testing.T) {
	assert := assert.New(t)
	he := NewWithErrorStatusCode(errors.New("abc"), 400)
	heClone := he.CloneWithMessage("def")
	assert.NotEqual(he, heClone)
	assert.NotEqual(he.Message, heClone.Message)
	assert.Equal("def", heClone.Message)

	heClone = he.Clone()
	assert.Equal(he.Message, heClone.Message)
	he.Message = he.Message + "xx"
	assert.NotEqual(he.Message, heClone.Message)
}

func TestABC(t *testing.T) {
	assert := assert.New(t)
	he := &Error{
		Message:  "error message",
		Code:     "cus-validate-fail",
		Category: "common",
	}
	assert.Equal("category=common, code=cus-validate-fail, message=error message", fmt.Sprintf("%s", he))
}

func TestWrap(t *testing.T) {
	assert := assert.New(t)
	he := &Error{
		Message: "error message",
	}
	assert.Equal(he, Wrap(he))

	err := errors.New("abcd")
	he = Wrap(err)

	assert.Equal(err, he.Err)
	assert.Equal(err.Error(), he.Message)
}
