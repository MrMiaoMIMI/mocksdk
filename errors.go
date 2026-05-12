package mocksdk

import (
	"errors"
	"fmt"
)

type ErrorKind string

const (
	ErrorKindConfig    ErrorKind = "config"
	ErrorKindTransport ErrorKind = "transport"
	ErrorKindServer    ErrorKind = "server"
	ErrorKindDecode    ErrorKind = "decode"
)

type Error struct {
	Kind       ErrorKind
	StatusCode int
	Message    string
	Err        error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.StatusCode != 0 {
		return fmt.Sprintf("%s error: status=%d message=%s", e.Kind, e.StatusCode, e.Message)
	}
	return fmt.Sprintf("%s error: %s", e.Kind, e.Message)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func IsErrorKind(err error, kind ErrorKind) bool {
	var sdkErr *Error
	return errors.As(err, &sdkErr) && sdkErr.Kind == kind
}

func newError(kind ErrorKind, message string, err error) *Error {
	return &Error{Kind: kind, Message: message, Err: err}
}

func newStatusError(kind ErrorKind, statusCode int, message string) *Error {
	return &Error{Kind: kind, StatusCode: statusCode, Message: message}
}
