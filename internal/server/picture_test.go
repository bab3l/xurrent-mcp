package server

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidatePictureURI_Empty(t *testing.T) {
	require.NoError(t, validatePictureURI(""))
}

func TestValidatePictureURI_HTTPURL(t *testing.T) {
	require.NoError(t, validatePictureURI("https://example.com/icon.png"))
	require.NoError(t, validatePictureURI("http://cdn.example.com/images/product.jpg"))
}

func TestValidatePictureURI_URLTooLong(t *testing.T) {
	long := "https://example.com/" + strings.Repeat("x", 2048)
	require.Error(t, validatePictureURI(long))
	require.Contains(t, validatePictureURI(long).Error(), "too long")
}

func TestValidatePictureURI_ValidDataURLs(t *testing.T) {
	valid := []string{
		"data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==",
		"data:image/jpeg;base64,/9j/4AAQSkZJRgABAQAAAQABAAD/2wBDAAgGBgcGBQg==",
		"data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7",
		"data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciPjwvc3ZnPg==",
		"data:image/webp;base64,UklGRiQAAABXRUJQVlA4IBgAAAAwAQCdASoBAAEAAIAOJaQAA3AA/v9gGgA=",
	}
	for i, v := range valid {
		t.Run(fmt.Sprintf("valid_%d", i), func(t *testing.T) {
			require.NoError(t, validatePictureURI(v))
		})
	}
}

func TestValidatePictureURI_InvalidFormat(t *testing.T) {
	err := validatePictureURI("data:image/bmp;base64,AAAA")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported image type")
	require.Contains(t, err.Error(), "image/bmp")
}

func TestValidatePictureURI_NotADataURL(t *testing.T) {
	err := validatePictureURI("just-some-text")
	require.Error(t, err)
	require.Contains(t, err.Error(), "must be a URL")
}

func TestValidatePictureURI_MissingComma(t *testing.T) {
	err := validatePictureURI("data:image/png;base64")
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing comma")
}

func TestValidatePictureURI_TooLarge(t *testing.T) {
	// Create a data URL that would decode to > 1MB.
	large := "data:image/png;base64," + strings.Repeat("A", 1400000) // ~1MB decoded
	err := validatePictureURI(large)
	require.Error(t, err)
	require.Contains(t, err.Error(), "too large")
}

func TestValidatePictureURI_AllAllowedTypes(t *testing.T) {
	for _, mt := range pictureLimits.AllowedTypes {
		require.NoError(t, validatePictureURI("data:"+mt+";base64,AAAA"))
	}
}

func TestValidatePictureURI_CaseInsensitive(t *testing.T) {
	require.NoError(t, validatePictureURI("data:IMAGE/PNG;base64,AAAA"))
	require.NoError(t, validatePictureURI("data:Image/Png;base64,AAAA"))
}

func TestTruncateForError(t *testing.T) {
	require.Equal(t, "hello", truncateForError("hello", 10))
	require.Equal(t, "hello worl...", truncateForError("hello world this is long", 10))
	require.Equal(t, "", truncateForError("", 10))
}
