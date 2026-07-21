package entity

import (
	"errors"
	"fmt"
	"strings"
)

// ErrorCode identifies a profile error category.
type ErrorCode int

const (
	// InternalErrorCode indicates an internal error.
	InternalErrorCode ErrorCode = iota
	// InvalidAvatarIDErrorCode indicates an invalid avatar ID.
	InvalidAvatarIDErrorCode
	// InvalidUserIDErrorCode indicates an invalid user ID.
	InvalidUserIDErrorCode
	// InvalidContentTypeErrorCode indicates an unsupported content type.
	InvalidContentTypeErrorCode
	// InvalidSizeErrorCode indicates an invalid size.
	InvalidSizeErrorCode
	// NotFoundEntityErrorCode indicates that an entity was not found.
	NotFoundEntityErrorCode

	// PermissionDeniedErrorCode indicates a permission denied error.
	PermissionDeniedErrorCode
)

// String returns the string representation of the error code.
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

// ErrInternalError is the sentinel error for internal failures.
var ErrInternalError = errors.New("internal error")

// ErrInvalidAvatarID is the sentinel error for invalid avatar IDs.
var ErrInvalidAvatarID = errors.New("invalid avatar id")

// ErrInvalidUserID is the sentinel error for invalid user IDs.
var ErrInvalidUserID = errors.New("invalid user id")

// ErrInvalidSize is the sentinel error for invalid sizes.
var ErrInvalidSize = errors.New("invalid size")

// ErrInvalidContentErrorMessage is the sentinel error for unsupported content types.
var ErrInvalidContentErrorMessage = errors.New("invalid content type")

// ErrNotFoundEntity is the sentinel error for missing entities.
var ErrNotFoundEntity = errors.New("entity not found")

// ErrPermissionDenied is the sentinel error for permission denied failures.
var ErrPermissionDenied = errors.New("permission denied")

// ProfileError describes a profile domain error with an optional cause.
type ProfileError struct {
	// Cause stores the cause value.
	Cause error
	// Kind stores the kind value.
	Kind error
	// Code stores the code value.
	Code ErrorCode
	// Message stores the message value.
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

// Error returns the profile error message.
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

// Unwrap returns the underlying cause and kind errors.
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

// WrapError wraps err into a ProfileError for the provided error code.
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
