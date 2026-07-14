package entity

import (
	"errors"
	"fmt"
)

type ErrorCode int

const (
	InternalErrorCode ErrorCode = iota
	InvalidAvatarIDErrorCode
	InvalidUserIDErrorCode
	InvalidContentTypeErrorCode
	InvalidSizeErrorCode
	NotFoundEntityErrorCode
	PermissionDeniedErrorCode
)

func (e ErrorCode) String() string {
	return [...]string{
		"internal",
		"invalid_avatar_id",
		"invalid_user_id",
		"invalid_content_type",
		"invalid_size",
		"not_found",
		"permission_denied",
	}[e]
}

var ErrUnknownError = errors.New("unknown error")

var ErrInvalidAvatarID = errors.New("invalid avatar id")

var ErrInvalidUserID = errors.New("invalid user id")

var ErrInvalidSize = errors.New("invalid size")

var ErrInvalidContentErrorMessage = errors.New("invalid content type")

var ErrNotFoundEntity = errors.New("entity not found")

var ErrPermissionDenied = errors.New("permission denied")

type ProfileError struct {
	inner   error
	Code    ErrorCode
	Message string
}

func newProfileError(inner error, code ErrorCode, message string) *ProfileError {
	return &ProfileError{
		inner:   inner,
		Code:    code,
		Message: message,
	}
}

func (pe *ProfileError) Error() string {
	return pe.Message
}

func (pe *ProfileError) Is(err error) bool {
	return errors.Is(err, pe.inner)
}

func (pe *ProfileError) Unwrap() error {
	return pe.inner
}

func Error(code ErrorCode, args any, errs ...error) error {
	switch code {
	case InternalErrorCode:
		return newProfileError(errors.Join(append(errs, ErrUnknownError)...), code, fmt.Sprintf("%v", args))
	case InvalidAvatarIDErrorCode:
		return newProfileError(errors.Join(append(errs, ErrInvalidAvatarID)...), code, fmt.Sprintf("%v", args))
	case InvalidUserIDErrorCode:
		return newProfileError(errors.Join(append(errs, ErrInvalidUserID)...), code, fmt.Sprintf("%v", args))
	case InvalidContentTypeErrorCode:
		return newProfileError(errors.Join(append(errs, ErrInvalidContentErrorMessage)...), code, fmt.Sprintf("not supported content type: %s", args))
	case InvalidSizeErrorCode:
		return newProfileError(errors.Join(append(errs, ErrInvalidSize)...), code, fmt.Sprintf("invalid size: %s", args))
	case NotFoundEntityErrorCode:
		return newProfileError(errors.Join(append(errs, ErrNotFoundEntity)...), code, fmt.Sprintf("entity not found: %s", args))
	case PermissionDeniedErrorCode:
		return newProfileError(errors.Join(append(errs, ErrPermissionDenied)...), code, fmt.Sprintf("permission denied: %s", args))
	default:
		panic(fmt.Sprintf("unexpected error code: %d", code))
	}
}
