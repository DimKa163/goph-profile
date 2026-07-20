package rest

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/DimKa163/goph-profile/internal/entity"
	"github.com/DimKa163/goph-profile/internal/logging"
	"github.com/DimKa163/goph-profile/internal/observability"
	"github.com/DimKa163/goph-profile/internal/usecase"
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type avatarController struct {
	metric  observability.MetricService
	service *usecase.AvatarService
}

func NewAvatarController(metric observability.MetricService, service *usecase.AvatarService) *avatarController {
	return &avatarController{
		metric:  metric,
		service: service,
	}
}

func (a *avatarController) Register(e Section) {
	e.GET("/avatars/:avatar_id", a.Get)
	e.POST("/avatars", a.Avatar)
	e.DELETE("/avatars/:avatar_id", a.Delete)
	e.GET("/avatars/:avatar_id/metadata", a.Metadata)
}

func (a *avatarController) Avatar(c echo.Context) error {
	span := trace.SpanFromContext(c.Request().Context())
	logger := logging.Logger(c.Request().Context())
	img, err := c.FormFile("image")
	if err != nil {
		logger.Error("error getting image", zap.Error(err))
		return Error(c, err)
	}

	userID, err := entity.ParseEmail(c.Request().Header.Get("X-User-Id"))
	if err != nil {
		logger.Error("error parsing user id", zap.Error(err))
		return Error(c, err)
	}
	src, err := img.Open()
	if err != nil {
		logger.Error("error opening image", zap.Error(err))
		return Error(c, err)
	}
	defer func() {
		_ = src.Close()
	}()

	mimeType, err := readMimeType(src)
	if err != nil {
		logger.Error("error reading mime type", zap.Error(err))
		return Error(c, err)
	}
	startTime := time.Now()
	status := observability.Success
	defer func() {
		since := time.Since(startTime)
		a.metric.AvatarUploaded(c.Request().Context(), userID, status)
		a.metric.AvatarUploadDuration(c.Request().Context(), status, since)
	}()
	span.SetAttributes(
		attribute.String("user_id", userID.String()),
		attribute.String("mime_type", mimeType),
		attribute.Int64("file_size", img.Size),
	)
	buf, size, err := readToBuffer(src)
	if err != nil {
		logger.Error("error reading file", zap.Error(err))
		return Error(c, err)
	}
	e, err := a.service.Upload(c.Request().Context(), &usecase.UploadCommand{
		Buf:      buf,
		FileName: img.Filename,
		MimeType: mimeType,
		Size:     size,
		UserID:   userID,
	})
	if err != nil {
		status = observability.Failure
		logger.Error("error uploading file", zap.Error(err))
		return Error(c, err)
	}

	return c.JSON(http.StatusCreated, UploadResponse{
		ID:        e.ID.String(),
		UserID:    e.UserID.String(),
		Status:    "processing",
		Url:       fmt.Sprintf("%s/api/v1/avatars/%s", buildBaseURL(c), e.ID.String()),
		CreatedAt: e.CreatedAt,
	})
}

func (a *avatarController) Get(c echo.Context) error {
	logger := logging.Logger(c.Request().Context())
	id, err := entity.ParseAvatarID(c.Param("avatar_id"))
	if err != nil {
		logger.Error("error parsing avatar id", zap.Error(err))
		return Error(c, err)
	}

	e, buf, err := a.service.Get(c.Request().Context(), c.Request().Header.Get("If-None-Match"), &usecase.Request{
		ID:     id,
		Format: c.QueryParam("format"),
		Size:   entity.Size(c.QueryParam("size")),
	})
	if err != nil {
		if errors.Is(err, usecase.ErrAvatarNotModified) {
			return c.NoContent(http.StatusNotModified)
		}
		logger.Error("error getting avatar", zap.Error(err))
		return Error(c, err)
	}

	c.Response().Header().Set("Cache-Control", "max-age=86400")
	c.Response().Header().Set("ETag", e.ETag)

	return c.Blob(http.StatusOK, e.MimeType, buf)
}

func (a *avatarController) Delete(c echo.Context) error {
	logger := logging.Logger(c.Request().Context())
	id, err := entity.ParseAvatarID(c.Param("avatar_id"))
	if err != nil {
		logger.Error("error parsing avatar id", zap.Error(err))
		return Error(c, err)
	}

	userID, err := entity.ParseEmail(c.Request().Header.Get("X-User-Id"))
	if err != nil {
		logger.Error("error parsing user id", zap.Error(err))
		return Error(c, err)
	}

	if err = a.service.Delete(c.Request().Context(), id, userID); err != nil {
		logger.Error("error deleting avatar", zap.Error(err))
		return Error(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (a *avatarController) Metadata(c echo.Context) error {
	logger := logging.Logger(c.Request().Context())
	avatarID, err := entity.ParseAvatarID(c.Param("avatar_id"))
	if err != nil {
		logger.Error("error parsing avatar id", zap.Error(err))
		return Error(c, err)
	}

	e, err := a.service.Metadata(c.Request().Context(), avatarID)
	if err != nil {
		logger.Error("error getting avatar metadata", zap.Error(err))
		return Error(c, err)
	}

	var response Metadata
	response.FromEntity(e, buildBaseURL(c))
	return c.JSON(http.StatusOK, &response)
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

func readToBuffer(rs io.ReadSeeker) ([]byte, int64, error) {
	if _, err := rs.Seek(0, io.SeekStart); err != nil {
		return nil, 0, err
	}

	buf, err := io.ReadAll(rs)
	if err != nil {
		return nil, 0, err
	}

	return buf, int64(len(buf)), nil
}
