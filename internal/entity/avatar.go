package entity

import (
	"time"

	"github.com/beevik/guid"
)

type Avatar struct {
	ID               guid.Guid
	CreatedAt        time.Time
	UpdatedAt        time.Time
	Name             string
	S3Key            string
	UserID           guid.Guid
	UploadStatus     string
	ProcessingStatus string
	MimeType         string
	Size             int64
	Width            int
	Height           int
	Thumbnails       []*Thumbnail
}

type Thumbnail struct {
	Size string
	Url  string
}
