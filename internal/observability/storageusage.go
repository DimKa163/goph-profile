package observability

import (
	"context"

	"github.com/DimKa163/goph-profile/pkg/retryablepgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	StorageUsageStmt = `
						SELECT avatars.user_id, COALESCE(SUM(images.file_size), 0)::BIGINT from public.images
						JOIN public.avatars ON images.avatar_id = public.avatars.id
						WHERE avatars.deleted_at IS NULL
						GROUP BY avatars.user_id
						`
)

var storageUsageMetric metric.Int64ObservableGauge

func UseStorageUsageObserver(name string, pool *retryablepgxpool.Pool) error {
	meter := otel.Meter(name)
	storageUsage, err := meter.Int64ObservableGauge(
		"avatars_storage_bytes",
		metric.WithDescription("Total storage used by avatars"),
		metric.WithInt64Callback(func(ctx context.Context, observer metric.Int64Observer) error {
			rows, err := pool.Query(ctx, StorageUsageStmt)
			if err != nil {
				return err
			}
			defer rows.Close()
			for rows.Next() {
				var userID string
				var size int64
				if err = rows.Scan(&userID, &size); err != nil {
					return err
				}
				observer.Observe(size, metric.WithAttributes(
					attribute.String("user_id", userID),
				))
			}
			return nil
		}),
	)
	if err != nil {
		return err
	}
	storageUsageMetric = storageUsage
	return nil
}
