package rest

import (
	"fmt"
	"time"

	"github.com/DimKa163/goph-profile/internal/entity"
)

type (
	// UploadResponse is returned after an avatar upload.
	UploadResponse struct {
		// ID stores the identifier.
		ID string `json:"id"`
		// UserID stores the user identifier.
		UserID string `json:"userId"`
		// Status stores the status value.
		Status string `json:"status"`
		// CreatedAt stores the created at value.
		CreatedAt time.Time `json:"created_at"`
		// Url stores the url value.
		Url string `json:"url"`
	}
	// Thumbnail describes a generated avatar thumbnail.
	Thumbnail struct {
		// URL stores the u r l value.
		URL string `json:"url"`
		// Size stores the size value.
		Size string `json:"size"`
	}
	// Dimension describes image dimensions.
	Dimension struct {
		// Height stores the height value.
		Height int `json:"height"`
		// Width stores the width value.
		Width int `json:"width"`
	}
	// Metadata describes avatar metadata returned by the API.
	Metadata struct {
		// ID stores the identifier.
		ID string `json:"id"`
		// UserID stores the user identifier.
		UserID string `json:"userId"`
		// Filename stores the filename value.
		Filename string `json:"file_name"`
		// MimeType stores the mime type value.
		MimeType string `json:"mime_type"`
		// Size stores the size value.
		Size int64 `json:"size"`
		// Dimension stores the dimension value.
		Dimension *Dimension `json:"dimension"`
		// Thumbnails stores the thumbnails value.
		Thumbnails []*Thumbnail `json:"thumbnails"`
		// Inactive stores the inactive value.
		Inactive bool `json:"inactive"`
		// CreatedAt stores the created at value.
		CreatedAt time.Time `json:"created_at"`
		// UpdatedAt stores the updated at value.
		UpdatedAt time.Time `json:"updated_at"`
	}
)

// FromEntity converts avatar metadata into a REST response.
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
