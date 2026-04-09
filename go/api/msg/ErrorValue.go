// Package msg with the response error value
package msg

import (
	"errors"
)

// Error response payload
type ErrorValue struct {
	// Status code: https://w3c.github.io/wot-profile/#error-responses
	Status int `json:"status"`
	// Type is a URI reference [RFC3986] that identifies the problem type.
	Type string `json:"type"`
	// Title contains a short, human-readable summary of the problem type
	Title string `json:"title"`
	// Detail a human-readable explanation
	Detail string `json:"detail,omitempty"`
}

func (e *ErrorValue) String() string {
	return e.Title
}

// AsError returns an error instance or nil if no error is contained
func (e *ErrorValue) AsError() error {
	if e.Title == "" && e.Status == 0 {
		return nil
	}
	return errors.New(e.String())
}

// Create an ErrorValue object from the given error. This returns nil if err is nil.
func ErrorValueFromError(err error) *ErrorValue {
	if err == nil {
		return nil
	}
	return &ErrorValue{
		Status: 400, // bad request
		Type:   "Bad request",
		Title:  err.Error(),
		// Detail: "",
	}
}
