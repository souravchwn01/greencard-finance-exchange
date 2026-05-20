package supabase

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Storage struct {
	client  *http.Client
	baseURL string
	bucket  string
	apiKey  string
}

func NewStorage(baseURL, bucket, apiKey string) *Storage {
	return &Storage{
		client: &http.Client{Timeout: 30 * time.Second},
		baseURL: strings.TrimRight(baseURL, "/"),
		bucket:  strings.Trim(bucket, "/"),
		apiKey:  apiKey,
	}
}

func (s *Storage) UploadQuoteEvidence(ctx context.Context, quoteID uuid.UUID, fileHeader *multipart.FileHeader) (string, error) {
	if s.baseURL == "" || s.bucket == "" || s.apiKey == "" {
		return "", fmt.Errorf("supabase storage is not configured")
	}

	file, err := fileHeader.Open()
	if err != nil {
		return "", fmt.Errorf("read evidence file: %w", err)
	}
	defer file.Close()

	objectName := fmt.Sprintf("quotes/%s/%s", quoteID.String(), sanitizeFilename(fileHeader.Filename))
	uploadPath := fmt.Sprintf("%s/storage/v1/object/%s/%s", s.baseURL, url.PathEscape(s.bucket), strings.ReplaceAll(objectName, " ", "_"))

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadPath, file)
	if err != nil {
		return "", fmt.Errorf("create supabase upload request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("apikey", s.apiKey)
	req.Header.Set("Cache-Control", "public, max-age=31536000")
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("upload evidence to supabase: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("supabase upload failed: status=%d body=%s", resp.StatusCode, string(body))
	}

	var result struct {
		Key string `json:"Key"`
		Path string `json:"path"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("parse supabase upload response: %w", err)
	}

	key := result.Key
	if key == "" {
		key = result.Path
	}
	if key == "" {
		key = objectName
	}

	publicURL := fmt.Sprintf("%s/storage/v1/object/public/%s/%s", s.baseURL, s.bucket, strings.ReplaceAll(key, " ", "_"))
	return publicURL, nil
}

func sanitizeFilename(filename string) string {
	filename = strings.TrimSpace(filename)
	filename = strings.ReplaceAll(filename, "\\", "_")
	filename = strings.ReplaceAll(filename, "/", "_")
	filename = strings.ReplaceAll(filename, " ", "_")
	filename = strings.ReplaceAll(filename, "..", "_")
	return filename
}
