package assets

import _ "embed"

//go:embed original_default.webp
var DefaultAvatarOriginalWeb []byte

//go:embed original_default.png
var DefaultAvatarOriginalPng []byte

//go:embed original_default.jpeg
var DefaultAvatarOriginalJpeg []byte

//go:embed 300x300_default.webp
var DefaultAvatarS300Webp []byte

//go:embed 300x300_default.png
var DefaultAvatarS300Png []byte

//go:embed 300x300_default.jpeg
var DefaultAvatarS300Jpeg []byte

//go:embed 100x100_default.webp
var DefaultAvatarS100Webp []byte

//go:embed 100x100_default.png
var DefaultAvatarS100Png []byte

//go:embed 100x100_default.jpeg
var DefaultAvatarS100Jpeg []byte

type DefaultAvatar struct {
	Format   string
	Size     string
	MimeType string
	ETag     string
	Data     []byte
}

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
