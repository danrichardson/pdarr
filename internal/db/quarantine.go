package db

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type QuarantineRecord struct {
	ID             int64
	JobID          int64
	OriginalPath   string
	QuarantinePath string
	ExpiresAt      time.Time
	DeletedAt      sql.NullTime
	CreatedAt      time.Time
}

func (db *DB) InsertQuarantine(q *QuarantineRecord) (int64, error) {
	res, err := db.conn.Exec(`
		INSERT INTO quarantine (job_id, original_path, quarantine_path, expires_at)
		VALUES (?, ?, ?, ?)`,
		q.JobID, q.OriginalPath, q.QuarantinePath, q.ExpiresAt)
	if err != nil {
		return 0, fmt.Errorf("insert quarantine: %w", err)
	}
	return res.LastInsertId()
}

func (db *DB) GetQuarantineByJobID(jobID int64) (*QuarantineRecord, error) {
	q := &QuarantineRecord{}
	err := db.conn.QueryRow(`
		SELECT id, job_id, original_path, quarantine_path, expires_at, deleted_at, created_at
		FROM quarantine WHERE job_id=? AND deleted_at IS NULL`, jobID).
		Scan(&q.ID, &q.JobID, &q.OriginalPath, &q.QuarantinePath, &q.ExpiresAt, &q.DeletedAt, &q.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get quarantine by job: %w", err)
	}
	return q, nil
}

// ExpiredQuarantines returns records past their expiry that haven't been deleted.
func (db *DB) ExpiredQuarantines() ([]*QuarantineRecord, error) {
	rows, err := db.conn.Query(`
		SELECT id, job_id, original_path, quarantine_path, expires_at, deleted_at, created_at
		FROM quarantine
		WHERE deleted_at IS NULL AND expires_at <= CURRENT_TIMESTAMP`)
	if err != nil {
		return nil, fmt.Errorf("expired quarantines: %w", err)
	}
	defer rows.Close()

	var records []*QuarantineRecord
	for rows.Next() {
		q := &QuarantineRecord{}
		if err := rows.Scan(&q.ID, &q.JobID, &q.OriginalPath, &q.QuarantinePath,
			&q.ExpiresAt, &q.DeletedAt, &q.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan quarantine: %w", err)
		}
		records = append(records, q)
	}
	return records, rows.Err()
}

func (db *DB) MarkQuarantineDeleted(id int64) error {
	_, err := db.conn.Exec(
		`UPDATE quarantine SET deleted_at=CURRENT_TIMESTAMP WHERE id=?`, id)
	if err != nil {
		return fmt.Errorf("mark quarantine deleted: %w", err)
	}
	return nil
}
