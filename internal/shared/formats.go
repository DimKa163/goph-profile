// Package shared contains common helpers used across profile packages.
package shared

import (
	"fmt"
	"strings"
)

// Formats maps supported image formats to MIME types.
var Formats = []string{
	"png",
	"jpeg",
	"webp",
}

// ContentType returns the MIME type for a supported format.
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
