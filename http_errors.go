package errors

import (
	"encoding/json"
	"fmt"
	"io"
	"runtime"
)

type (
	// HTTPError http error
	HTTPError struct {
		StatusCode int    `json:"statusCode,omitempty"`
		Code       string `json:"code,omitempty"`
		Category   string `json:"category,omitempty"`
		Message    string `json:"message,omitempty"`
		Exception  bool   `json:"exception,omitempty"`

		File  string                 `json:"file,omitempty"`
		Line  int                    `json:"line,omitempty"`
		Extra map[string]interface{} `json:"extra,omitempty"`
	}
)

// Error error interface
func (e *HTTPError) Error() string {
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
func (e *HTTPError) Format(s fmt.State, verb rune) {
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
func (e *HTTPError) SetCaller(skip int) {
	_, file, line, _ := runtime.Caller(skip)
	e.File = file
	e.Line = line
}

// ToJSON error to json
func (e *HTTPError) ToJSON() []byte {
	buf, _ := json.Marshal(e)
	return buf
}

// New create a http error
func New(message string) *HTTPError {
	return &HTTPError{
		Message: message,
	}
}

// NewWithCaller create a http error with caller
func NewWithCaller(message string) *HTTPError {
	he := &HTTPError{
		Message: message,
	}
	he.SetCaller(1)
	return he
}
