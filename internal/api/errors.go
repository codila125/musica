package api

import (
	"errors"
	"fmt"
)

type ErrorKind string

const (
	ErrorKindAuth    ErrorKind = "auth"
	ErrorKindNetwork ErrorKind = "network"
	ErrorKindConfig  ErrorKind = "config"
	ErrorKindUnknown ErrorKind = "unknown"
)

type Error struct {
	Kind ErrorKind
	Op   string
	Err  error
}

func (e *Error) Error() string {
	if e.Op == "" {
		return fmt.Sprintf("%s: %v", e.Kind, e.Err)
	}
	return fmt.Sprintf("%s %s: %v", e.Op, e.Kind, e.Err)
}

func (e *Error) Unwrap() error { return e.Err }

func Wrap(kind ErrorKind, op string, err error) error {
	if err == nil {
		return nil
	}
	return &Error{Kind: kind, Op: op, Err: err}
}

func KindOf(err error) ErrorKind {
	if err == nil {
		return ErrorKindUnknown
	}
	var ae *Error
	if ok := errors.As(err, &ae); ok {
		return ae.Kind
	}
	return ErrorKindUnknown
}
