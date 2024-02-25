package hes

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"sync"
)

const (
	defaultStatusCode = http.StatusBadRequest
)

var callerEnabled = false

type (
	// Error http error
	Error struct {
		lock *sync.RWMutex
		// http status code
		StatusCode int `json:"statusCode,omitempty"`
		// error code
		Code string `json:"code,omitempty"`
		// category
		Category string `json:"category,omitempty"`
		// sub category
		SubCategory string `json:"subCategory,omitempty"`
		// title
		Title string `json:"title,omitempty"`
		// message
		Message string `json:"message,omitempty"`
		// exception error
		Exception bool `json:"exception,omitempty"`
		// original error
		Err error `json:"-"`
		// File caller file
		File string `json:"file,omitempty"`
		// Line caller line
		Line int `json:"line,omitempty"`
		// extra info for error
		Extra map[string]interface{} `json:"extra,omitempty"`
		// sub errors
		Errs []*Error `json:"errs,omitempty"`
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
	if e.lock != nil {
		e.lock.Lock()
		defer e.lock.Unlock()
	}
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
	if e.lock != nil {
		e.lock.RLock()
		defer e.lock.RUnlock()
	}
	return len(e.Errs) == 0
}

// IsNotEmpty check the error list is not empty
func (e *Error) IsNotEmpty() bool {
	// is empty有判断锁，因此不需要判断
	return !e.IsEmpty()
}

// AddExtra add extra value to error
func (e *Error) AddExtra(key string, value interface{}) {
	if e.lock != nil {
		e.lock.Lock()
		defer e.lock.Unlock()
	}
	if e.Extra == nil {
		e.Extra = make(map[string]interface{})
	}
	e.Extra[key] = value
}

// Exists return true if it already exists
func (e *Error) exists(he *Error) bool {
	for _, item := range e.Errs {
		if he.Title == item.Title &&
			he.Message == item.Message &&
			he.Category == item.Category {
			return true
		}
	}
	return false
}
func (e *Error) add(errs ...error) {
	if len(e.Errs) == 0 {
		e.Errs = make([]*Error, 0)
	}
	for _, err := range errs {
		if err == nil {
			continue
		}
		he := Wrap(err)
		// 如果包括子错误，则直接添加子错误列表
		if he.IsNotEmpty() {
			for _, err := range he.Errs {
				e.add(err)
			}
			continue
		}
		// 判断是否已存在相同的error
		// 如果已有相同error，则不添加
		if e.exists(he) {
			continue
		}

		e.Errs = append(e.Errs, he)
	}
}

// Add add error to error list
func (e *Error) Add(errs ...error) {
	if len(errs) == 0 {
		return
	}
	if e.lock != nil {
		e.lock.Lock()
		defer e.lock.Unlock()
	}
	e.add(errs...)
}

// Clone clone error
func (e *Error) Clone() *Error {
	he := new(Error)
	*he = *e
	if he.lock != nil {
		he.lock = &sync.RWMutex{}
	}
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

// NewMutex create a http error with mutex
func NewMutex(message string, category ...string) *Error {
	he := New(message, category...)
	he.lock = &sync.RWMutex{}
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

// NewWithException create a http error and set exception to true
func NewWithException(message string) *Error {
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
