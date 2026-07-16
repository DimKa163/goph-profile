package rest

import (
	"errors"
	"net/http"

	"github.com/DimKa163/goph-profile/internal/entity"
	"github.com/DimKa163/goph-profile/internal/logging"
	"github.com/DimKa163/goph-profile/internal/usecase"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type userController struct {
	userService *usecase.UserService
}

func NewUserController(userServices *usecase.UserService) *userController {
	return &userController{
		userService: userServices,
	}
}

func (u *userController) Register(c Section) {
	c.GET("/users/:userId/avatar", u.Avatar)
	c.GET("/users/:userId/avatars", u.Avatars)
	c.DELETE("/users/:userId/avatar", u.Delete)
}

func (u *userController) Avatar(c echo.Context) error {
	logger := logging.Logger(c.Request().Context())
	userID, err := entity.ParseEmail(c.Param("userId"))
	if err != nil {
		logger.Error("error parsing user id", zap.Error(err))
		return Error(c, err)
	}

	req := usecase.Request{
		Format: c.QueryParam("format"),
		Size:   entity.Size(c.QueryParam("size")),
	}

	e, buf, err := u.userService.Get(c.Request().Context(), c.Request().Header.Get("If-None-Match"), userID, &req)
	switch {
	case err == nil:
	case errors.Is(err, usecase.ErrAvatarNotModified):
		return c.NoContent(http.StatusNotModified)
	case errors.Is(err, entity.ErrNotFoundEntity):
		e, buf, err = u.userService.GetDefault(c.Request().Header.Get("If-None-Match"), &req)
		if err != nil {
			if errors.Is(err, usecase.ErrAvatarNotModified) {
				return c.NoContent(http.StatusNotModified)
			}
			logger.Error("error with getting default avatar", zap.Error(err))
			return Error(c, err)
		}
	default:
		logger.Error("error with getting user avatar", zap.Error(err))
		return Error(c, err)
	}

	return avatarBlob(c, e, buf)
}

func avatarBlob(c echo.Context, e *entity.Image, buf []byte) error {
	logger := logging.Logger(c.Request().Context())
	if e == nil {
		logger.Error("avatarBlob called with nil image")
		return Error(c, entity.Error(entity.InternalErrorCode, "avatar image is empty"))
	}

	c.Response().Header().Set("Cache-Control", "max-age=86400")
	if e.ETag != "" {
		c.Response().Header().Set("ETag", e.ETag)
	}

	return c.Blob(http.StatusOK, e.MimeType, buf)
}

func (u *userController) Avatars(c echo.Context) error {
	logger := logging.Logger(c.Request().Context())
	userID, err := entity.ParseEmail(c.Param("userId"))
	if err != nil {
		logger.Error("error parsing user id", zap.Error(err))
		return Error(c, err)
	}

	list, err := u.userService.ListByUserID(c.Request().Context(), userID)
	if err != nil {
		logger.Error("error listing user's avatars", zap.Error(err))
		return Error(c, err)
	}

	meta := make([]*Metadata, len(list))
	for i, e := range list {
		var metadata Metadata
		metadata.FromEntity(e, buildBaseURL(c))
		meta[i] = &metadata
	}
	return c.JSON(http.StatusOK, meta)
}

func (u *userController) Delete(c echo.Context) error {
	logger := logging.Logger(c.Request().Context())
	userID, err := entity.ParseEmail(c.Param("userId"))
	if err != nil {
		logger.Error("error parsing user id", zap.Error(err))
		return Error(c, err)
	}

	if err = u.userService.Delete(c.Request().Context(), userID); err != nil {
		logger.Error("error deleting user avatar", zap.Error(err))
		return Error(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}
