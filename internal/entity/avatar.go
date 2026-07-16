package entity

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/beevik/guid"
)

type AvatarStatus int

const (
	UploadingAvatarStatus AvatarStatus = iota
	PendingAvatarStatus
	ProcessingAvatarStatus
	CompletedAvatarStatus
)

func (s AvatarStatus) String() string {
	return []string{"uploading", "pending", "processing", "completed"}[s]
}

func (s *AvatarStatus) Scan(value any) error {
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
	switch str {
	case "uploading":
		*s = UploadingAvatarStatus
	case "pending":
		*s = PendingAvatarStatus
	case "processing":
		*s = ProcessingAvatarStatus
	case "completed":
		*s = CompletedAvatarStatus
	}
	return nil
}

type Size string

const (
	OriginalSize Size = "original"
	S100x100Size Size = "100x100"
	S300x300Size Size = "300x300"
)

func (s Size) String() string {
	return string(s)
}

var emailRegex = regexp.MustCompile(
	`^[a-zA-Z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)+$`,
)

type Email string

func ParseEmail(email string) (Email, error) {
	email = strings.TrimSpace(email)
	if email == "" {
		return "", Error(InvalidUserIDErrorCode, "user id must not be empty")
	}
	if !emailRegex.MatchString(email) {
		return "", Error(InvalidUserIDErrorCode, email)
	}
	return Email(email), nil
}

func (e Email) String() string {
	return string(e)
}

type AvatarID guid.Guid

func NewAvatarID() AvatarID {
	uuid := guid.New()
	return AvatarID(*uuid)
}

func (e *AvatarID) String() string {
	uuid := guid.Guid(*e)
	return uuid.String()
}

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

func ParseAvatarID(value string) (AvatarID, error) {
	var avatarID AvatarID
	id, err := guid.ParseString(value)
	if err != nil {
		return avatarID, Error(InvalidAvatarIDErrorCode, value, err)
	}
	return AvatarID(*id), nil
}

type Avatar struct {
	ID        AvatarID
	CreatedAt time.Time
	UpdatedAt time.Time
	Name      string
	UserID    Email
	Width     int
	Height    int
	Size      int64
	MimeType  string
	Inactive  bool
	Images    []*Image
	DeletedAt *time.Time
}

type Image struct {
	CreatedAt time.Time
	Format    string
	S3Key     string
	MimeType  string
	ETag      string
	FileSize  int64
	Size      Size
}

//go:generate mockgen -source=avatar.go -destination=mocks/mock_avatar.go -package=mocks
type AvatarRepository interface {
	FindByUserID(context.Context, Email) (*Avatar, error)
	ListByUserID(context.Context, Email) ([]*Avatar, error)
	Find(ctx context.Context, id AvatarID) (*Avatar, error)
	FindImage(ctx context.Context, avatarID AvatarID) ([]*Image, error)
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
	ActivateOnlyThis(context.Context, Email, AvatarID) error
	InsertImage(ctx context.Context, e *Avatar) error
	Delete(ctx context.Context, id AvatarID) error
	DeleteImages(ctx context.Context, id AvatarID) error
}
