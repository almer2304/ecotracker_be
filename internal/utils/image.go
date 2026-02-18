package utils

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"mime/multipart"
	"strings"

	"github.com/disintegration/imaging"

	// Register formats explicitly to prevent "unknown format" error
	_ "image/jpeg"
	_ "image/png"
	_ "golang.org/x/image/webp"
)

const (
	MaxImageWidth  = 1920
	MaxImageHeight = 1080
	JPEGQuality    = 85
)

// AllowedImageTypes lists accepted MIME types for upload
var AllowedImageTypes = map[string]bool{
	"image/jpeg": true,
	"image/jpg":  true,
	"image/png":  true,
	"image/webp": true,
}

// ProcessImage reads, resizes, and returns the image as JPEG bytes.
// Supports WebP, JPG, PNG input formats.
func ProcessImage(fileHeader *multipart.FileHeader) ([]byte, string, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return nil, "", fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer file.Close()

	// Validate content type
	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		// Try to infer from filename
		ext := strings.ToLower(fileHeader.Filename)
		switch {
		case strings.HasSuffix(ext, ".jpg") || strings.HasSuffix(ext, ".jpeg"):
			contentType = "image/jpeg"
		case strings.HasSuffix(ext, ".png"):
			contentType = "image/png"
		case strings.HasSuffix(ext, ".webp"):
			contentType = "image/webp"
		default:
			contentType = "application/octet-stream"
		}
	}

	if !AllowedImageTypes[contentType] {
		return nil, "", fmt.Errorf("unsupported image format: %s. Allowed: JPG, PNG, WebP", contentType)
	}

	// Decode image (imaging handles all registered formats)
	img, err := imaging.Decode(file, imaging.AutoOrientation(true))
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode image: %w", err)
	}

	// Resize if needed while preserving aspect ratio
	img = resizeIfNeeded(img)

	// Encode to JPEG
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: JPEGQuality}); err != nil {
		return nil, "", fmt.Errorf("failed to encode image to JPEG: %w", err)
	}

	return buf.Bytes(), "image/jpeg", nil
}

func resizeIfNeeded(img image.Image) image.Image {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	if w <= MaxImageWidth && h <= MaxImageHeight {
		return img
	}

	return imaging.Fit(img, MaxImageWidth, MaxImageHeight, imaging.Lanczos)
}

// PNGBytes encodes an image to PNG bytes (for flexibility)
func PNGBytes(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("failed to encode image to PNG: %w", err)
	}
	return buf.Bytes(), nil
}
