// Package queue provides SQLite-based persistent queue for tracking log file uploads.
// It manages the lifecycle of log files from local disk to S3 storage.
package queue

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Status represents the upload status of a log file.
type Status string

const (
	StatusPending         Status = "pending"
	StatusUploading       Status = "uploading"
	StatusUploaded        Status = "uploaded"
	StatusFailed          Status = "failed"
	StatusFailedPermanent Status = "failed_permanent"
)

// LogUpload represents a log file entry in the queue.
type LogUpload struct {
	ID            int64
	LocalFilePath string
	S3Path        string
	Status        Status
	RetryCount    int
	LastError     string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	UploadedAt    *time.Time
}

// Queue manages the SQLite-based upload queue.
type Queue struct {
	db *sql.DB
}

// New creates a new queue instance with the given database path.
func New(dbPath string) (*Queue, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite db: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping sqlite db: %w", err)
	}

	q := &Queue{db: db}
	if err := q.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate db: %w", err)
	}

	return q, nil
}

// migrate creates the necessary tables.
func (q *Queue) migrate() error {
	query := `
	CREATE TABLE IF NOT EXISTS log_uploads (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		local_file_path TEXT NOT NULL UNIQUE,
		s3_path TEXT,
		status TEXT NOT NULL DEFAULT 'pending',
		retry_count INTEGER DEFAULT 0,
		last_error TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		uploaded_at DATETIME
	);

	CREATE INDEX IF NOT EXISTS idx_status ON log_uploads(status);
	CREATE INDEX IF NOT EXISTS idx_created_at ON log_uploads(created_at);
	`

	_, err := q.db.Exec(query)
	return err
}

// Enqueue adds a new log file to the queue.
// Returns error if file already exists in queue.
func (q *Queue) Enqueue(localFilePath string) (*LogUpload, error) {
	query := `
		INSERT INTO log_uploads (local_file_path, status)
		VALUES (?, 'pending')
		ON CONFLICT(local_file_path) DO NOTHING
		RETURNING id, local_file_path, s3_path, status, retry_count, last_error, created_at, updated_at, uploaded_at
	`

	var upload LogUpload
	var uploadedAt sql.NullTime

	err := q.db.QueryRow(query, localFilePath).Scan(
		&upload.ID,
		&upload.LocalFilePath,
		&upload.S3Path,
		&upload.Status,
		&upload.RetryCount,
		&upload.LastError,
		&upload.CreatedAt,
		&upload.UpdatedAt,
		&uploadedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			// File already exists in queue, return existing
			return q.GetByPath(localFilePath)
		}
		return nil, fmt.Errorf("failed to enqueue: %w", err)
	}

	if uploadedAt.Valid {
		upload.UploadedAt = &uploadedAt.Time
	}

	return &upload, nil
}

// GetByPath retrieves a log upload by its local file path.
func (q *Queue) GetByPath(localFilePath string) (*LogUpload, error) {
	query := `
		SELECT id, local_file_path, s3_path, status, retry_count, last_error, 
		       created_at, updated_at, uploaded_at
		FROM log_uploads
		WHERE local_file_path = ?
	`

	var upload LogUpload
	var uploadedAt sql.NullTime

	err := q.db.QueryRow(query, localFilePath).Scan(
		&upload.ID,
		&upload.LocalFilePath,
		&upload.S3Path,
		&upload.Status,
		&upload.RetryCount,
		&upload.LastError,
		&upload.CreatedAt,
		&upload.UpdatedAt,
		&uploadedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get by path: %w", err)
	}

	if uploadedAt.Valid {
		upload.UploadedAt = &uploadedAt.Time
	}

	return &upload, nil
}

// GetPending retrieves all pending uploads (not yet processed).
func (q *Queue) GetPending(limit int) ([]*LogUpload, error) {
	query := `
		SELECT id, local_file_path, s3_path, status, retry_count, last_error, 
		       created_at, updated_at, uploaded_at
		FROM log_uploads
		WHERE status IN ('pending', 'failed')
		  AND retry_count < 3
		ORDER BY created_at ASC
		LIMIT ?
	`

	return q.queryUploads(query, limit)
}

// GetFailedPermanent retrieves all permanently failed uploads (retry count >= 3).
func (q *Queue) GetFailedPermanent(limit int) ([]*LogUpload, error) {
	query := `
		SELECT id, local_file_path, s3_path, status, retry_count, last_error, 
		       created_at, updated_at, uploaded_at
		FROM log_uploads
		WHERE status = 'failed_permanent'
		ORDER BY created_at ASC
		LIMIT ?
	`

	return q.queryUploads(query, limit)
}

// GetUploaded retrieves all successfully uploaded files.
func (q *Queue) GetUploaded(olderThan time.Duration, limit int) ([]*LogUpload, error) {
	cutoff := time.Now().Add(-olderThan)
	query := `
		SELECT id, local_file_path, s3_path, status, retry_count, last_error, 
		       created_at, updated_at, uploaded_at
		FROM log_uploads
		WHERE status = 'uploaded'
		  AND uploaded_at < ?
		ORDER BY uploaded_at ASC
		LIMIT ?
	`

	return q.queryUploads(query, cutoff, limit)
}

// queryUploads executes a query and returns LogUpload slice.
func (q *Queue) queryUploads(query string, args ...interface{}) ([]*LogUpload, error) {
	rows, err := q.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query: %w", err)
	}
	defer rows.Close()

	var uploads []*LogUpload
	for rows.Next() {
		var upload LogUpload
		var uploadedAt sql.NullTime

		err := rows.Scan(
			&upload.ID,
			&upload.LocalFilePath,
			&upload.S3Path,
			&upload.Status,
			&upload.RetryCount,
			&upload.LastError,
			&upload.CreatedAt,
			&upload.UpdatedAt,
			&uploadedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		if uploadedAt.Valid {
			upload.UploadedAt = &uploadedAt.Time
		}

		uploads = append(uploads, &upload)
	}

	return uploads, rows.Err()
}

// MarkUploading marks a file as currently being uploaded.
func (q *Queue) MarkUploading(id int64) error {
	query := `
		UPDATE log_uploads
		SET status = 'uploading', updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	_, err := q.db.Exec(query, id)
	return err
}

// MarkUploaded marks a file as successfully uploaded to S3.
func (q *Queue) MarkUploaded(id int64, s3Path string) error {
	query := `
		UPDATE log_uploads
		SET status = 'uploaded', 
		    s3_path = ?, 
		    uploaded_at = CURRENT_TIMESTAMP,
		    updated_at = CURRENT_TIMESTAMP,
		    last_error = NULL
		WHERE id = ?
	`
	_, err := q.db.Exec(query, s3Path, id)
	return err
}

// MarkFailed marks a file as failed and increments retry count.
// If retry count reaches 3, status becomes 'failed_permanent'.
func (q *Queue) MarkFailed(id int64, errorMsg string) error {
	query := `
		UPDATE log_uploads
		SET status = CASE 
				WHEN retry_count + 1 >= 3 THEN 'failed_permanent'
				ELSE 'failed'
			END,
		    retry_count = retry_count + 1,
		    last_error = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	_, err := q.db.Exec(query, errorMsg, id)
	return err
}

// ResetForRetry resets a failed_permanent file for another retry attempt.
// Used for manual retry operations.
func (q *Queue) ResetForRetry(id int64) error {
	query := `
		UPDATE log_uploads
		SET status = 'pending',
		    retry_count = 0,
		    last_error = NULL,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	_, err := q.db.Exec(query, id)
	return err
}

// Delete removes an entry from the queue.
func (q *Queue) Delete(id int64) error {
	query := `DELETE FROM log_uploads WHERE id = ?`
	_, err := q.db.Exec(query, id)
	return err
}

// GetStats returns statistics about the queue.
func (q *Queue) GetStats() (map[Status]int64, error) {
	query := `
		SELECT status, COUNT(*) as count
		FROM log_uploads
		GROUP BY status
	`

	rows, err := q.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make(map[Status]int64)
	for rows.Next() {
		var status Status
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		stats[status] = count
	}

	return stats, rows.Err()
}

// Close closes the database connection.
func (q *Queue) Close() error {
	return q.db.Close()
}
