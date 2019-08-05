package hes

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"runtime"
)

const (
	defaultStatusCode = http.StatusBadRequest
	idLen             = 6
)

type (
	// Error http error
	Error struct {
		ID         string                 `json:"id,omitempty"`
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

// https://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-go
const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

// RandStringBytes create a rand string
func RandStringBytes(n int) string {
	b := make([]byte, n)
	// A rand.Int63() generates 63 random bits, enough for letterIdxMax letters!
	for i, cache, remain := n-1, rand.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = rand.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
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
		ID:         RandStringBytes(idLen),
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
		ID:         RandStringBytes(idLen),
		Message:    err.Error(),
		StatusCode: statusCode,
		Err:        err,
	}
}

// NewWithCaller create a http error with caller
func NewWithCaller(message string) *Error {
	he := &Error{
		ID:         RandStringBytes(idLen),
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
