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
	s := &Stats{}
	err := db.conn.QueryRow(
		`SELECT total_bytes_saved, total_jobs_done, total_jobs_failed, updated_at FROM stats WHERE id=1`).
		Scan(&s.TotalBytesSaved, &s.TotalJobsDone, &s.TotalJobsFailed, &s.UpdatedAt)
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
