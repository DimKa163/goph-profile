// Package assets collection default images
package assets

import _ "embed"

// DefaultAvatarOriginalWeb original webp
//
//go:embed original_default.webp
var DefaultAvatarOriginalWeb []byte

// DefaultAvatarOriginalPng original png
//
//go:embed original_default.png
var DefaultAvatarOriginalPng []byte

// DefaultAvatarOriginalJpeg original jpg
//
//go:embed original_default.jpeg
var DefaultAvatarOriginalJpeg []byte

// DefaultAvatarS300Webp 300x300 webp
//
//go:embed 300x300_default.webp
var DefaultAvatarS300Webp []byte

// DefaultAvatarS300Png  300x300 png
//
//go:embed 300x300_default.png
var DefaultAvatarS300Png []byte

// DefaultAvatarS300Jpeg 300x300 jpg
//
//go:embed 300x300_default.jpeg
var DefaultAvatarS300Jpeg []byte

// DefaultAvatarS100Webp 100x100 webp
//
//go:embed 100x100_default.webp
var DefaultAvatarS100Webp []byte

// DefaultAvatarS100Png 100x100 png
//
//go:embed 100x100_default.png
var DefaultAvatarS100Png []byte

// DefaultAvatarS100Jpeg 100x100 jpg
//
//go:embed 100x100_default.jpeg
var DefaultAvatarS100Jpeg []byte

// DefaultAvatar default avatar
type DefaultAvatar struct {
	// Format is the default avatar image format.
	Format string
	// Size is the default avatar image size.
	Size string
	// MimeType is the default avatar MIME type.
	MimeType string
	// ETag is the default avatar entity tag.
	ETag string
	// Data contains the default avatar bytes.
	Data []byte
}

// DefaultAvatars slice of default avatars
var DefaultAvatars = []*DefaultAvatar{
	{
		Format:   "jpeg",
		Size:     "original",
		MimeType: "image/jpeg",
		ETag:     `"default-avatar-jpeg-original"`,
		Data:     DefaultAvatarOriginalJpeg,
	},
	{
		Format:   "jpeg",
		Size:     "300x300",
		MimeType: "image/jpeg",
		ETag:     `"default-avatar-jpeg-300x300"`,
		Data:     DefaultAvatarS300Jpeg,
	},
	{
		Format:   "jpeg",
		Size:     "100x100",
		MimeType: "image/jpeg",
		ETag:     `"default-avatar-jpeg-100x100"`,
		Data:     DefaultAvatarS100Jpeg,
	},
	{
		Format:   "png",
		Size:     "original",
		MimeType: "image/png",
		ETag:     `"default-avatar-png-original"`,
		Data:     DefaultAvatarOriginalPng,
	},
	{
		Format:   "png",
		Size:     "300x300",
		MimeType: "image/png",
		ETag:     `"default-avatar-png-300x300"`,
		Data:     DefaultAvatarS300Png,
	},
	{
		Format:   "png",
		Size:     "100x100",
		MimeType: "image/png",
		ETag:     `"default-avatar-png-100x100"`,
		Data:     DefaultAvatarS100Png,
	},
	{
		Format:   "webp",
		Size:     "original",
		MimeType: "image/webp",
		ETag:     `"default-avatar-webp-original"`,
		Data:     DefaultAvatarOriginalWeb,
	},
	{
		Format:   "webp",
		Size:     "300x300",
		MimeType: "image/webp",
		ETag:     `"default-avatar-webp-300x300"`,
		Data:     DefaultAvatarS300Webp,
	},
	{
		Format:   "webp",
		Size:     "100x100",
		MimeType: "image/webp",
		ETag:     `"default-avatar-webp-100x100"`,
		Data:     DefaultAvatarS100Webp,
	},
}
