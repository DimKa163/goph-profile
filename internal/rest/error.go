package rest

import (
	"errors"
	"net/http"

	"github.com/DimKa163/goph-profile/internal/entity"
	"github.com/labstack/echo/v4"
)

type (
	// ServiceError is an API error response.
	ServiceError struct {
		// Message stores the message value.
		Message string `json:"message"`
		// Code stores the code value.
		Code string `json:"code"`
	}
)

// Error returns the error message.
func Error(c echo.Context, err error) error {
	if errors.Is(err, http.ErrMissingFile) {
		return c.JSON(http.StatusBadRequest, ServiceError{
			Message: err.Error(),
		})
	}

	pe, ok := errors.AsType[*entity.ProfileError](err)
	if ok {
		switch pe.Code {
		case entity.InvalidAvatarIDErrorCode:
			return c.JSON(http.StatusBadRequest, ServiceError{
				Message: pe.Message,
				Code:    pe.Code.String(),
			})
		case entity.InvalidUserIDErrorCode:
			return c.JSON(http.StatusBadRequest, ServiceError{
				Message: pe.Message,
				Code:    pe.Code.String(),
			})
		case entity.NotFoundEntityErrorCode:
			return c.JSON(http.StatusNotFound, ServiceError{
				Message: pe.Message,
				Code:    pe.Code.String(),
			})
		case entity.InvalidSizeErrorCode:
			return c.JSON(http.StatusBadRequest, ServiceError{
				Message: pe.Message,
				Code:    pe.Code.String(),
			})
		case entity.InvalidContentTypeErrorCode:
			return c.JSON(http.StatusBadRequest, ServiceError{
				Message: pe.Message,
				Code:    pe.Code.String(),
			})
		case entity.PermissionDeniedErrorCode:
			return c.JSON(http.StatusForbidden, ServiceError{
				Message: pe.Message,
				Code:    pe.Code.String(),
			})
		default:
			return c.JSON(http.StatusInternalServerError, ServiceError{
				Code: entity.InternalErrorCode.String(),
			})
		}
	}
	return c.JSON(http.StatusInternalServerError, ServiceError{
		Code: entity.InternalErrorCode.String(),
	})
}
