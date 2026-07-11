package hes

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// ErrNil is returned when a method is called on a nil *Error receiver
// that cannot produce a meaningful HTTP response.
var ErrNil = errors.New("hes: nil error")

// publicDTO is the client-facing JSON shape (caller File/Line omitted).
// Built without cloning the live *Error tree.
type publicDTO struct {
	StatusCode  int            `json:"statusCode,omitempty"`
	Code        string         `json:"code,omitempty"`
	Category    string         `json:"category,omitempty"`
	SubCategory string         `json:"subCategory,omitempty"`
	Title       string         `json:"title,omitempty"`
	Message     string         `json:"message,omitempty"`
	Exception   bool           `json:"exception,omitempty"`
	Extra       map[string]any `json:"extra,omitempty"`
	Errs        []publicDTO    `json:"errs,omitempty"`
}

func (e *Error) toPublicDTO() publicDTO {
	if e == nil {
		return publicDTO{}
	}
	p := publicDTO{
		StatusCode:  e.StatusCode,
		Code:        e.Code,
		Category:    e.Category,
		SubCategory: e.SubCategory,
		Title:       e.Title,
		Message:     e.Message,
		Exception:   e.Exception,
		Extra:       e.Extra,
	}
	if len(e.Errs) > 0 {
		p.Errs = make([]publicDTO, len(e.Errs))
		for i, sub := range e.Errs {
			p.Errs[i] = sub.toPublicDTO()
		}
	}
	return p
}

// Public returns a deep clone safe for client responses:
// caller File/Line are stripped from the error and all sub-errors.
//
// Prefer MarshalPublic / WriteJSON when you only need JSON output;
// those paths avoid a full tree clone.
func (e *Error) Public() *Error {
	if e == nil {
		return nil
	}
	clone := e.Clone()
	stripCaller(clone)
	return clone
}

func stripCaller(e *Error) {
	if e == nil {
		return
	}
	e.File = ""
	e.Line = 0
	for _, sub := range e.Errs {
		stripCaller(sub)
	}
}

// ToJSON marshals the full error (including caller info when present).
func (e *Error) ToJSON() ([]byte, error) {
	if e == nil {
		return []byte("null"), nil
	}
	return json.Marshal(e)
}

// MarshalPublic marshals the client-facing view (no file/line) without cloning.
func (e *Error) MarshalPublic() ([]byte, error) {
	if e == nil {
		return []byte("null"), nil
	}
	return json.Marshal(e.toPublicDTO())
}

// WriteJSON writes the public JSON body and status to w.
// A nil receiver returns ErrNil and writes nothing.
func (e *Error) WriteJSON(w http.ResponseWriter) error {
	if e == nil {
		return ErrNil
	}
	body, err := e.MarshalPublic()
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(e.StatusOrInternal())
	_, err = w.Write(body)
	return err
}

// WriteJSONTo marshals the public error to any writer without a full clone.
// A nil receiver returns ErrNil and writes nothing.
func (e *Error) WriteJSONTo(w io.Writer) error {
	if e == nil {
		return ErrNil
	}
	return json.NewEncoder(w).Encode(e.toPublicDTO())
}
