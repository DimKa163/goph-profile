package api

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

type avatarController struct {
}

func NewAvatarController() *avatarController {
	return &avatarController{}
}

func (a *avatarController) Register(e *echo.Group) {
	e.GET("/avatars/:avatar_id", a.Get)
	e.POST("/avatars", a.Avatar)
	e.DELETE("/avatars/:avatar_id", a.Delete)
	e.GET("/avatars/:avatar_id/metadata", a.Metadata)
}

func (a *avatarController) Avatar(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, "Not Implemented")
}

func (a *avatarController) Get(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, "Not Implemented")
}

func (a *avatarController) Delete(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, "Not Implemented")
}

func (a *avatarController) Metadata(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, "Not Implemented")
}
