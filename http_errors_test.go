package hes

import (
	"errors"
	"fmt"
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
		assert.Equal("category=custom, code=1-101, message=my error", he.Error(), "get error message fail")
	})

	t.Run("format", func(t *testing.T) {
		assert := assert.New(t)
		he := New("my error")
		assert.Equal("message=my error", fmt.Sprintf("%s", he))
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
}

func TestNewWithCaller(t *testing.T) {
	assert := assert.New(t)
	he := NewWithCaller("my error")
	assert.NotEmpty(he.File)
	assert.NotEqual(0, he.Line)
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
	str := fmt.Sprintf(`{"id":"%s","statusCode":500,"code":"code-001","category":"cat","message":"my error","exception":true,"file":"%s","line":%d,"extra":{"a":1,"b":"2"}}`, he.ID, he.File, he.Line)
	assert.Equal(str, string(he.ToJSON()))
}

func TestClone(t *testing.T) {
	assert := assert.New(t)
	he := NewWithErrorStatusCode(errors.New("abc"), 400)
	heClone := he.CloneWithMessage("def")
	assert.NotEqual(he, heClone)
	assert.Equal(he.ID, heClone.ID)
	assert.NotEqual(he.Message, heClone.Message)
	assert.Equal("def", heClone.Message)
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
