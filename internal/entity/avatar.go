package entity

import (
	"encoding/json"
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

type AvatarUploadedEvent struct {
	AvatarID string `json:"avatar_id"`
	UserID   string `json:"user_id"`
	S3Key    string `json:"s3_key"`
}

func (e *AvatarUploadedEvent) Read(data []byte) error {
	return json.Unmarshal(data, e)
}

func (e *AvatarUploadedEvent) Bytes() ([]byte, error) {
	return json.Marshal(e)
}

func (e *AvatarUploadedEvent) String() string {
	return "AvatarUploadedEvent"
}
