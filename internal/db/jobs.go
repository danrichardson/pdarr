package db

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type JobStatus string

const (
	JobPending    JobStatus = "pending"
	JobRunning    JobStatus = "running"
	JobDone       JobStatus = "done"
	JobFailed     JobStatus = "failed"
	JobCancelled  JobStatus = "cancelled"
	JobSkipped    JobStatus = "skipped"
)

type Job struct {
	ID             int64
	DirectoryID    sql.NullInt64
	SourcePath     string
	SourceSize     int64
	SourceCodec    string
	SourceDuration float64
	SourceBitrate  int64
	OutputPath     sql.NullString
	OutputSize     sql.NullInt64
	EncoderUsed    sql.NullString
	Status         JobStatus
	Priority       int
	ErrorMessage   sql.NullString
	Progress       float64
	BytesSaved     sql.NullInt64
	StartedAt      sql.NullTime
	FinishedAt     sql.NullTime
	CreatedAt      time.Time
}

func (db *DB) InsertJob(j *Job) (int64, error) {
	res, err := db.conn.Exec(`
		INSERT INTO jobs
		  (directory_id, source_path, source_size, source_codec, source_duration, source_bitrate, status, priority)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		j.DirectoryID, j.SourcePath, j.SourceSize, j.SourceCodec,
		j.SourceDuration, j.SourceBitrate, j.Status, j.Priority)
	if err != nil {
		return 0, fmt.Errorf("insert job: %w", err)
	}
	return res.LastInsertId()
}

func (db *DB) GetJob(id int64) (*Job, error) {
	j := &Job{}
	err := db.conn.QueryRow(`
		SELECT id, directory_id, source_path, source_size, source_codec, source_duration,
		       source_bitrate, output_path, output_size, encoder_used, status, priority,
		       error_message, progress, bytes_saved, started_at, finished_at, created_at
		FROM jobs WHERE id = ?`, id).Scan(
		&j.ID, &j.DirectoryID, &j.SourcePath, &j.SourceSize, &j.SourceCodec, &j.SourceDuration,
		&j.SourceBitrate, &j.OutputPath, &j.OutputSize, &j.EncoderUsed, &j.Status, &j.Priority,
		&j.ErrorMessage, &j.Progress, &j.BytesSaved, &j.StartedAt, &j.FinishedAt, &j.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get job: %w", err)
	}
	return j, nil
}

// NextPendingJob returns the next job to process ordered by priority desc, created_at asc.
func (db *DB) NextPendingJob() (*Job, error) {
	j := &Job{}
	err := db.conn.QueryRow(`
		SELECT id, directory_id, source_path, source_size, source_codec, source_duration,
		       source_bitrate, output_path, output_size, encoder_used, status, priority,
		       error_message, progress, bytes_saved, started_at, finished_at, created_at
		FROM jobs WHERE status = 'pending'
		ORDER BY priority DESC, created_at ASC
		LIMIT 1`).Scan(
		&j.ID, &j.DirectoryID, &j.SourcePath, &j.SourceSize, &j.SourceCodec, &j.SourceDuration,
		&j.SourceBitrate, &j.OutputPath, &j.OutputSize, &j.EncoderUsed, &j.Status, &j.Priority,
		&j.ErrorMessage, &j.Progress, &j.BytesSaved, &j.StartedAt, &j.FinishedAt, &j.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("next pending job: %w", err)
	}
	return j, nil
}

func (db *DB) ListJobs(status JobStatus, limit, offset int) ([]*Job, error) {
	query := `
		SELECT id, directory_id, source_path, source_size, source_codec, source_duration,
		       source_bitrate, output_path, output_size, encoder_used, status, priority,
		       error_message, progress, bytes_saved, started_at, finished_at, created_at
		FROM jobs`
	args := []any{}

	if status != "" {
		query += " WHERE status = ?"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC"
	if limit > 0 {
		query += " LIMIT ? OFFSET ?"
		args = append(args, limit, offset)
	}

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*Job
	for rows.Next() {
		j := &Job{}
		if err := rows.Scan(
			&j.ID, &j.DirectoryID, &j.SourcePath, &j.SourceSize, &j.SourceCodec, &j.SourceDuration,
			&j.SourceBitrate, &j.OutputPath, &j.OutputSize, &j.EncoderUsed, &j.Status, &j.Priority,
			&j.ErrorMessage, &j.Progress, &j.BytesSaved, &j.StartedAt, &j.FinishedAt, &j.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan job: %w", err)
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

func (db *DB) UpdateJobStatus(id int64, status JobStatus, errMsg string) error {
	var err error
	if status == JobRunning {
		_, err = db.conn.Exec(
			`UPDATE jobs SET status=?, started_at=CURRENT_TIMESTAMP, error_message=NULL WHERE id=?`,
			status, id)
	} else if status == JobDone || status == JobFailed || status == JobCancelled {
		var errVal any
		if errMsg != "" {
			errVal = errMsg
		}
		_, err = db.conn.Exec(
			`UPDATE jobs SET status=?, finished_at=CURRENT_TIMESTAMP, error_message=? WHERE id=?`,
			status, errVal, id)
	} else {
		_, err = db.conn.Exec(`UPDATE jobs SET status=?, error_message=? WHERE id=?`,
			status, errMsg, id)
	}
	if err != nil {
		return fmt.Errorf("update job status: %w", err)
	}
	return nil
}

func (db *DB) UpdateJobProgress(id int64, progress float64) error {
	_, err := db.conn.Exec(`UPDATE jobs SET progress=? WHERE id=?`, progress, id)
	if err != nil {
		return fmt.Errorf("update job progress: %w", err)
	}
	return nil
}

func (db *DB) CompleteJob(id int64, outputPath string, outputSize int64, encoderUsed string, bytesSaved int64) error {
	_, err := db.conn.Exec(`
		UPDATE jobs
		SET status='done', output_path=?, output_size=?, encoder_used=?,
		    bytes_saved=?, progress=1.0, finished_at=CURRENT_TIMESTAMP
		WHERE id=?`,
		outputPath, outputSize, encoderUsed, bytesSaved, id)
	if err != nil {
		return fmt.Errorf("complete job: %w", err)
	}
	return nil
}

// SourcePathExists returns true if a non-failed job exists for the given path.
func (db *DB) SourcePathExists(path string) (bool, error) {
	var count int
	err := db.conn.QueryRow(
		`SELECT COUNT(*) FROM jobs WHERE source_path=? AND status != 'failed'`, path).
		Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check source path: %w", err)
	}
	return count > 0, nil
}
