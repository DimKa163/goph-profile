package repository

import (
	"context"
	"database/sql"

	"github.com/DimKa163/goph-profile/internal/entity"
	"github.com/beevik/guid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	FindStmt = `SELECT 
    				id, 
    				created_at, 
    				updated_at, 
    				name, 
    				s3_key, 
    				mime_type, 
    				user_id, 
    				upload_status, 
    				processing_status, 
    				file_size, 
    				width, 
    				height,
    				size, 
    				url
				FROM avatars
				LEFT JOIN public.thumbnails ON avatars.id = thumbnails.avatar_id
				WHERE id = $1`
	InsertAvatarMetadataStmt = `INSERT INTO public.avatars(name, mime_type, user_id, file_size, width, height) 
								VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, created_at, updated_at, name, 
								mime_type, user_id, upload_status, processing_status, file_size, width, height;`
	UpdateStmt = `UPDATE public.avatars SET s3_key= $1, upload_status=$2, processing_status=$3, updated_at=now() WHERE id=$4
 								RETURNING id, created_at, updated_at, name, s3_key, mime_type, user_id, upload_status, processing_status, file_size, width, height;`
	InsertThumbStmt = `INSERT INTO public.thumbnails(avatar_id, size, url) VALUES ($1, $2, $3);`
)

type avatarRepository struct {
	pool *pgxpool.Pool
}

func NewAvatarRepository(pool *pgxpool.Pool) *avatarRepository {
	return &avatarRepository{pool: pool}
}

func (r *avatarRepository) Find(ctx context.Context, id guid.Guid) (*entity.Avatar, error) {
	var avatar entity.Avatar
	rows, err := r.pool.Query(ctx, FindStmt, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var thumbnailSize, thumbnailURL sql.NullString
		if err = rows.Scan(
			&avatar.ID,
			&avatar.CreatedAt,
			&avatar.UpdatedAt,
			&avatar.Name,
			&avatar.S3Key,
			&avatar.MimeType,
			&avatar.UserID,
			&avatar.UploadStatus,
			&avatar.ProcessingStatus,
			&avatar.Size,
			&avatar.Width,
			&avatar.Height,
			&thumbnailSize,
			&thumbnailURL,
		); err != nil {
			return nil, err
		}
		if thumbnailSize.Valid && thumbnailURL.Valid {
			avatar.Thumbnails = append(avatar.Thumbnails, &entity.Thumbnail{
				Size: thumbnailSize.String,
				Url:  thumbnailURL.String,
			})
		}
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	if avatar.ID == (guid.Guid{}) {
		return nil, pgx.ErrNoRows
	}
	return &avatar, nil
}

func (r *avatarRepository) Insert(ctx context.Context, name, mimeType string, size int64, width, height int, userID guid.Guid) (*entity.Avatar, error) {
	var avatar entity.Avatar
	if err := r.pool.QueryRow(
		ctx,
		InsertAvatarMetadataStmt,
		name,
		mimeType,
		userID,
		size,
		width,
		height,
	).
		Scan(
			&avatar.ID,
			&avatar.CreatedAt,
			&avatar.UpdatedAt,
			&avatar.Name,
			&avatar.MimeType,
			&avatar.UserID,
			&avatar.UploadStatus,
			&avatar.ProcessingStatus,
			&avatar.Size,
			&avatar.Width,
			&avatar.Height,
		); err != nil {
		return nil, err
	}
	return &avatar, nil
}

func (r *avatarRepository) Update(ctx context.Context, e *entity.Avatar) (*entity.Avatar, error) {
	var avatar entity.Avatar
	if err := r.pool.QueryRow(
		ctx,
		UpdateStmt,
		e.S3Key,
		e.UploadStatus,
		e.ProcessingStatus,
		e.ID,
	).
		Scan(
			&avatar.ID,
			&avatar.CreatedAt,
			&avatar.UpdatedAt,
			&avatar.Name,
			&avatar.S3Key,
			&avatar.MimeType,
			&avatar.UserID,
			&avatar.UploadStatus,
			&avatar.ProcessingStatus,
			&avatar.Size,
			&avatar.Width,
			&avatar.Height,
		); err != nil {
		return nil, err
	}
	if e.Thumbnails != nil {
		for _, thumbnail := range e.Thumbnails {
			if _, err := r.pool.Exec(ctx, InsertThumbStmt, e.ID, thumbnail.Size, thumbnail.Url); err != nil {
				return nil, err
			}
		}
	}

	return &avatar, nil
}
