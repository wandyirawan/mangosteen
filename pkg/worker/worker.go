// Package worker provides background workers for processing log file uploads.
// It handles uploading log files to S3 (Garage) and retry logic with exponential backoff.
package worker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/rs/zerolog/log"

	"mangosteen/pkg/queue"
)

// Config holds configuration for the upload worker.
type Config struct {
	// S3/Garage configuration
	Endpoint  string // e.g., "garage:3900"
	AccessKey string
	SecretKey string
	Bucket    string
	Region    string // e.g., "us-east-1"
	UseSSL    bool

	// Worker configuration
	CheckInterval time.Duration // How often to check for new files
	MaxRetries    int
	UploadTimeout time.Duration

	// S3 path prefix
	S3Prefix string // e.g., "logs/mangosteen/"
}

// UploadWorker handles uploading log files to S3.
type UploadWorker struct {
	config Config
	queue  *queue.Queue
	client *minio.Client
	ctx    context.Context
	cancel context.CancelFunc
}

// NewUploadWorker creates a new upload worker instance.
func NewUploadWorker(cfg Config, q *queue.Queue) (*UploadWorker, error) {
	// Initialize MinIO client
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create s3 client: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &UploadWorker{
		config: cfg,
		queue:  q,
		client: client,
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

// Start begins the background worker.
func (w *UploadWorker) Start() {
	log.Info().
		Str("endpoint", w.config.Endpoint).
		Str("bucket", w.config.Bucket).
		Dur("interval", w.config.CheckInterval).
		Msg("Starting upload worker")

	go w.run()
}

// Stop stops the background worker.
func (w *UploadWorker) Stop() {
	log.Info().Msg("Stopping upload worker")
	w.cancel()
}

// run is the main worker loop.
func (w *UploadWorker) run() {
	ticker := time.NewTicker(w.config.CheckInterval)
	defer ticker.Stop()

	// Process immediately on start
	w.processBatch()

	for {
		select {
		case <-ticker.C:
			w.processBatch()
		case <-w.ctx.Done():
			return
		}
	}
}

// processBatch processes a batch of pending uploads.
func (w *UploadWorker) processBatch() {
	// Get pending uploads
	uploads, err := w.queue.GetPending(10)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get pending uploads")
		return
	}

	if len(uploads) == 0 {
		return
	}

	log.Info().Int("count", len(uploads)).Msg("Processing upload batch")

	for _, upload := range uploads {
		// Check if file is "closed" (older than 1 hour)
		if !w.isFileClosed(upload.LocalFilePath) {
			log.Debug().
				Str("file", upload.LocalFilePath).
				Msg("Skipping active file")
			continue
		}

		// Mark as uploading
		if err := w.queue.MarkUploading(upload.ID); err != nil {
			log.Error().Err(err).Int64("id", upload.ID).Msg("Failed to mark uploading")
			continue
		}

		// Upload to S3
		s3Path, err := w.uploadToS3(upload.LocalFilePath)
		if err != nil {
			log.Error().
				Err(err).
				Str("file", upload.LocalFilePath).
				Msg("Upload failed")

			if markErr := w.queue.MarkFailed(upload.ID, err.Error()); markErr != nil {
				log.Error().Err(markErr).Int64("id", upload.ID).Msg("Failed to mark failed")
			}
			continue
		}

		// Mark as uploaded
		if err := w.queue.MarkUploaded(upload.ID, s3Path); err != nil {
			log.Error().Err(err).Int64("id", upload.ID).Msg("Failed to mark uploaded")
			continue
		}

		// Delete local file
		if err := os.Remove(upload.LocalFilePath); err != nil {
			log.Error().
				Err(err).
				Str("file", upload.LocalFilePath).
				Msg("Failed to delete local file")
			// Continue anyway - file is already in S3
		}

		log.Info().
			Str("file", upload.LocalFilePath).
			Str("s3_path", s3Path).
			Msg("Upload successful")
	}
}

// isFileClosed checks if a log file is closed (older than 1 hour).
func (w *UploadWorker) isFileClosed(filePath string) bool {
	stat, err := os.Stat(filePath)
	if err != nil {
		// If we can't stat the file, consider it "closed" (will fail upload anyway)
		return true
	}

	// File is closed if it hasn't been modified in the last hour
	age := time.Since(stat.ModTime())
	return age > time.Hour
}

// uploadToS3 uploads a file to S3 and returns the S3 path.
func (w *UploadWorker) uploadToS3(localPath string) (string, error) {
	// Check if file exists
	if _, err := os.Stat(localPath); err != nil {
		return "", fmt.Errorf("file not found: %w", err)
	}

	// Generate S3 key (path)
	filename := filepath.Base(localPath)
	s3Key := w.config.S3Prefix + filename

	// Upload with timeout
	ctx, cancel := context.WithTimeout(w.ctx, w.config.UploadTimeout)
	defer cancel()

	// Upload file
	_, err := w.client.FPutObject(ctx, w.config.Bucket, s3Key, localPath, minio.PutObjectOptions{})
	if err != nil {
		return "", fmt.Errorf("s3 upload failed: %w", err)
	}

	// Return full S3 path
	s3Path := fmt.Sprintf("s3://%s/%s", w.config.Bucket, s3Key)
	return s3Path, nil
}

// RetryFailedPermanent attempts to retry permanently failed uploads.
// This can be triggered manually via HTTP endpoint.
func (w *UploadWorker) RetryFailedPermanent() error {
	uploads, err := w.queue.GetFailedPermanent(10)
	if err != nil {
		return fmt.Errorf("failed to get failed permanent: %w", err)
	}

	log.Info().Int("count", len(uploads)).Msg("Retrying failed permanent uploads")

	for _, upload := range uploads {
		// Check if file still exists
		if _, err := os.Stat(upload.LocalFilePath); err != nil {
			log.Warn().
				Str("file", upload.LocalFilePath).
				Msg("File no longer exists, removing from queue")
			w.queue.Delete(upload.ID)
			continue
		}

		// Reset for retry
		if err := w.queue.ResetForRetry(upload.ID); err != nil {
			log.Error().Err(err).Int64("id", upload.ID).Msg("Failed to reset for retry")
			continue
		}

		log.Info().
			Str("file", upload.LocalFilePath).
			Msg("Reset for retry")
	}

	return nil
}

// GetStats returns worker statistics.
func (w *UploadWorker) GetStats() (map[queue.Status]int64, error) {
	return w.queue.GetStats()
}
