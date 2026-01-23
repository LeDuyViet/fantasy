package google

import (
	"context"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"time"

	"google.golang.org/genai"
)

// FileUploader is a utility for uploading files to Google Gemini File API.
type FileUploader struct {
	client *genai.Client
}

// NewFileUploader creates a new FileUploader.
func NewFileUploader(ctx context.Context, apiKey string) (*FileUploader, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create genai client: %w", err)
	}
	return &FileUploader{client: client}, nil
}

// Upload uploads a file to the Gemini File API and waits for it to become ACTIVE.
// It returns the uploaded file info, a cleanup function to delete the file, and any error.
func (u *FileUploader) Upload(ctx context.Context, path string) (*genai.File, func(), error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	displayName := filepath.Base(path)

	mimeType := mime.TypeByExtension(filepath.Ext(path))
	if mimeType == "" {
		switch filepath.Ext(path) {
		case ".pdf":
			mimeType = "application/pdf"
		case ".jpg", ".jpeg":
			mimeType = "image/jpeg"
		case ".png":
			mimeType = "image/png"
		case ".txt":
			mimeType = "text/plain"
		}
	}

	uploadResult, err := u.client.Files.Upload(ctx, f, &genai.UploadFileConfig{
		DisplayName: displayName,
		MIMEType:    mimeType,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to upload file: %w", err)
	}

	cleanup := func() {
		_, _ = u.client.Files.Delete(context.Background(), uploadResult.Name, nil)
	}

	// Poll for active state
	for {
		file, err := u.client.Files.Get(ctx, uploadResult.Name, nil)
		if err != nil {
			cleanup()
			return nil, nil, fmt.Errorf("failed to check file status: %w", err)
		}

		if file.State == "ACTIVE" {
			return file, cleanup, nil
		}

		if file.State == "FAILED" {
			cleanup()
			return nil, nil, fmt.Errorf("file processing failed")
		}

		select {
		case <-ctx.Done():
			cleanup()
			return nil, nil, ctx.Err()
		case <-time.After(1 * time.Second):
		}
	}
}
