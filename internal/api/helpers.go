package api

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
)

func jsonOK(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func pathID(r *http.Request) (int64, bool) {
	s := r.PathValue("id")
	id, err := strconv.ParseInt(s, 10, 64)
	return id, err == nil
}

// safePathIn checks that path is within one of the allowed directory roots.
func (s *Server) safePathIn(path string) bool {
	if strings.Contains(path, "..") {
		return false
	}
	dirs, err := s.db.ListDirectories()
	if err != nil {
		return false
	}
	clean := filepath.Clean(path)
	for _, d := range dirs {
		if strings.HasPrefix(clean, filepath.Clean(d.Path)) {
			return true
		}
	}
	return false
}
