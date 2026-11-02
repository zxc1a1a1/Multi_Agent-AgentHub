package httpapi

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/artifacts"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/render"
)

const renderMaxRequestBody = 16 * 1024

type renderRequest struct {
	RunID    string `json:"runId"`
	TaskID   string `json:"taskId"`
	URL      string `json:"url"`
	HTML     string `json:"html"`
	Format   string `json:"format"`
	FullPage bool   `json:"fullPage"`
}

// handleRender renders a screenshot/PDF, stores the bytes in the configured
// object store and publishes artifact metadata with checksum/storageRef.
func (s *Server) handleRender(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if !s.checkServiceAuth(r) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if s.pageRenderer == nil || s.artifactBlobs == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "renderer unavailable"})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, renderMaxRequestBody)
	var req renderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	req.RunID = strings.TrimSpace(req.RunID)
	req.URL = strings.TrimSpace(req.URL)
	html := strings.TrimSpace(req.HTML)
	format := strings.ToLower(strings.TrimSpace(req.Format))
	if req.RunID == "" || (req.URL == "" && html == "") || (req.URL != "" && html != "") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "runId and exactly one of url or html are required"})
		return
	}
	if format != render.FormatPNG && format != render.FormatPDF {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "format must be png or pdf"})
		return
	}
	if html != "" {
		if len(html) > render.MaxHTMLBytes {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "html exceeds size limit"})
			return
		}
	} else if err := render.ValidateURL(req.URL); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "url is not allowed"})
		return
	}
	data, contentType, err := s.pageRenderer.Render(r.Context(), render.Request{
		URL: req.URL, HTML: html, Format: format, FullPage: req.FullPage,
	})
	if err != nil {
		log.Printf("orchestrator: render %s failed: %v", format, err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "render failed"})
		return
	}
	artifactID, err := newArtifactID()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "artifact id unavailable"})
		return
	}
	ext := "." + format
	key := "runs/" + sanitizeRenderSegment(req.RunID) + "/" + artifactID + ext
	if err := s.artifactBlobs.Put(r.Context(), key, data, contentType); err != nil {
		log.Printf("orchestrator: render blob put failed: %v", err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "artifact storage failed"})
		return
	}
	sum := sha256.Sum256(data)
	checksum := "sha256:" + hex.EncodeToString(sum[:])
	kind := "image"
	name := "screenshot-" + artifactID + ext
	if format == render.FormatPDF {
		kind = "document"
		name = "page-" + artifactID + ext
	}
	record := artifacts.ArtifactRecord{
		ID: artifactID, RunID: req.RunID, TaskID: strings.TrimSpace(req.TaskID),
		Name: name, Kind: kind, MimeType: contentType, ContentType: contentType,
		Size: int64(len(data)), StorageRef: "obj://" + key, Checksum: checksum, Version: 1,
		DownloadPath: "/internal/orchestrator/artifacts/" + artifactID + "/content",
		CreatedAt:    time.Now().UTC(),
	}
	if err := s.ensureArtifactStore().Create(r.Context(), record); err != nil {
		_ = s.artifactBlobs.Delete(r.Context(), key)
		log.Printf("orchestrator: render artifact metadata failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "artifact metadata failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"artifactId": artifactID, "kind": kind, "mimeType": contentType,
		"size": len(data), "checksum": checksum, "downloadPath": record.DownloadPath,
	})
}

func (s *Server) serveArtifactContent(w http.ResponseWriter, r *http.Request, artifactID string) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodHead)
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.artifactBlobs == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "artifact blobs unavailable")
		return
	}
	record, ok, err := s.ensureArtifactStore().Get(r.Context(), artifactID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to get artifact")
		return
	}
	if !ok {
		writeJSONError(w, http.StatusNotFound, "artifact not found")
		return
	}
	key, ok := strings.CutPrefix(record.StorageRef, "obj://")
	if !ok || key == "" {
		writeJSONError(w, http.StatusConflict, "artifact has no stored content")
		return
	}
	data, err := s.artifactBlobs.Get(r.Context(), key)
	if errors.Is(err, artifacts.ErrObjectNotFound) {
		writeJSONError(w, http.StatusNotFound, "artifact content not found")
		return
	}
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "artifact content unavailable")
		return
	}
	if record.Checksum != "" {
		expected, ok := strings.CutPrefix(record.Checksum, "sha256:")
		sum := sha256.Sum256(data)
		actual := hex.EncodeToString(sum[:])
		if !ok || subtle.ConstantTimeCompare([]byte(expected), []byte(actual)) != 1 {
			writeJSONError(w, http.StatusConflict, "artifact checksum mismatch")
			return
		}
	}
	if record.ContentType != "" {
		w.Header().Set("Content-Type", record.ContentType)
	} else if record.MimeType != "" {
		w.Header().Set("Content-Type", record.MimeType)
	}
	w.Header().Set("Content-Disposition", `attachment; filename="`+sanitizeFilename(record.Name)+`"`)
	http.ServeContent(w, r, record.Name, record.CreatedAt, bytes.NewReader(data))
}

func newArtifactID() (string, error) {
	buffer := make([]byte, 12)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return "art_" + hex.EncodeToString(buffer), nil
}

func sanitizeRenderSegment(value string) string {
	var builder strings.Builder
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '-' || char == '_' {
			builder.WriteRune(char)
		}
	}
	if builder.Len() == 0 || builder.Len() > 64 {
		return "adhoc"
	}
	return builder.String()
}

func sanitizeFilename(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "artifact"
	}
	return strings.Map(func(char rune) rune {
		switch char {
		case '"', '\\', '\r', '\n':
			return '_'
		default:
			return char
		}
	}, name)
}
