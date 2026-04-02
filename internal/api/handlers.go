package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/crypto/bcrypt"

	"github.com/danrichardson/pdarr/internal/db"
)

// GET /status
func (s *Server) handleGetStatus(w http.ResponseWriter, r *http.Request) {
	stats, err := s.db.GetStats()
	if err != nil {
		jsonError(w, "database error", http.StatusInternalServerError)
		return
	}

	diskFreeGB := s.diskFreeGB()

	jsonOK(w, map[string]any{
		"version":         "1.0.0",
		"encoder":         s.encoder.DisplayName,
		"paused":          s.worker.IsPaused(),
		"total_saved_gb":  float64(stats.TotalBytesSaved) / (1024 * 1024 * 1024),
		"jobs_done":       stats.TotalJobsDone,
		"jobs_failed":     stats.TotalJobsFailed,
		"disk_free_gb":    diskFreeGB,
		"pause_threshold": s.cfg.Safety.DiskFreePauseGB,
	})
}

// GET /stats
func (s *Server) handleGetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.db.GetStats()
	if err != nil {
		jsonError(w, "database error", http.StatusInternalServerError)
		return
	}
	jsonOK(w, stats)
}

// GET /jobs
func (s *Server) handleListJobs(w http.ResponseWriter, r *http.Request) {
	statusFilter := db.JobStatus(r.URL.Query().Get("status"))
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 50
	offset := 0
	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 {
			limit = v
		}
	}
	if offsetStr != "" {
		if v, err := strconv.Atoi(offsetStr); err == nil && v >= 0 {
			offset = v
		}
	}

	jobs, err := s.db.ListJobs(statusFilter, limit, offset)
	if err != nil {
		jsonError(w, "database error", http.StatusInternalServerError)
		return
	}
	jsonOK(w, jobs)
}

// POST /jobs — manually enqueue a file
func (s *Server) handleCreateJob(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Path == "" {
		jsonError(w, "path is required", http.StatusBadRequest)
		return
	}
	if !s.safePathIn(req.Path) {
		jsonError(w, "path is not within a configured directory", http.StatusBadRequest)
		return
	}

	exists, _ := s.db.SourcePathExists(req.Path)
	if exists {
		jsonError(w, "job already exists for this path", http.StatusConflict)
		return
	}

	id, err := s.db.InsertJob(&db.Job{
		SourcePath: req.Path,
		Status:     db.JobPending,
		Priority:   1, // manual jobs get slightly higher priority
	})
	if err != nil {
		jsonError(w, "database error", http.StatusInternalServerError)
		return
	}
	job, _ := s.db.GetJob(id)
	w.WriteHeader(http.StatusCreated)
	jsonOK(w, job)
}

// GET /jobs/{id}
func (s *Server) handleGetJob(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		jsonError(w, "invalid job id", http.StatusBadRequest)
		return
	}
	job, err := s.db.GetJob(id)
	if err != nil || job == nil {
		jsonError(w, "job not found", http.StatusNotFound)
		return
	}
	jsonOK(w, job)
}

// DELETE /jobs/{id} — cancel a pending job
func (s *Server) handleCancelJob(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		jsonError(w, "invalid job id", http.StatusBadRequest)
		return
	}
	job, err := s.db.GetJob(id)
	if err != nil || job == nil {
		jsonError(w, "job not found", http.StatusNotFound)
		return
	}
	if job.Status != db.JobPending {
		jsonError(w, "only pending jobs can be cancelled", http.StatusBadRequest)
		return
	}
	s.db.UpdateJobStatus(id, db.JobCancelled, "cancelled by user")
	w.WriteHeader(http.StatusNoContent)
}

// POST /jobs/{id}/retry
func (s *Server) handleRetryJob(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		jsonError(w, "invalid job id", http.StatusBadRequest)
		return
	}
	job, err := s.db.GetJob(id)
	if err != nil || job == nil {
		jsonError(w, "job not found", http.StatusNotFound)
		return
	}
	if job.Status != db.JobFailed && job.Status != db.JobCancelled {
		jsonError(w, "only failed or cancelled jobs can be retried", http.StatusBadRequest)
		return
	}
	s.db.UpdateJobStatus(id, db.JobPending, "")
	w.WriteHeader(http.StatusNoContent)
}

// GET /directories
func (s *Server) handleListDirectories(w http.ResponseWriter, r *http.Request) {
	dirs, err := s.db.ListDirectories()
	if err != nil {
		jsonError(w, "database error", http.StatusInternalServerError)
		return
	}
	jsonOK(w, dirs)
}

// POST /directories
func (s *Server) handleCreateDirectory(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path        string `json:"path"`
		Enabled     *bool  `json:"enabled"`
		MinAgeDays  int    `json:"min_age_days"`
		MaxBitrate  int64  `json:"max_bitrate"`
		MinSizeMB   int64  `json:"min_size_mb"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Path == "" {
		jsonError(w, "path is required", http.StatusBadRequest)
		return
	}
	if strings.Contains(req.Path, "..") {
		jsonError(w, "path must not contain ..", http.StatusBadRequest)
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	if req.MinAgeDays == 0 {
		req.MinAgeDays = 7
	}
	if req.MaxBitrate == 0 {
		req.MaxBitrate = 4_000_000
	}
	if req.MinSizeMB == 0 {
		req.MinSizeMB = 500
	}

	dir := &db.Directory{
		Path:       req.Path,
		Enabled:    enabled,
		MinAgeDays: req.MinAgeDays,
		MaxBitrate: req.MaxBitrate,
		MinSizeMB:  req.MinSizeMB,
	}
	id, err := s.db.InsertDirectory(dir)
	if err != nil {
		jsonError(w, "database error", http.StatusInternalServerError)
		return
	}
	created, _ := s.db.GetDirectory(id)
	w.WriteHeader(http.StatusCreated)
	jsonOK(w, created)
}

// GET /directories/{id}
func (s *Server) handleGetDirectory(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		jsonError(w, "invalid id", http.StatusBadRequest)
		return
	}
	d, err := s.db.GetDirectory(id)
	if err != nil || d == nil {
		jsonError(w, "directory not found", http.StatusNotFound)
		return
	}
	jsonOK(w, d)
}

// PUT /directories/{id}
func (s *Server) handleUpdateDirectory(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		jsonError(w, "invalid id", http.StatusBadRequest)
		return
	}
	d, err := s.db.GetDirectory(id)
	if err != nil || d == nil {
		jsonError(w, "directory not found", http.StatusNotFound)
		return
	}

	var req struct {
		Path       *string `json:"path"`
		Enabled    *bool   `json:"enabled"`
		MinAgeDays *int    `json:"min_age_days"`
		MaxBitrate *int64  `json:"max_bitrate"`
		MinSizeMB  *int64  `json:"min_size_mb"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Path != nil {
		if strings.Contains(*req.Path, "..") {
			jsonError(w, "path must not contain ..", http.StatusBadRequest)
			return
		}
		d.Path = *req.Path
	}
	if req.Enabled != nil {
		d.Enabled = *req.Enabled
	}
	if req.MinAgeDays != nil {
		d.MinAgeDays = *req.MinAgeDays
	}
	if req.MaxBitrate != nil {
		d.MaxBitrate = *req.MaxBitrate
	}
	if req.MinSizeMB != nil {
		d.MinSizeMB = *req.MinSizeMB
	}

	if err := s.db.UpdateDirectory(d); err != nil {
		jsonError(w, "database error", http.StatusInternalServerError)
		return
	}
	updated, _ := s.db.GetDirectory(id)
	jsonOK(w, updated)
}

// DELETE /directories/{id}
func (s *Server) handleDeleteDirectory(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		jsonError(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := s.db.DeleteDirectory(id); err != nil {
		jsonError(w, "database error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// POST /scan — trigger an immediate scan
func (s *Server) handleTriggerScan(w http.ResponseWriter, r *http.Request) {
	dirs, err := s.db.ListDirectories()
	if err != nil {
		jsonError(w, "database error", http.StatusInternalServerError)
		return
	}
	go func() {
		for _, d := range dirs {
			if _, err := s.scanner.ScanDirectory(r.Context(), d); err != nil {
				s.log.Error("scan error", "directory", d.Path, "error", err)
			}
		}
	}()
	jsonOK(w, map[string]string{"status": "scan started"})
}

// POST /queue/pause
func (s *Server) handlePauseQueue(w http.ResponseWriter, r *http.Request) {
	s.worker.SetPaused(true)
	jsonOK(w, map[string]bool{"paused": true})
}

// POST /queue/resume
func (s *Server) handleResumeQueue(w http.ResponseWriter, r *http.Request) {
	s.worker.SetPaused(false)
	jsonOK(w, map[string]bool{"paused": false})
}

// POST /auth/login
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Auth.PasswordHash == "" {
		jsonError(w, "authentication not configured", http.StatusNotFound)
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := bcrypt.CompareHashAndPassword(
		[]byte(s.cfg.Auth.PasswordHash),
		[]byte(req.Password),
	); err != nil {
		jsonError(w, "invalid password", http.StatusUnauthorized)
		return
	}

	token, err := s.issueJWT()
	if err != nil {
		jsonError(w, "could not issue token", http.StatusInternalServerError)
		return
	}

	jsonOK(w, map[string]string{"token": token})
}

func (s *Server) diskFreeGB() float64 {
	if runtime.GOOS == "windows" {
		return -1
	}
	var stat syscall.Statfs_t
	dir := s.cfg.Server.DataDir
	if err := syscall.Statfs(dir, &stat); err != nil {
		return -1
	}
	return float64(stat.Bavail) * float64(stat.Bsize) / (1024 * 1024 * 1024)
}

var _ = sql.NullInt64{} // ensure database/sql import is used
