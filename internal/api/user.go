package api

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

type userController struct {
}

func NewUserController() *userController {
	return &userController{}
}

func (u *userController) Register(c *echo.Group) {
	c.GET("/users/:userId/avatar", u.Avatar)
	c.GET("/users/:userId/avatars", u.Avatars)
	c.DELETE("/users/:userId/avatar", u.Delete)
}

func (u *userController) Avatar(c echo.Context) error {
	return c.JSON(http.StatusGone, u)
}

func (u *userController) Avatars(c echo.Context) error {
	return c.JSON(http.StatusGone, u)
}

func (u *userController) Delete(c echo.Context) error {
	return c.JSON(http.StatusGone, u)
}
