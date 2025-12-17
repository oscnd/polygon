package flow

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
	Dimensions *Dimension `json:"type,omitempty"`
	Trace      *Trace     `json:"trace,omitempty"`
	Message    *string    `json:"message,omitempty"`
	Error      error      `json:"error,omitempty"`
}

func NewError(dimension *Dimension, message string, err error) *Error {
	trace := NewTrace(2)
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
