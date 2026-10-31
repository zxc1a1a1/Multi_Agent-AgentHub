package httpapi

import (
	"bytes"
	"net/http"
)

// captureResponseWriter tees the handler response into memory so the SQL
// idempotency record can replay it for later requests.
type captureResponseWriter struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
}

func (w *captureResponseWriter) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *captureResponseWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	w.body.Write(data)
	return w.ResponseWriter.Write(data)
}

func (w *captureResponseWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}
