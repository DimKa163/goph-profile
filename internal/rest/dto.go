package rest

import (
	"fmt"
	"time"

	"github.com/DimKa163/goph-profile/internal/entity"
)

type (
	UploadResponse struct {
		ID        string    `json:"id"`
		UserID    string    `json:"userId"`
		Status    string    `json:"status"`
		CreatedAt time.Time `json:"created_at"`
		Url       string    `json:"url"`
	}
	Thumbnail struct {
		URL  string `json:"url"`
		Size string `json:"size"`
	}
	Dimension struct {
		Height int `json:"height"`
		Width  int `json:"width"`
	}
	Metadata struct {
		ID         string       `json:"id"`
		UserID     string       `json:"userId"`
		Filename   string       `json:"file_name"`
		MimeType   string       `json:"mime_type"`
		Size       int64        `json:"size"`
		Dimension  *Dimension   `json:"dimension"`
		Thumbnails []*Thumbnail `json:"thumbnails"`
		Inactive   bool         `json:"inactive"`
		CreatedAt  time.Time    `json:"created_at"`
		UpdatedAt  time.Time    `json:"updated_at"`
	}
)

func (m *Metadata) FromEntity(e *entity.Avatar, baseUrl string) {
	m.ID = e.ID.String()
	m.UserID = e.UserID.String()
	m.Filename = e.Name
	m.MimeType = e.MimeType
	m.Size = e.Size
	m.CreatedAt = e.CreatedAt
	m.UpdatedAt = e.UpdatedAt
	m.Dimension = &Dimension{
		Height: e.Height,
		Width:  e.Width,
	}
	m.Thumbnails = make([]*Thumbnail, len(e.Images))
	for i, thumbnail := range e.Images {
		m.Thumbnails[i] = &Thumbnail{
			URL:  fmt.Sprintf("%s/api/v1/avatars/%s?size=%s&format=%s", baseUrl, e.ID.String(), thumbnail.Size, thumbnail.Format),
			Size: thumbnail.Size.String(),
		}
	}
}
