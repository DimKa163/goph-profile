// Package entity package
package entity

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/beevik/guid"
)

// Size image size
type Size string

const (
	// OriginalSize original size
	OriginalSize Size = "original"
	// S100x100Size 100x100 size
	S100x100Size Size = "100x100"
	// S300x300Size  300x300 size
	S300x300Size Size = "300x300"
)

// String
// String returns the string representation.
func (s Size) String() string {
	return string(s)
}

var emailRegex = regexp.MustCompile(
	`^[a-zA-Z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)+$`,
)

// Email user identifier
type Email string

// ParseEmail parse string as Email
func ParseEmail(email string) (Email, error) {
	email = strings.TrimSpace(email)
	if email == "" {
		return "", WrapError(InvalidUserIDErrorCode, "user id must not be empty", nil)
	}
	if !emailRegex.MatchString(email) {
		return "", WrapError(InvalidUserIDErrorCode, email, nil)
	}
	return Email(email), nil
}

// String
// String returns the string representation.
func (e Email) String() string {
	return string(e)
}

// AvatarID avatar identifier
type AvatarID guid.Guid

// NewAvatarID create AvatarID
func NewAvatarID() AvatarID {
	uuid := guid.New()
	return AvatarID(*uuid)
}

// String
// String returns the string representation.
func (e *AvatarID) String() string {
	uuid := guid.Guid(*e)
	return uuid.String()
}

// Scan read AvatarID
func (e *AvatarID) Scan(value any) error {
	if value == nil {
		return fmt.Errorf("avatar status can not be nil")
	}
	var str string
	switch v := value.(type) {
	case string:
		str = v
	case []byte:
		str = string(v)
	default:
		return fmt.Errorf("avatar status can not be scanned")
	}
	uuid, err := guid.ParseString(str)
	if err != nil {
		return err
	}
	*e = AvatarID(*uuid)
	return nil
}

// ParseAvatarID parse string as AvatarID
func ParseAvatarID(value string) (AvatarID, error) {
	var avatarID AvatarID
	if value == "" {
		return avatarID, WrapError(InvalidAvatarIDErrorCode, "value is empty", nil)
	}
	id, err := guid.ParseString(value)
	if err != nil {
		return avatarID, WrapError(InvalidAvatarIDErrorCode, fmt.Sprintf("value %s", value), err)
	}
	return AvatarID(*id), nil
}

// Avatar metadata
type Avatar struct {
	// ID stores the identifier.
	ID AvatarID
	// CreatedAt stores the created at value.
	CreatedAt time.Time
	// UpdatedAt stores the updated at value.
	UpdatedAt time.Time
	// Name stores the name value.
	Name string
	// UserID stores the user identifier.
	UserID Email
	// Width stores the width value.
	Width int
	// Height stores the height value.
	Height int
	// Size stores the size value.
	Size int64
	// MimeType stores the mime type value.
	MimeType string
	// Inactive stores the inactive value.
	Inactive bool
	// Images stores the images value.
	Images []*Image
	// DeletedAt stores the deleted at value.
	DeletedAt *time.Time
}

// Image metadata
type Image struct {
	// CreatedAt stores the created at value.
	CreatedAt time.Time
	// Format stores the format value.
	Format string
	// S3Key stores the s3 key value.
	S3Key string
	// MimeType stores the mime type value.
	MimeType string
	// ETag stores the e tag value.
	ETag string
	// FileSize stores the file size value.
	FileSize int64
	// Size stores the size value.
	Size Size
}

// AvatarRepository repository
//
//go:generate mockgen -source=avatar.go -destination=mocks/mock_avatar.go -package=mocks
type AvatarRepository interface {
	// FindByUserID returns the active avatar for a user.
	FindByUserID(context.Context, Email) (*Avatar, error)
	// ListByUserID returns avatars for a user.
	ListByUserID(context.Context, Email) ([]*Avatar, error)
	// Find returns avatar metadata by ID.
	Find(ctx context.Context, id AvatarID) (*Avatar, error)
	// FindImage returns image metadata by storage key.
	FindImage(ctx context.Context, avatarID AvatarID) ([]*Image, error)
	// Insert stores a new record.
	Insert(
		ctx context.Context,
		id AvatarID,
		name string,
		userID Email,
		width, height int,
		size int64,
		mimeType string,
		rows ...*Image,
	) (*Avatar, error)
	// ActivateOnlyThis activates one avatar and deactivates the others for the user.
	ActivateOnlyThis(context.Context, Email, AvatarID) error
	// InsertImage stores generated image metadata.
	InsertImage(ctx context.Context, e *Avatar) error
	// Delete removes or marks a record as deleted.
	Delete(ctx context.Context, id AvatarID) error
	// DeleteImages removes image metadata for an avatar.
	DeleteImages(ctx context.Context, id AvatarID) error
}
