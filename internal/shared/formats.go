package shared

import (
	"fmt"
	"strings"
)

var Formats = []string{
	"png",
	"jpeg",
	"webp",
}

func ContentType(format string) (string, error) {
	format = strings.ToLower(strings.TrimPrefix(format, "."))

	switch format {
	case "jpg", "jpeg":
		return "image/jpeg", nil
	case "png":
		return "image/png", nil
	case "webp":
		return "image/webp", nil
	default:
		return "", fmt.Errorf("unsupported image format: %q", format)
	}
}
