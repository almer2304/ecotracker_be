package utils

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	storage_go "github.com/supabase-community/storage-go"
)

// StorageClient mengelola upload file ke Supabase Storage
type StorageClient struct {
	client        *storage_go.Client
	supabaseURL   string
	bucketPickups string
	bucketReports string
	bucketAvatars string
}

func NewStorageClient(supabaseURL, key, bucketPickups, bucketReports, bucketAvatars string) *StorageClient {
	client := storage_go.NewClient(supabaseURL+"/storage/v1", key, nil)
	return &StorageClient{
		client:        client,
		supabaseURL:   supabaseURL,
		bucketPickups: bucketPickups,
		bucketReports: bucketReports,
		bucketAvatars: bucketAvatars,
	}
}

// UploadPickupPhoto mengupload foto pickup dan mengembalikan public URL
func (s *StorageClient) UploadPickupPhoto(ctx context.Context, file multipart.File, header *multipart.FileHeader) (string, error) {
	return s.uploadFile(ctx, s.bucketPickups, "pickups", file, header)
}

// UploadReportPhoto mengupload foto laporan dan mengembalikan public URL
func (s *StorageClient) UploadReportPhoto(ctx context.Context, file multipart.File, header *multipart.FileHeader) (string, error) {
	return s.uploadFile(ctx, s.bucketReports, "reports", file, header)
}

func (s *StorageClient) uploadFile(ctx context.Context, bucket, folder string, file multipart.File, header *multipart.FileHeader) (string, error) {
	// Validasi tipe file
	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowedExts := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".webp": "image/webp",
	}

	contentType, ok := allowedExts[ext]
	if !ok {
		return "", fmt.Errorf("tipe file tidak didukung: %s", ext)
	}

	// Validasi ukuran (max 5MB)
	const maxSize = 5 * 1024 * 1024
	if header.Size > maxSize {
		return "", fmt.Errorf("ukuran file maksimal 5MB")
	}

	// Baca konten file
	data, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("gagal membaca file: %w", err)
	}

	// Generate nama file unik
	filename := fmt.Sprintf("%s/%d_%s", folder, time.Now().UnixNano(), header.Filename)

	// Upload ke Supabase
	_, err = s.client.UploadFile(bucket, filename, bytes.NewReader(data), storage_go.FileOptions{
		ContentType: &contentType,
	})
	if err != nil {
		return "", fmt.Errorf("gagal upload file: %w", err)
	}

	// Return public URL
	publicURL := fmt.Sprintf("%s/storage/v1/object/public/%s/%s", s.supabaseURL, bucket, filename)
	return publicURL, nil
}
