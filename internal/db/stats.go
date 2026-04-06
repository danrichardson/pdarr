package db

import (
	"fmt"
	"time"
)

type Stats struct {
	TotalBytesSaved int64
	TotalJobsDone   int64
	TotalJobsFailed int64
	UpdatedAt       time.Time
}

func (db *DB) GetStats() (*Stats, error) {
	// Compute live from jobs table so counts stay accurate after clearing history.
	s := &Stats{UpdatedAt: time.Now()}
	err := db.conn.QueryRow(`
		SELECT
			COALESCE(SUM(CASE WHEN status IN ('done','staged') THEN COALESCE(bytes_saved,0) ELSE 0 END), 0),
			COUNT(CASE WHEN status IN ('done','staged') THEN 1 END),
			COUNT(CASE WHEN status='failed' THEN 1 END)
		FROM jobs`).Scan(&s.TotalBytesSaved, &s.TotalJobsDone, &s.TotalJobsFailed)
	if err != nil {
		return nil, fmt.Errorf("get stats: %w", err)
	}
	return s, nil
}

func (db *DB) RecordJobDone(bytesSaved int64) error {
	_, err := db.conn.Exec(`
		UPDATE stats
		SET total_bytes_saved = total_bytes_saved + ?,
		    total_jobs_done   = total_jobs_done + 1,
		    updated_at        = CURRENT_TIMESTAMP
		WHERE id = 1`, bytesSaved)
	if err != nil {
		return fmt.Errorf("record job done: %w", err)
	}
	return nil
}

func (db *DB) RecordJobFailed() error {
	_, err := db.conn.Exec(`
		UPDATE stats
		SET total_jobs_failed = total_jobs_failed + 1,
		    updated_at        = CURRENT_TIMESTAMP
		WHERE id = 1`)
	if err != nil {
		return fmt.Errorf("record job failed: %w", err)
	}
	return nil
}
