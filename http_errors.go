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
		Errs       []*Error               `json:"errs,omitempty"`
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
		_, _ = io.WriteString(s, e.Error())
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

// CloneWithMessage clone error and update message
func (e *Error) CloneWithMessage(message string) *Error {
	clone := *e
	clone.Message = message
	return &clone
}

// IsEmpty check the error list is empty
func (e *Error) IsEmpty() bool {
	return len(e.Errs) == 0
}

// Add add error to error list
func (e *Error) Add(err *Error) {
	if len(e.Errs) == 0 {
		e.Errs = make([]*Error, 0)
	}
	e.Errs = append(e.Errs, err)
}

// New create a http error
func New(message string) *Error {
	return NewWithStatusCode(message, defaultStatusCode)
}

// NewWithStatusCode create a http error with status code
func NewWithStatusCode(message string, statusCode int, category ...string) *Error {

	he := &Error{
		Message:    message,
		StatusCode: statusCode,
	}
	if len(category) != 0 {
		he.Category = category[0]
	}
	return he
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

// IsError check the error whether or not hes error
func IsError(err error) bool {
	_, ok := err.(*Error)
	return ok
}

// Wrap wrap error
func Wrap(err error) *Error {
	he, ok := err.(*Error)
	if ok {
		return he
	}
	return NewWithError(err)
}
