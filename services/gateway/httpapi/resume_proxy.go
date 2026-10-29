package httpapi

import (
	"context"
	"io"
	"net/http"
)

// RunResumeService streams a resumed run's SSE response.
type RunResumeService interface {
	ResumeRun(ctx context.Context, runID string) (*http.Response, error)
}

func (s *Server) handleRunResume(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.resumeService == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "resume unavailable")
		return
	}
	runID, ok := runIDAndSuffix(r.URL.Path, "/resume")
	if !ok {
		http.NotFound(w, r)
		return
	}
	resp, err := s.resumeService.ResumeRun(r.Context(), runID)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "resume service unavailable")
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		_, _ = w.Write(body)
		return
	}
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "text/event-stream"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	flusher, _ := w.(http.Flusher)
	buf := make([]byte, 4096)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			_, _ = w.Write(buf[:n])
			if flusher != nil {
				flusher.Flush()
			}
		}
		if readErr != nil {
			if readErr != io.EOF {
				// Client may have gone away; the run itself is durable.
				_ = readErr
			}
			return
		}
	}
}
