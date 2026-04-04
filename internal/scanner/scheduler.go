package scanner

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/danrichardson/sqzarr/internal/db"
)

// Scheduler runs periodic directory scans and tracks timing state.
type Scheduler struct {
	scanner     *Scanner
	db          *db.DB
	log         *slog.Logger

	mu          sync.RWMutex
	intervalHrs int
	nextScanAt  time.Time
	lastScanAt  *time.Time

	resetCh chan struct{}
}

// NewScheduler creates a Scheduler. intervalHrs is the initial scan interval.
func NewScheduler(s *Scanner, database *db.DB, intervalHrs int, log *slog.Logger) *Scheduler {
	if intervalHrs < 1 {
		intervalHrs = 1
	}
	return &Scheduler{
		scanner:     s,
		db:          database,
		log:         log,
		intervalHrs: intervalHrs,
		resetCh:     make(chan struct{}, 1),
	}
}

// SetInterval updates the scan interval live. Takes effect after the current
// wait period; calling this cancels the existing wait and starts a new one.
func (sc *Scheduler) SetInterval(hours int) {
	if hours < 1 {
		hours = 1
	}
	sc.mu.Lock()
	sc.intervalHrs = hours
	sc.mu.Unlock()
	// Non-blocking send — if there's already a reset pending, that's fine.
	select {
	case sc.resetCh <- struct{}{}:
	default:
	}
}

// NextScanAt returns the scheduled time of the next automatic scan.
func (sc *Scheduler) NextScanAt() time.Time {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.nextScanAt
}

// LastScanAt returns the time of the most recent completed scan, or nil.
func (sc *Scheduler) LastScanAt() *time.Time {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.lastScanAt
}

// RecordManualScan updates lastScanAt after a manually triggered scan.
func (sc *Scheduler) RecordManualScan() {
	now := time.Now()
	sc.mu.Lock()
	sc.lastScanAt = &now
	sc.mu.Unlock()
}

// Run starts the scheduler loop. Blocks until ctx is cancelled.
func (sc *Scheduler) Run(ctx context.Context) {
	sc.log.Info("scan scheduler started", "interval_hours", sc.intervalHrs)
	sc.scheduleNext()

	for {
		sc.mu.RLock()
		next := sc.nextScanAt
		sc.mu.RUnlock()

		wait := time.Until(next)
		if wait < 0 {
			wait = 0
		}

		select {
		case <-ctx.Done():
			return
		case <-sc.resetCh:
			sc.scheduleNext()
		case <-time.After(wait):
			sc.runScan(ctx)
			sc.scheduleNext()
		}
	}
}

func (sc *Scheduler) scheduleNext() {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.nextScanAt = time.Now().Add(time.Duration(sc.intervalHrs) * time.Hour)
}

func (sc *Scheduler) runScan(ctx context.Context) {
	sc.log.Info("scheduled scan starting")
	dirs, err := sc.db.ListDirectories()
	if err != nil {
		sc.log.Error("scan: list directories", "error", err)
		return
	}
	for _, d := range dirs {
		if _, err := sc.scanner.ScanDirectory(ctx, d); err != nil {
			sc.log.Error("scan: directory error", "path", d.Path, "error", err)
		}
	}
	now := time.Now()
	sc.mu.Lock()
	sc.lastScanAt = &now
	sc.mu.Unlock()
	sc.log.Info("scheduled scan complete")
}
