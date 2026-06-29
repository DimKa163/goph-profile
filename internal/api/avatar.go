package api

import (
	"fmt"
	"io"
	"net/http"

	"github.com/DimKa163/goph-profile/internal/handlers"
	"github.com/beevik/guid"
	"github.com/labstack/echo/v4"
)

type avatarController struct {
	uploader handlers.Uploader
}

func NewAvatarController(uploader handlers.Uploader) *avatarController {
	return &avatarController{
		uploader: uploader,
	}
}

func (a *avatarController) Register(e *echo.Group) {
	e.GET("/avatars/:avatar_id", a.Get)
	e.POST("/avatars", a.Avatar)
	e.DELETE("/avatars/:avatar_id", a.Delete)
	e.GET("/avatars/:avatar_id/metadata", a.Metadata)
}

func (a *avatarController) Avatar(c echo.Context) error {
	userIDString := c.Request().Header.Get("X-User-Id")
	userID, err := guid.ParseString(userIDString)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	img, err := c.FormFile("image")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	src, _ := img.Open()
	defer src.Close()
	mimeType, _ := readMimeType(src)
	st, err := a.uploader(c.Request().Context(), &handlers.Avatar{
		Reader:   src,
		Name:     img.Filename,
		MimeType: mimeType,
		Size:     img.Size,
	}, *userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusCreated, struct {
		handlers.UploaderState
		url string
	}{
		UploaderState: *st,
		url:           fmt.Sprintf("%s/api/v1/avatars/%s", buildBaseURL(c), userID),
	})
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

func buildBaseURL(c echo.Context) string {
	req := c.Request()

	scheme := req.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		if req.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}

	host := req.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = req.Host
	}

	return scheme + "://" + host
}

func readMimeType(r io.ReadSeeker) (string, error) {
	buf := make([]byte, 512)

	n, err := r.Read(buf)
	if err != nil && err != io.EOF {
		return "", err
	}

	mimeType := http.DetectContentType(buf[:n])
	_, _ = r.Seek(0, io.SeekStart)
	return mimeType, nil
}
