package utils

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"
)

type StorageClient struct {
	supabaseURL    string
	serviceRoleKey string
	httpClient     *http.Client
}

func NewStorageClient(supabaseURL, serviceRoleKey string) *StorageClient {
	return &StorageClient{
		supabaseURL:    supabaseURL,
		serviceRoleKey: serviceRoleKey,
		httpClient:     &http.Client{Timeout: 30 * time.Second},
	}
}

// UploadImage uploads bytes to Supabase Storage and returns the public URL.
// filePath example: "pickups/uuid-filename.jpg"
func (s *StorageClient) UploadImage(bucket, filePath string, data []byte, contentType string) (string, error) {
	url := fmt.Sprintf("%s/storage/v1/object/%s/%s", s.supabaseURL, bucket, filePath)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("failed to create storage request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.serviceRoleKey)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("x-upsert", "true") // overwrite if exists

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("storage request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("storage upload failed (status %d): %s", resp.StatusCode, string(body))
	}

	// Return the public URL
	publicURL := fmt.Sprintf("%s/storage/v1/object/public/%s/%s", s.supabaseURL, bucket, filePath)
	return publicURL, nil
}

// DeleteImage deletes a file from Supabase Storage
func (s *StorageClient) DeleteImage(bucket, filePath string) error {
	url := fmt.Sprintf("%s/storage/v1/object/%s/%s", s.supabaseURL, bucket, filePath)

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.serviceRoleKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("delete request failed: %w", err)
	}
	defer resp.Body.Close()

	return nil
}
