package db

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// OriginalRecord tracks a source file that was moved to the processed directory
// while its transcoded replacement is under review.
type OriginalRecord struct {
	ID           int64
	JobID        int64
	OriginalPath string   // where the file was originally
	HeldPath     string   // where the original now lives (processed dir)
	OutputPath   string   // where the transcoded file was placed
	OriginalSize int64
	OutputSize   int64
	ExpiresAt    time.Time
	DeletedAt    sql.NullTime
	CreatedAt    time.Time
}

func (db *DB) InsertOriginal(r *OriginalRecord) (int64, error) {
	res, err := db.conn.Exec(`
		INSERT INTO originals
		  (job_id, original_path, held_path, output_path, original_size, output_size, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		r.JobID, r.OriginalPath, r.HeldPath, r.OutputPath,
		r.OriginalSize, r.OutputSize, r.ExpiresAt)
	if err != nil {
		return 0, fmt.Errorf("insert original: %w", err)
	}
	return res.LastInsertId()
}

func (db *DB) GetOriginalByJobID(jobID int64) (*OriginalRecord, error) {
	r := &OriginalRecord{}
	err := db.conn.QueryRow(`
		SELECT id, job_id, original_path, held_path, output_path,
		       original_size, output_size, expires_at, deleted_at, created_at
		FROM originals WHERE job_id=? AND deleted_at IS NULL`, jobID).
		Scan(&r.ID, &r.JobID, &r.OriginalPath, &r.HeldPath, &r.OutputPath,
			&r.OriginalSize, &r.OutputSize, &r.ExpiresAt, &r.DeletedAt, &r.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get original by job: %w", err)
	}
	return r, nil
}

func (db *DB) GetOriginal(id int64) (*OriginalRecord, error) {
	r := &OriginalRecord{}
	err := db.conn.QueryRow(`
		SELECT id, job_id, original_path, held_path, output_path,
		       original_size, output_size, expires_at, deleted_at, created_at
		FROM originals WHERE id=? AND deleted_at IS NULL`, id).
		Scan(&r.ID, &r.JobID, &r.OriginalPath, &r.HeldPath, &r.OutputPath,
			&r.OriginalSize, &r.OutputSize, &r.ExpiresAt, &r.DeletedAt, &r.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get original: %w", err)
	}
	return r, nil
}

// ActiveOriginals returns records that have not been deleted, ordered by
// soonest expiry first.
func (db *DB) ActiveOriginals() ([]*OriginalRecord, error) {
	rows, err := db.conn.Query(`
		SELECT id, job_id, original_path, held_path, output_path,
		       original_size, output_size, expires_at, deleted_at, created_at
		FROM originals
		WHERE deleted_at IS NULL ORDER BY expires_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("active originals: %w", err)
	}
	defer rows.Close()
	return scanOriginals(rows)
}

// ExpiredOriginals returns records past their expiry that haven't been deleted.
func (db *DB) ExpiredOriginals() ([]*OriginalRecord, error) {
	rows, err := db.conn.Query(`
		SELECT id, job_id, original_path, held_path, output_path,
		       original_size, output_size, expires_at, deleted_at, created_at
		FROM originals
		WHERE deleted_at IS NULL AND expires_at <= CURRENT_TIMESTAMP`)
	if err != nil {
		return nil, fmt.Errorf("expired originals: %w", err)
	}
	defer rows.Close()
	return scanOriginals(rows)
}

// MarkOriginalDeleted soft-deletes an original record (sets deleted_at).
func (db *DB) MarkOriginalDeleted(id int64) error {
	_, err := db.conn.Exec(
		`UPDATE originals SET deleted_at=CURRENT_TIMESTAMP WHERE id=?`, id)
	if err != nil {
		return fmt.Errorf("mark original deleted: %w", err)
	}
	return nil
}

func scanOriginals(rows *sql.Rows) ([]*OriginalRecord, error) {
	var records []*OriginalRecord
	for rows.Next() {
		r := &OriginalRecord{}
		if err := rows.Scan(&r.ID, &r.JobID, &r.OriginalPath, &r.HeldPath, &r.OutputPath,
			&r.OriginalSize, &r.OutputSize, &r.ExpiresAt, &r.DeletedAt, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan original: %w", err)
		}
		records = append(records, r)
	}
	return records, rows.Err()
}
