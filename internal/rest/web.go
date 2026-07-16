package rest

import (
	"fmt"
	"net/http"

	"github.com/DimKa163/goph-profile/internal/entity"
	"github.com/DimKa163/goph-profile/internal/logging"
	"github.com/DimKa163/goph-profile/internal/usecase"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type Image struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Inactive bool   `json:"inactive"`
	Original string `json:"original"`
	S300     string `json:"s300"`
	S100     string `json:"s100"`
}
type webController struct {
	userService *usecase.UserService
}

func NewWebController(userServices *usecase.UserService) *webController {
	return &webController{userService: userServices}
}

func (w *webController) Register(e Section) {
	e.GET("/web/upload", w.Index)
	e.GET("/web/gallery/:userId", w.Gallery)
}

func (w *webController) Index(c echo.Context) error {
	return c.File("web/static/index.html")
}

func (w *webController) Gallery(c echo.Context) error {
	logger := logging.Logger(c.Request().Context())
	userIDStr := c.Param("userId")
	userID, err := entity.ParseEmail(userIDStr)
	if err != nil {
		logger.Error("error parsing user id", zap.Error(err))
		return Error(c, err)
	}
	avatars, err := w.userService.ListByUserID(c.Request().Context(), userID)
	if err != nil {
		logger.Error("error getting user avatars", zap.Error(err))
		return Error(c, err)
	}
	m := make([]*Image, len(avatars))
	for i, avatar := range avatars {
		m[i] = &Image{
			ID:       avatar.ID.String(),
			Name:     avatar.Name,
			Inactive: avatar.Inactive,
		}
		for _, th := range avatar.Images {
			if th.Format == "webp" {
				switch th.Size {
				case entity.OriginalSize:
					m[i].Original = fmt.Sprintf("%s/api/v1/avatars/%s?size=%s&format=%s", buildBaseURL(c), avatar.ID.String(), th.Size, th.Format)
				case entity.S300x300Size:
					m[i].S300 = fmt.Sprintf("%s/api/v1/avatars/%s?size=%s&format=%s", buildBaseURL(c), avatar.ID.String(), th.Size, th.Format)
				case entity.S100x100Size:
					m[i].S100 = fmt.Sprintf("%s/api/v1/avatars/%s?size=%s&format=%s", buildBaseURL(c), avatar.ID.String(), th.Size, th.Format)
				}
			}
		}
	}
	return c.Render(http.StatusOK, "gallery.html", map[string]interface{}{
		"UserID": userID,
		"Images": m,
	})
}
