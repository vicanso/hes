package hes

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
)

const (
	defaultStatusCode = http.StatusBadRequest
)

var callerEnabled = false

type (
	// Error http error
	Error struct {
		StatusCode int    `json:"statusCode,omitempty"`
		Code       string `json:"code,omitempty"`
		Category   string `json:"category,omitempty"`
		Title      string `json:"title,omitempty"`
		Message    string `json:"message,omitempty"`
		Exception  bool   `json:"exception,omitempty"`
		Err        error  `json:"-"`
		// File caller file
		File string `json:"file,omitempty"`
		// Line caller line
		Line  int                    `json:"line,omitempty"`
		Extra map[string]interface{} `json:"extra,omitempty"`
		Errs  []*Error               `json:"errs,omitempty"`
	}
	FileConvertor func(file string) string
)

// EnableCaller enable caller
func EnableCaller(enabled bool) {
	callerEnabled = enabled
}

// fileConvertor file convertor
var fileConvertor FileConvertor

// SetFileConvertor set file convertor
func SetFileConvertor(fn FileConvertor) {
	fileConvertor = fn
}

// Error error interface
func (e *Error) Error() string {
	str := fmt.Sprintf("message=%s", e.Message)

	if e.Code != "" {
		str = fmt.Sprintf("code=%s, %s", e.Code, str)
	}

	if e.Category != "" {
		str = fmt.Sprintf("category=%s, %s", e.Category, str)
	}

	if e.StatusCode != 0 {
		str = fmt.Sprintf("statusCode=%d, %s", e.StatusCode, str)
	}

	if e.File != "" {
		str = fmt.Sprintf("file=%s, line=%d, %s", e.File, e.Line, str)
	}
	if len(e.Errs) != 0 {
		arr := make([]string, len(e.Errs))
		for index, err := range e.Errs {
			arr[index] = err.Error()
		}
		str = fmt.Sprintf("%s, errs:(%s)", str, strings.Join(arr, ","))
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
	if fileConvertor != nil {
		file = fileConvertor(file)
	}
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
	clone := e.Clone()
	clone.Message = message
	return clone
}

// IsEmpty check the error list is empty
func (e *Error) IsEmpty() bool {
	return len(e.Errs) == 0
}

// IsNotEmpty check the error list is not empty
func (e *Error) IsNotEmpty() bool {
	return !e.IsEmpty()
}

// Add add error to error list
func (e *Error) Add(errs ...error) {
	if len(errs) == 0 {
		return
	}
	if len(e.Errs) == 0 {
		e.Errs = make([]*Error, 0)
	}
	for _, err := range errs {
		he := Wrap(err)
		// 如果包括子错误，则直接添加子错误列表
		if he.IsNotEmpty() {
			for _, err := range he.Errs {
				e.Add(err)
			}
			continue
		}

		e.Errs = append(e.Errs, he)
	}
}

// Clone clone error
func (e *Error) Clone() *Error {
	he := new(Error)
	*he = *e
	return he
}

func newError(message string, statusCode, skip int, category ...string) *Error {
	he := &Error{
		Message:    message,
		StatusCode: statusCode,
	}
	if len(category) != 0 {
		he.Category = category[0]
	}
	if callerEnabled {
		he.SetCaller(skip)
	}
	return he
}

// New create a http error
func New(message string, category ...string) *Error {
	he := newError(message, defaultStatusCode, 3, category...)
	return he
}

// NewWithStatusCode create a http error with status code
func NewWithStatusCode(message string, statusCode int, category ...string) *Error {
	he := newError(message, statusCode, 3, category...)
	return he
}

// NewWithError create a http error with error
func NewWithError(err error) *Error {
	he := newError(err.Error(), defaultStatusCode, 3)
	he.Err = err
	return he
}

// NewWithErrorStatusCode create a http error with error and status code
func NewWithErrorStatusCode(err error, statusCode int) *Error {
	he := newError(err.Error(), statusCode, 3)
	he.Err = err
	return he
}

// NewWithCaller create a http error with caller
func NewWithCaller(message string) *Error {
	he := &Error{
		Message:    message,
		StatusCode: defaultStatusCode,
	}
	he.SetCaller(2)
	return he
}

// NewWithExcpetion create a http error and set exception to true
func NewWithExcpetion(message string) *Error {
	he := newError(message, defaultStatusCode, 3)
	he.Exception = true
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
		return he.Clone()
	}
	he = newError(err.Error(), defaultStatusCode, 3)
	he.Err = err
	return he
}
