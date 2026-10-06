package server

import (
	"fmt"
	"strings"
)

// pictureLimits defines the constraints Xurrent places on picture_uri values.
var pictureLimits = struct {
	MaxBytes     int
	AllowedTypes []string
}{
	MaxBytes: 1024 * 1024, // 1MB — data URLs add ~33% overhead, so ~750KB actual image
	AllowedTypes: []string{
		"image/png",
		"image/jpeg",
		"image/gif",
		"image/svg+xml",
		"image/webp",
	},
}

// validatePictureURI checks a picture_uri value against Xurrent constraints.
// Returns nil if valid, or an error with a user-friendly message.
func validatePictureURI(value string) error {
	if value == "" {
		return nil // clearing the picture is allowed
	}

	// URL passthrough — any http/https URL is passed directly to Xurrent.
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		if len(value) > 2048 {
			return fmt.Errorf("picture URL is too long (%d bytes); Xurrent accepts URLs up to ~2048 characters", len(value))
		}
		return nil
	}

	// Data URL: data:[<mediatype>][;base64],<data>
	if !strings.HasPrefix(value, "data:") {
		return fmt.Errorf("picture_uri must be a URL (https://...), a data URL (data:image/...;base64,...), or empty to clear; got %q", truncateForError(value, 50))
	}

	// Parse the data URL.
	afterData := strings.TrimPrefix(value, "data:")
	commaIdx := strings.Index(afterData, ",")
	if commaIdx < 0 {
		return fmt.Errorf("invalid data URL: missing comma separator after media type")
	}
	mediaPart := afterData[:commaIdx]
	dataPart := afterData[commaIdx+1:]

	// Extract media type and check if base64.
	isBase64 := strings.Contains(mediaPart, "base64")
	mediaType := strings.TrimSuffix(mediaPart, ";base64")

	// Validate media type.
	validType := false
	for _, allowed := range pictureLimits.AllowedTypes {
		if strings.EqualFold(mediaType, allowed) {
			validType = true
			break
		}
	}
	if !validType {
		return fmt.Errorf("unsupported image type %q; Xurrent accepts: %s",
			mediaType, strings.Join(pictureLimits.AllowedTypes, ", "))
	}

	// Check base64 data size.
	if isBase64 {
		decodedSize := len(dataPart) * 3 / 4 // approximate decoded size
		if decodedSize > pictureLimits.MaxBytes {
			return fmt.Errorf("image too large (~%d KB decoded); Xurrent recommends images under %d KB for picture_uri. Use a smaller image or resize before encoding",
				decodedSize/1024, pictureLimits.MaxBytes/1024)
		}
	}

	return nil
}

// truncateForError truncates a string for error messages.
func truncateForError(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
