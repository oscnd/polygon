package span

import (
	"errors"
)

type Error struct {
	Items []*ErrorItem `json:"items,omitempty"`
}

func (r *Error) Error() string {
	return r.Items[len(r.Items)-1].Error.Error()
}

type ErrorItem struct {
	Span    *Span   `json:"type,omitempty"`
	Trace   *Caller `json:"trace,omitempty"`
	Message *string `json:"message,omitempty"`
	Error   error   `json:"error,omitempty"`
}

func NewError(span *Span, message string, err error) error {
	trace := NewCaller(2)
	if err == nil {
		return &Error{
			Items: []*ErrorItem{
				{
					Span:    span,
					Trace:   trace,
					Message: &message,
					Error:   nil,
				},
			},
		}
	}

	var e *Error
	if errors.As(err, &e) {
		e.Items = append(e.Items, &ErrorItem{
			Span:    span,
			Trace:   trace,
			Message: &message,
			Error:   nil,
		})
		return e
	}

	return &Error{
		Items: []*ErrorItem{
			{
				Span:    span,
				Trace:   trace,
				Message: &message,
				Error:   err,
			},
		},
	}
}
