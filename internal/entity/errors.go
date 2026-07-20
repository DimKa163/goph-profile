package entity

import (
	"errors"
	"fmt"
	"strings"
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

var ErrInternalError = errors.New("internal error")

var ErrInvalidAvatarID = errors.New("invalid avatar id")

var ErrInvalidUserID = errors.New("invalid user id")

var ErrInvalidSize = errors.New("invalid size")

var ErrInvalidContentErrorMessage = errors.New("invalid content type")

var ErrNotFoundEntity = errors.New("entity not found")

var ErrPermissionDenied = errors.New("permission denied")

type ProfileError struct {
	Cause   error
	Kind    error
	Code    ErrorCode
	Message string
}

func newProfileError(cause error, kind error, code ErrorCode, message string) *ProfileError {
	return &ProfileError{
		Cause:   cause,
		Kind:    kind,
		Code:    code,
		Message: message,
	}
}

func (pe *ProfileError) Error() string {
	if pe == nil {
		return "<nil>"
	}

	parts := make([]string, 0, 3)

	if pe.Kind != nil {
		parts = append(parts, pe.Kind.Error())
	}

	if pe.Message != "" {
		parts = append(parts, pe.Message)
	}

	if pe.Cause != nil {
		parts = append(parts, pe.Cause.Error())
	}

	return strings.Join(parts, ": ")
}

func (pe *ProfileError) Unwrap() []error {
	if pe == nil {
		return nil
	}

	errs := make([]error, 0, 2)

	if pe.Cause != nil {
		errs = append(errs, pe.Cause)
	}

	if pe.Kind != nil {
		errs = append(errs, pe.Kind)
	}

	return errs
}

func WrapError(code ErrorCode, args any, err error) error {
	switch code {
	case InternalErrorCode:
		return newProfileError(err, ErrInternalError, code, fmt.Sprintf("%v", args))
	case InvalidAvatarIDErrorCode:
		return newProfileError(err, ErrInvalidAvatarID, code, fmt.Sprintf("%v", args))
	case InvalidUserIDErrorCode:
		return newProfileError(err, ErrInvalidUserID, code, fmt.Sprintf("%v", args))
	case InvalidContentTypeErrorCode:
		return newProfileError(err, ErrInvalidContentErrorMessage, code, fmt.Sprintf("not supported content type: %s", args))
	case InvalidSizeErrorCode:
		return newProfileError(err, ErrInvalidSize, code, fmt.Sprintf("invalid size: %s", args))
	case NotFoundEntityErrorCode:
		return newProfileError(err, ErrNotFoundEntity, code, fmt.Sprintf("entity not found: %s", args))
	case PermissionDeniedErrorCode:
		return newProfileError(err, ErrPermissionDenied, code, fmt.Sprintf("permission denied: %s", args))
	default:
		panic(fmt.Sprintf("unexpected error code: %d", code))
	}
}
