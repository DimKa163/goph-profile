package observability

import (
	"context"
	"net/http"

	"github.com/DimKa163/goph-profile/internal/entity"
	"github.com/DimKa163/goph-profile/internal/logging"
	"github.com/DimKa163/goph-profile/pkg/retryablepgxpool"
	"github.com/labstack/echo/v5"
	"go.uber.org/zap"
)

type healthState struct {
	Server bool `json:"server"`
	Db     bool `json:"db"`
	S3     bool `json:"s3"`
}

func (s healthState) statusCode() int {
	if s.Server && s.Db && s.S3 {
		return http.StatusOK
	}
	return http.StatusServiceUnavailable
}

func Health(ctx context.Context, e *echo.Echo, pool *retryablepgxpool.Pool, s3 entity.S3) {
	logger := logging.Logger(ctx)
	e.GET("/healthy", func(c *echo.Context) error {
		state := healthState{
			Server: true,
			Db:     true,
			S3:     true,
		}
		if err := pool.Ping(c.Request().Context()); err != nil {
			logger.Error("failed to ping postgres", zap.Error(err))
			state.Db = false
		}
		if err := s3.Check(c.Request().Context()); err != nil {
			logger.Error("failed to check S3 connection", zap.Error(err))
			state.S3 = false
		}
		return c.JSON(state.statusCode(), state)
	})
}
