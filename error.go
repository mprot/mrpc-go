package mrpc

import (
	"context"
	"fmt"
)

// ErrorCode determines the error code of the given error.
func ErrorCode(err error) ErrCode {
	switch err {
	case nil:
		return OK
	case context.DeadlineExceeded:
		return Timeout
	}

	if e, ok := err.(interface{ ErrorCode() ErrCode }); ok {
		return e.ErrorCode()
	}
	return Unknown
}

// Error returns an error with the given code and error text. If code
// is OK, nil will be returned.
func Error(code ErrCode, text string) error {
	if code == OK {
		return nil
	}
	return codeError{code: code, text: text}
}

// Errorf returns an error with the given code and the formatted error
// text. If code is OK, nil will be returned.
func Errorf(code ErrCode, format string, args ...interface{}) error {
	return Error(code, fmt.Sprintf(format, args...))
}

// ResponseError returns the error for the given response.
func ResponseError(resp Response) error {
	return Error(resp.ErrorCode, resp.ErrorText)
}

// ErrorResponse returns a response which indicates the given error.
func ErrorResponse(err error) Response {
	return Response{ErrorCode: ErrorCode(err), ErrorText: err.Error()}
}

func ErrorResponsef(code ErrCode, format string, args ...interface{}) Response {
	return Response{ErrorCode: code, ErrorText: fmt.Sprintf(format, args...)}
}

type codeError struct {
	code ErrCode
	text string
}

func (e codeError) ErrorCode() ErrCode {
	return e.code
}

func (e codeError) Error() string {
	return e.text
}

type optionError string

func (e optionError) Error() string {
	return "invalid option: " + string(e)
}
