// Package infra contains infrastructure adapters for persistence, storage, and messaging.
package infra

import (
	"context"
	"database/sql"
	"errors"

	"github.com/DimKa163/goph-profile/internal/entity"
	"github.com/DimKa163/goph-profile/pkg/retryablepgxpool"
	"github.com/jackc/pgx/v5"
)

const (
	findStmt = `SELECT 
    				id, 
    				avatars.created_at, 
    				updated_at, 
    				name,
    				user_id,
    				width, 
    				height,
    				avatars.file_size,
    				avatars.mime_type,
    				avatars.inactive,
    				images.format,
    				images.s3_key, 
    				images.mime_type, 
    				images.e_tag,
    				images.file_size, 
    				images.size, 
    				images.created_at
				FROM avatars
				LEFT JOIN public.images ON avatars.id = images.avatar_id
				WHERE id = $1 AND deleted_at IS NULL;`
	findByUserIDStmt = `SELECT
    				id, 
    				avatars.created_at, 
    				updated_at, 
    				name,
    				user_id,
    				width, 
    				height,
    				avatars.file_size,
    				avatars.mime_type,
    				avatars.inactive,
    				images.format,
    				images.s3_key, 
    				images.mime_type, 
    				images.e_tag,
    				images.file_size, 
    				images.size, 
    				images.created_at
				FROM avatars
				LEFT JOIN public.images ON avatars.id = images.avatar_id
				WHERE user_id = $1 AND inactive = false AND deleted_at IS NULL;`
	listByUserIDStmt = `SELECT
    				id, 
    				avatars.created_at, 
    				updated_at, 
    				name,
    				user_id,
    				width, 
    				height,
    				avatars.file_size,
    				avatars.mime_type,
    				avatars.inactive,
    				images.format,
    				images.s3_key, 
    				images.mime_type, 
    				images.e_tag,
    				images.file_size, 
    				images.size, 
    				images.created_at
				FROM avatars
				LEFT JOIN public.images ON avatars.id = images.avatar_id
				WHERE user_id = $1 AND deleted_at IS NULL
				ORDER BY avatars.id ASC;`
	findImageStmt = `SELECT
					format, 
					size, 
					file_size,
					s3_key,
					e_tag, 
					mime_type, 
					created_at 
					FROM public.images
					WHERE avatar_id = $1
					ORDER BY created_at ASC;`
	insertAvatarMetadataStmt = `INSERT INTO public.avatars(id, name, user_id, width, height, file_size, mime_type, inactive) 
								VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id, created_at, updated_at, name, user_id, width, height, file_size, mime_type, inactive;`
	insertImageMetadataStmt = `INSERT INTO public.images(avatar_id, format, size, file_size, mime_type, s3_key, e_tag) VALUES ($1, $2, $3, $4, $5, $6, $7) 
								ON CONFLICT(avatar_id, format, size) DO NOTHING
								RETURNING format, size, file_size, mime_type, s3_key, e_tag, created_at;`
	deleteStmt                      = `UPDATE public.avatars SET inactive=true, deleted_at=now(), updated_at=now() WHERE id=$1 AND deleted_at IS NULL;`
	deleteImagesStmt                = `DELETE FROM public.images WHERE avatar_id = $1;`
	deactivateUserAvatarsExceptStmt = `UPDATE public.avatars
						SET inactive = TRUE,
						updated_at = now()
						WHERE user_id = $1
  							AND id <> $2
  							AND inactive = FALSE
  							AND deleted_at IS NULL;`
	activateUserAvatarStmt = `UPDATE public.avatars
							SET inactive = FALSE,
    						updated_at = now()
								WHERE id = $2
								AND user_id = $1
							  AND deleted_at IS NULL;`
)

type avatarRepository struct {
	pool *retryablepgxpool.Pool
}

// NewAvatarRepository creates an avatar repository.
func NewAvatarRepository(pool *retryablepgxpool.Pool) *avatarRepository {
	return &avatarRepository{pool: pool}
}

// Find returns avatar metadata by ID.
func (r *avatarRepository) Find(ctx context.Context, id entity.AvatarID) (*entity.Avatar, error) {
	var avatar entity.Avatar
	rows, err := getCon(ctx, r.pool).Query(ctx, findStmt, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var imageFormat, imageS3, imageMimeType, imageEtag, imageSize sql.NullString
		var imageFileSize sql.NullInt64
		var imageCreatedAt sql.NullTime
		if err = rows.Scan(
			&avatar.ID,
			&avatar.CreatedAt,
			&avatar.UpdatedAt,
			&avatar.Name,
			&avatar.UserID,
			&avatar.Width,
			&avatar.Height,
			&avatar.Size,
			&avatar.MimeType,
			&avatar.Inactive,
			&imageFormat,
			&imageS3,
			&imageMimeType,
			&imageEtag,
			&imageFileSize,
			&imageSize,
			&imageCreatedAt,
		); err != nil {
			return nil, err
		}
		if imageFormat.Valid && imageS3.Valid && imageMimeType.Valid && imageEtag.Valid && imageSize.Valid && imageCreatedAt.Valid {
			avatar.Images = append(avatar.Images, &entity.Image{
				CreatedAt: imageCreatedAt.Time,
				Format:    imageFormat.String,
				S3Key:     imageS3.String,
				MimeType:  imageMimeType.String,
				ETag:      imageEtag.String,
				Size:      entity.Size(imageSize.String),
				FileSize:  imageFileSize.Int64,
			})
		}
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	if avatar.ID == (entity.AvatarID{}) {
		return nil, ErrNoRows
	}
	return &avatar, nil
}

// FindByUserID returns the active avatar for a user.
func (r *avatarRepository) FindByUserID(ctx context.Context, id entity.Email) (*entity.Avatar, error) {
	var avatar entity.Avatar
	rows, err := getCon(ctx, r.pool).Query(ctx, findByUserIDStmt, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var imageFormat, imageS3, imageMimeType, imageEtag, imageSize sql.NullString
		var imageFileSize sql.NullInt64
		var imageCreatedAt sql.NullTime
		if err = rows.Scan(
			&avatar.ID,
			&avatar.CreatedAt,
			&avatar.UpdatedAt,
			&avatar.Name,
			&avatar.UserID,
			&avatar.Width,
			&avatar.Height,
			&avatar.Size,
			&avatar.MimeType,
			&avatar.Inactive,
			&imageFormat,
			&imageS3,
			&imageMimeType,
			&imageEtag,
			&imageFileSize,
			&imageSize,
			&imageCreatedAt,
		); err != nil {
			return nil, err
		}
		if imageFormat.Valid && imageS3.Valid && imageMimeType.Valid && imageEtag.Valid && imageSize.Valid && imageCreatedAt.Valid {
			avatar.Images = append(avatar.Images, &entity.Image{
				CreatedAt: imageCreatedAt.Time,
				Format:    imageFormat.String,
				S3Key:     imageS3.String,
				MimeType:  imageMimeType.String,
				ETag:      imageEtag.String,
				Size:      entity.Size(imageSize.String),
				FileSize:  imageFileSize.Int64,
			})
		}
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	if avatar.ID == (entity.AvatarID{}) {
		return nil, ErrNoRows
	}
	return &avatar, nil
}

// ListByUserID returns avatars for a user.
func (r *avatarRepository) ListByUserID(ctx context.Context, userID entity.Email) ([]*entity.Avatar, error) {
	rows, err := getCon(ctx, r.pool).Query(ctx, listByUserIDStmt, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := make(map[entity.AvatarID]*entity.Avatar)
	ordered := make([]*entity.Avatar, 0)
	for rows.Next() {
		var avatar entity.Avatar
		var imageFormat, imageS3, imageMimeType, imageEtag, imageSize sql.NullString
		var imageFileSize sql.NullInt64
		var imageCreatedAt sql.NullTime
		if err = rows.Scan(
			&avatar.ID,
			&avatar.CreatedAt,
			&avatar.UpdatedAt,
			&avatar.Name,
			&avatar.UserID,
			&avatar.Width,
			&avatar.Height,
			&avatar.Size,
			&avatar.MimeType,
			&avatar.Inactive,
			&imageFormat,
			&imageS3,
			&imageMimeType,
			&imageEtag,
			&imageFileSize,
			&imageSize,
			&imageCreatedAt,
		); err != nil {
			return nil, err
		}
		e, ok := m[avatar.ID]
		if !ok {
			e = &avatar
			m[avatar.ID] = e
			ordered = append(ordered, e)
		}
		if imageFormat.Valid && imageS3.Valid && imageMimeType.Valid && imageEtag.Valid && imageSize.Valid && imageCreatedAt.Valid {
			e.Images = append(e.Images, &entity.Image{
				CreatedAt: imageCreatedAt.Time,
				Format:    imageFormat.String,
				S3Key:     imageS3.String,
				MimeType:  imageMimeType.String,
				ETag:      imageEtag.String,
				Size:      entity.Size(imageSize.String),
				FileSize:  imageFileSize.Int64,
			})
		}
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return ordered, nil
}

// FindImage returns image metadata by storage key.
func (r *avatarRepository) FindImage(ctx context.Context, avatarID entity.AvatarID) ([]*entity.Image, error) {
	c := getCon(ctx, r.pool)
	rows, err := c.Query(
		ctx,
		findImageStmt,
		avatarID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNoRows
		}
		return nil, err
	}
	defer rows.Close()
	images := make([]*entity.Image, 0, 9)
	for rows.Next() {
		var image entity.Image
		if err = rows.Scan(
			&image.Format,
			&image.Size,
			&image.FileSize,
			&image.S3Key,
			&image.ETag,
			&image.MimeType,
			&image.CreatedAt); err != nil {
			return nil, err
		}
		images = append(images, &image)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	if len(images) == 0 {
		return nil, ErrNoRows
	}

	return images, nil
}

// Insert stores a new record.
func (r *avatarRepository) Insert(
	ctx context.Context,
	id entity.AvatarID,
	name string,
	userID entity.Email,
	width, height int, size int64, mimeType string,
	rows ...*entity.Image,
) (*entity.Avatar, error) {
	var avatar entity.Avatar
	c := getCon(ctx, r.pool)
	if err := c.QueryRow(
		ctx,
		insertAvatarMetadataStmt,
		id,
		name,
		userID,
		width,
		height,
		size,
		mimeType,
		true,
	).
		Scan(
			&avatar.ID,
			&avatar.CreatedAt,
			&avatar.UpdatedAt,
			&avatar.Name,
			&avatar.UserID,
			&avatar.Width,
			&avatar.Height,
			&avatar.Size,
			&avatar.MimeType,
			&avatar.Inactive,
		); err != nil {
		return nil, err
	}
	for _, row := range rows {
		if err := c.QueryRow(
			ctx,
			insertImageMetadataStmt,
			avatar.ID,
			row.Format,
			row.Size,
			row.FileSize,
			row.MimeType,
			row.S3Key,
			row.ETag,
		).
			Scan(
				&row.Format,
				&row.Size,
				&row.FileSize,
				&row.MimeType,
				&row.S3Key,
				&row.ETag,
				&row.CreatedAt,
			); err != nil {
			return nil, err
		}
		avatar.Images = append(avatar.Images, row)
	}
	return &avatar, nil
}

// InsertImage stores generated image metadata.
func (r *avatarRepository) InsertImage(ctx context.Context, e *entity.Avatar) error {
	var err error
	c := getCon(ctx, r.pool)
	for _, row := range e.Images {
		if _, err = c.Exec(
			ctx,
			insertImageMetadataStmt,
			e.ID,
			row.Format,
			row.Size.String(),
			row.FileSize,
			row.MimeType,
			row.S3Key,
			row.ETag,
		); err != nil {
			return err
		}
	}
	return err
}

// ActivateOnlyThis activates one avatar and deactivates the others for the user.
func (r *avatarRepository) ActivateOnlyThis(ctx context.Context, userID entity.Email, id entity.AvatarID) error {
	c := getCon(ctx, r.pool)
	if _, err := c.Exec(ctx, deactivateUserAvatarsExceptStmt, userID, id); err != nil {
		return err
	}
	if _, err := c.Exec(ctx, activateUserAvatarStmt, userID, id); err != nil {
		return err
	}
	return nil
}

// Delete removes or marks a record as deleted.
func (r *avatarRepository) Delete(ctx context.Context, id entity.AvatarID) error {
	c := getCon(ctx, r.pool)
	tg, err := c.Exec(ctx, deleteStmt, id)
	if err != nil {
		return err
	}
	if tg.RowsAffected() == 0 {
		return ErrNoRows
	}
	return nil
}

// DeleteImages removes image metadata for an avatar.
func (r *avatarRepository) DeleteImages(ctx context.Context, id entity.AvatarID) error {
	c := getCon(ctx, r.pool)
	tg, err := c.Exec(ctx, deleteImagesStmt, id)
	if err != nil {
		return err
	}
	if tg.RowsAffected() == 0 {
		return ErrNoRows
	}
	return nil
}
