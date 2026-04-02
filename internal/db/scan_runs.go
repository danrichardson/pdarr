package db

import (
	"database/sql"
	"fmt"
	"time"
)

type ScanRun struct {
	ID           int64
	DirectoryID  sql.NullInt64
	FilesScanned int
	FilesQueued  int
	FilesSkipped int
	DurationMS   sql.NullInt64
	Error        sql.NullString
	StartedAt    time.Time
	FinishedAt   sql.NullTime
}

func (db *DB) InsertScanRun(dirID sql.NullInt64) (int64, error) {
	res, err := db.conn.Exec(
		`INSERT INTO scan_runs (directory_id) VALUES (?)`, dirID)
	if err != nil {
		return 0, fmt.Errorf("insert scan run: %w", err)
	}
	return res.LastInsertId()
}

func (db *DB) FinishScanRun(id int64, scanned, queued, skipped int, durationMS int64, errMsg string) error {
	var errVal any
	if errMsg != "" {
		errVal = errMsg
	}
	_, err := db.conn.Exec(`
		UPDATE scan_runs
		SET files_scanned=?, files_queued=?, files_skipped=?,
		    duration_ms=?, error=?, finished_at=CURRENT_TIMESTAMP
		WHERE id=?`,
		scanned, queued, skipped, durationMS, errVal, id)
	if err != nil {
		return fmt.Errorf("finish scan run: %w", err)
	}
	return nil
}
