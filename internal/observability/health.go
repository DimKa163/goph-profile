package observability

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"time"

	"github.com/DimKa163/goph-profile/internal/entity"
	"github.com/DimKa163/goph-profile/internal/logging"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/zap"
)

type healthState struct {
	Server bool  `json:"server"`
	Db     bool  `json:"db"`
	S3     *bool `json:"s3,omitempty"`
	Kafka  *bool `json:"kafka,omitempty"`
}

func (s healthState) statusCode() int {
	if !s.Server {
		return http.StatusServiceUnavailable
	}
	if !s.Db {
		return http.StatusServiceUnavailable
	}
	if s.S3 != nil && !*s.S3 {
		return http.StatusServiceUnavailable
	}
	if s.Kafka != nil && !*s.Kafka {
		return http.StatusServiceUnavailable
	}
	return http.StatusOK
}

func Health(ctx context.Context, addr string, pool *pgxpool.Pool, s3 entity.S3, kafkaClient *kgo.Client) error {
	logger := logging.Logger(ctx)
	listenConfig := net.ListenConfig{}
	listener, err := listenConfig.Listen(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	go func() {
		server := http.Server{}
		mux := http.NewServeMux()
		mux.HandleFunc("/healthy", func(w http.ResponseWriter, r *http.Request) {
			state := healthState{
				Server: true,
				Db:     true,
			}
			if err := pool.Ping(r.Context()); err != nil {
				logger.Error("failed to ping postgres", zap.Error(err))
				state.Db = false
			}
			if s3 != nil {
				state.S3 = new(true)
				if err := s3.Check(r.Context()); err != nil {
					logger.Error("failed to check S3 connection", zap.Error(err))
					state.S3 = new(false)
				}
			}
			if kafkaClient != nil {
				state.Kafka = new(true)
				if err := kafkaClient.Ping(r.Context()); err != nil {
					logger.Error("failed to ping kafka", zap.Error(err))
					state.Kafka = new(false)
				}
			}

			data, err := json.Marshal(state)
			if err != nil {
				logger.Error("failed to marshal state", zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(state.statusCode())
			_, _ = w.Write(data)
		})
		server.Handler = mux
		go func() {
			<-ctx.Done()
			timeoutCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 1*time.Second)
			defer cancel()
			if err = server.Shutdown(timeoutCtx); err != nil {
				logger.Warn("failed to shutdown server", zap.Error(err))
			}
		}()
		if err = server.Serve(listener); err != nil {
			logger.Fatal("failed to serve http", zap.Error(err))
		}
	}()
	return nil
}
