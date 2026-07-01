package api

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/DimKa163/goph-profile/internal/services"
	"github.com/beevik/guid"
	"github.com/labstack/echo/v4"
)

type (
	UploadResponse struct {
		ID        string    `json:"id"`
		UserID    string    `json:"userId"`
		Status    string    `json:"status"`
		CreatedAt time.Time `json:"created_at"`
		Url       string    `json:"url"`
	}
	Thumbnail struct {
		URL  string `json:"url"`
		Size string `json:"size"`
	}
	Dimension struct {
		Height int `json:"height"`
		Width  int `json:"width"`
	}
	Metadata struct {
		ID         string       `json:"id"`
		UserID     string       `json:"userId"`
		Filename   string       `json:"file_name"`
		MimeType   string       `json:"mime_type"`
		Size       int64        `json:"size"`
		Dimension  *Dimension   `json:"dimension"`
		Thumbnails []*Thumbnail `json:"thumbnail"`
		CreatedAt  time.Time    `json:"created_at"`
		UpdatedAt  time.Time    `json:"updated_at"`
	}
)

type avatarController struct {
	service *services.AvatarService
}

func NewAvatarController(service *services.AvatarService) *avatarController {
	return &avatarController{
		service: service,
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
	e, err := a.service.Upload(c.Request().Context(), &services.UploadCommand{
		Reader:   src,
		FileName: img.Filename,
		MimeType: mimeType,
		Size:     img.Size,
		UserID:   *userID,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusCreated, UploadResponse{
		ID:        e.ID.String(),
		UserID:    e.UserID.String(),
		Status:    e.ProcessingStatus,
		Url:       fmt.Sprintf("%s/api/v1/avatars/%s", buildBaseURL(c), userID),
		CreatedAt: e.CreatedAt,
	})
}

func (a *avatarController) Get(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, "Not Implemented")
}

func (a *avatarController) Delete(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, "Not Implemented")
}

func (a *avatarController) Metadata(c echo.Context) error {
	avatarParam := c.Param("avatar_id")
	avatarID, err := guid.ParseString(avatarParam)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	e, err := a.service.Metadata(c.Request().Context(), *avatarID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	var response Metadata
	response.ID = e.ID.String()
	response.UserID = e.UserID.String()
	response.Filename = e.Name
	response.MimeType = e.MimeType
	response.Size = e.Size
	response.CreatedAt = e.CreatedAt
	response.UpdatedAt = e.UpdatedAt
	response.Dimension = &Dimension{
		Height: e.Height,
		Width:  e.Width,
	}
	response.Thumbnails = make([]*Thumbnail, len(e.Thumbnails))
	for i, thumbnail := range e.Thumbnails {
		response.Thumbnails[i] = &Thumbnail{
			URL:  fmt.Sprintf("%s/api/v1/avatars/%s?size=%s&format=wbep", buildBaseURL(c), e.UserID.String(), thumbnail.Size),
			Size: thumbnail.Size,
		}
	}
	return c.JSON(http.StatusOK, response)
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
