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
	Dimensions *Span   `json:"type,omitempty"`
	Trace      *Caller `json:"trace,omitempty"`
	Message    *string `json:"message,omitempty"`
	Error      error   `json:"error,omitempty"`
}

func NewError(dimension *Span, message string, err error) *Error {
	trace := NewCaller(2)
	if err == nil {
		return &Error{
			Items: []*ErrorItem{
				{
					Dimensions: dimension,
					Trace:      trace,
					Message:    &message,
					Error:      nil,
				},
			},
		}
	}

	var e *Error
	if errors.As(err, &e) {
		e.Items = append(e.Items, &ErrorItem{
			Dimensions: dimension,
			Trace:      trace,
			Message:    &message,
			Error:      nil,
		})
		return e
	}

	return &Error{
		Items: []*ErrorItem{
			{
				Dimensions: dimension,
				Trace:      trace,
				Message:    &message,
				Error:      err,
			},
		},
	}
}
