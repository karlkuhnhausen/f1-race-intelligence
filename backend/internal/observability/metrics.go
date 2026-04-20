package observability

import (
	"net/http"
	"sync/atomic"
	"time"
)

// Metrics tracks basic application health counters.
type Metrics struct {
	requestCount    atomic.Int64
	requestErrors   atomic.Int64
	pollSuccessCount atomic.Int64
	pollErrorCount   atomic.Int64
}

// NewMetrics creates a new Metrics instance.
func NewMetrics() *Metrics {
	return &Metrics{}
}

// IncRequest increments the request counter.
func (m *Metrics) IncRequest() { m.requestCount.Add(1) }

// IncRequestError increments the request error counter.
func (m *Metrics) IncRequestError() { m.requestErrors.Add(1) }

// IncPollSuccess increments the poll success counter.
func (m *Metrics) IncPollSuccess() { m.pollSuccessCount.Add(1) }

// IncPollError increments the poll error counter.
func (m *Metrics) IncPollError() { m.pollErrorCount.Add(1) }

// Snapshot returns current metric values.
func (m *Metrics) Snapshot() map[string]int64 {
	return map[string]int64{
		"request_count":      m.requestCount.Load(),
		"request_errors":     m.requestErrors.Load(),
		"poll_success_count": m.pollSuccessCount.Load(),
		"poll_error_count":   m.pollErrorCount.Load(),
	}
}

// Middleware returns an HTTP middleware that tracks request counts and latency.
func (m *Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.IncRequest()
		start := time.Now()

		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rw, r)

		if rw.statusCode >= 500 {
			m.IncRequestError()
		}

		_ = time.Since(start) // Available for future histogram integration.
	})
}

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
