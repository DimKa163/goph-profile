package rest

import (
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/DimKa163/goph-profile/internal/entity"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/bytes"
)

func bodyLimit(maxSize string) echo.MiddlewareFunc {
	limit, err := bytes.Parse(maxSize)
	if err != nil {
		panic(fmt.Errorf("body limit parse error: %v", err))
	}
	pool := limitedReaderPool(limit)
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			if req.ContentLength > limit {
				return Error(c, entity.WrapError(entity.InvalidSizeErrorCode, req.ContentLength, nil))
			}
			r, ok := pool.Get().(*limitedReader)
			if !ok {
				return echo.NewHTTPError(http.StatusInternalServerError, "cast error")
			}
			r.Reset(req.Body)
			defer pool.Put(r)
			req.Body = r
			return next(c)
		}
	}
}

type limitedReader struct {
	maxSize int64
	reader  io.Reader
	read    int64
}

// Read decodes bytes into the receiver.
func (r *limitedReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	r.read += int64(n)
	if r.read > r.maxSize {
		return 0, entity.WrapError(entity.InvalidSizeErrorCode, r.read, nil)
	}
	return n, err
}

// Close releases resources.
func (r *limitedReader) Close() error {
	return r.reader.(io.Closer).Close()
}

// Reset restores the request body for another read.
func (r *limitedReader) Reset(reader io.ReadCloser) {
	r.reader = reader
	r.read = 0
}

func limitedReaderPool(maxSize int64) sync.Pool {
	return sync.Pool{
		New: func() interface{} {
			return &limitedReader{maxSize: maxSize}
		},
	}
}
