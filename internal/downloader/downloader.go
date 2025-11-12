package downloader

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type Downloader struct {
	OutputDir string
}

type Meta struct {
	ETag         string `json:"etag,omitempty"`
	LastModified string `json:"last_modified,omitempty"`
	SourceURL    string `json:"source_url,omitempty"`
	UpdatedAt    string `json:"updated_at,omitempty"`
}

func New(outputDir string) *Downloader {
	return &Downloader{
		OutputDir: outputDir,
	}
}

func (d *Downloader) Download(ctx context.Context, outPath, url string) (updated bool, status string, err error) {
	fullPath := filepath.Join(d.OutputDir, outPath)
	metaPath := fullPath + ".meta.json"

	var meta Meta
	b, err := os.ReadFile(metaPath)
	if err == nil {
		if err := json.Unmarshal(b, &meta); err != nil {
			return false, "", fmt.Errorf("unmarshal meta: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", "svg-logo-downloader/1.0")
	if meta.ETag != "" {
		req.Header.Set("If-None-Match", meta.ETag)
	}
	if meta.LastModified != "" {
		req.Header.Set("If-Modified-Since", meta.LastModified)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return false, "", fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		return false, "304 not modified", nil
	}

	if resp.StatusCode != http.StatusOK {
		return false, "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	err = os.MkdirAll(filepath.Dir(fullPath), 0o755)

	if err != nil {
		return false, "", fmt.Errorf("create directories: %w", err)
	}

	tmp := fullPath + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return false, "", fmt.Errorf("create temp file: %w", err)
	}

	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		return false, "", fmt.Errorf("write to temp file: %w", err)
	}
	f.Close()

	if err := os.Rename(tmp, fullPath); err != nil {
		return false, "", fmt.Errorf("rename temp file: %w", err)
	}

	meta.ETag = resp.Header.Get("ETag")
	meta.LastModified = resp.Header.Get("Last-Modified")
	meta.SourceURL = url
	meta.UpdatedAt = time.Now().Format(time.RFC3339)

	b, err = json.MarshalIndent(&meta, "", "  ")
	if err == nil {
		_ = os.WriteFile(metaPath, b, 0o644)
	}
	return true, resp.Status, nil
}
