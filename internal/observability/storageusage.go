package observability

import (
	"context"

	"github.com/DimKa163/goph-profile/pkg/retryablepgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	// storageUsageStmt defines the storage usage stmt value.
	storageUsageStmt = `
						SELECT avatars.user_id, COALESCE(SUM(images.file_size), 0)::BIGINT from public.images
						JOIN public.avatars ON images.avatar_id = public.avatars.id
						WHERE avatars.deleted_at IS NULL
						GROUP BY avatars.user_id
						`
)

// UseStorageUsageObserver registers a storage usage gauge callback.
func UseStorageUsageObserver(name string, pool *retryablepgxpool.Pool) error {
	meter := otel.Meter(name)
	storageUsage, err := meter.Int64ObservableGauge(
		"avatars_storage_bytes",
		metric.WithDescription("Total storage used by avatars"),
	)
	if err != nil {
		return err
	}

	_, err = meter.RegisterCallback(
		observeStorageUsage(pool, storageUsage),
		storageUsage,
	)
	return err
}

func observeStorageUsage(pool *retryablepgxpool.Pool, storageUsage metric.Int64ObservableGauge) metric.Callback {
	return func(ctx context.Context, observer metric.Observer) error {
		rows, err := pool.Query(ctx, storageUsageStmt)
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

			observer.ObserveInt64(storageUsage, size, metric.WithAttributes(
				attribute.String("user_id", userID),
			))
		}

		return rows.Err()
	}
}
