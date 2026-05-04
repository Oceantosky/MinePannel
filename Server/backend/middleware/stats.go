package middleware

import (
	"net/http"
	"sync/atomic"
)

type statsResponseWriter struct {
	http.ResponseWriter
	counter *int64
}

func (w *statsResponseWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	atomic.AddInt64(w.counter, int64(n))
	return n, err
}

func (w *statsResponseWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// WithByteCounter wraps an http.Handler to count response bytes written.
func WithByteCounter(counter *int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(&statsResponseWriter{w, counter}, r)
		})
	}
}
