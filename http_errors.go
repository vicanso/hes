package hes

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
)

const (
	defaultStatusCode = http.StatusBadRequest
)

type (
	// Error http error
	Error struct {
		StatusCode int                    `json:"statusCode,omitempty"`
		Code       string                 `json:"code,omitempty"`
		Category   string                 `json:"category,omitempty"`
		Message    string                 `json:"message,omitempty"`
		Exception  bool                   `json:"exception,omitempty"`
		Err        error                  `json:"-"`
		File       string                 `json:"file,omitempty"`
		Line       int                    `json:"line,omitempty"`
		Extra      map[string]interface{} `json:"extra,omitempty"`
	}
)

// Error error interface
func (e *Error) Error() string {
	str := fmt.Sprintf("message=%s", e.Message)

	if e.Code != "" {
		str = fmt.Sprintf("code=%s, %s", e.Code, str)
	}

	if e.Category != "" {
		str = fmt.Sprintf("category=%s, %s", e.Category, str)
	}
	return str
}

// Format error format
func (e *Error) Format(s fmt.State, verb rune) {
	switch verb {
	default:
		fallthrough
	case 's':
		io.WriteString(s, e.Error())
	case 'q':
		fmt.Fprintf(s, "%q", e.Message)
	}
}

// SetCaller set info of caller
func (e *Error) SetCaller(skip int) {
	_, file, line, _ := runtime.Caller(skip)
	e.File = file
	e.Line = line
}

// ToJSON error to json
func (e *Error) ToJSON() []byte {
	buf, _ := json.Marshal(e)
	return buf
}

// New create a http error
func New(message string) *Error {
	return NewWithStatusCode(message, defaultStatusCode)
}

// NewWithStatusCode create a http error with status code
func NewWithStatusCode(message string, statusCode int) *Error {
	return &Error{
		Message:    message,
		StatusCode: statusCode,
	}
}

// NewWithError create a http error with error
func NewWithError(err error) *Error {
	return NewWithErrorStatusCode(err, defaultStatusCode)
}

// NewWithErrorStatusCode create a http error with error and status code
func NewWithErrorStatusCode(err error, statusCode int) *Error {
	return &Error{
		Message:    err.Error(),
		StatusCode: statusCode,
		Err:        err,
	}
}

// NewWithCaller create a http error with caller
func NewWithCaller(message string) *Error {
	he := &Error{
		Message:    message,
		StatusCode: http.StatusBadRequest,
	}
	he.SetCaller(1)
	return he
}
