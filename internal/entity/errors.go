package entity

import "errors"

type ErrorCode int

const (
	_ ErrorCode = iota
	InvalidContentErrorCode
	TooBigSizeErrorCode
)

var ErrUnknownError = errors.New("unknown error")
var ErrInvalidContentErrorMessage = errors.New("invalid content type")
var ErrTooBigSizeErrorMessage = errors.New("too big size")

func Error(code ErrorCode) error {
	switch code {
	case InvalidContentErrorCode:
		return ErrInvalidContentErrorMessage
	case TooBigSizeErrorCode:
		return ErrTooBigSizeErrorMessage
	default:
		return ErrUnknownError
	}
}
