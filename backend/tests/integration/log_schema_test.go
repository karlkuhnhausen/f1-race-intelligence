package integration

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
)

// TestStructuredLogSchemaFields validates that the JSON logger emits the
// required fields consumed by Azure Log Analytics.
func TestStructuredLogSchemaFields(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(handler)

	logger.Info("request",
		slog.String("method", "GET"),
		slog.String("path", "/api/v1/calendar"),
		slog.Int("status", 200),
		slog.Float64("duration_ms", 12.34),
	)

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("log output is not valid JSON: %v\nbody: %s", err, buf.String())
	}

	requiredKeys := []string{"time", "level", "msg"}
	for _, key := range requiredKeys {
		if _, ok := entry[key]; !ok {
			t.Errorf("missing required log field %q in output: %s", key, buf.String())
		}
	}

	if entry["level"] != "INFO" {
		t.Errorf("expected level INFO, got %v", entry["level"])
	}
	if entry["msg"] != "request" {
		t.Errorf("expected msg 'request', got %v", entry["msg"])
	}
}

// TestStructuredLogSchemaExtraAttributes validates that caller-supplied
// attributes appear in the JSON output.
func TestStructuredLogSchemaExtraAttributes(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(handler)

	logger.Info("poll_complete",
		slog.String("source", "openf1"),
		slog.Int("records", 24),
	)

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("log output is not valid JSON: %v", err)
	}

	if entry["source"] != "openf1" {
		t.Errorf("expected source 'openf1', got %v", entry["source"])
	}
	if entry["records"] != float64(24) {
		t.Errorf("expected records 24, got %v", entry["records"])
	}
}

// TestStructuredLogSchemaErrorLevel validates error-level output.
func TestStructuredLogSchemaErrorLevel(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(handler)

	logger.Error("poll_error",
		slog.String("source", "openf1"),
		slog.String("error", "connection refused"),
	)

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("log output is not valid JSON: %v", err)
	}

	if entry["level"] != "ERROR" {
		t.Errorf("expected level ERROR, got %v", entry["level"])
	}
	if entry["msg"] != "poll_error" {
		t.Errorf("expected msg 'poll_error', got %v", entry["msg"])
	}
	if entry["error"] != "connection refused" {
		t.Errorf("expected error 'connection refused', got %v", entry["error"])
	}
}

// TestStructuredLogSchemaSessionIngestion validates session ingestion log output.
func TestStructuredLogSchemaSessionIngestion(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(handler)

	logger.Info("session_poll_complete",
		slog.String("source", "openf1"),
		slog.Int("season", 2026),
		slog.Int("sessions", 7),
	)

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("log output is not valid JSON: %v", err)
	}

	if entry["msg"] != "session_poll_complete" {
		t.Errorf("expected msg 'session_poll_complete', got %v", entry["msg"])
	}
	if entry["source"] != "openf1" {
		t.Errorf("expected source 'openf1', got %v", entry["source"])
	}
	if entry["season"] != float64(2026) {
		t.Errorf("expected season 2026, got %v", entry["season"])
	}
	if entry["sessions"] != float64(7) {
		t.Errorf("expected sessions 7, got %v", entry["sessions"])
	}
}

// TestStructuredLogSchemaRoundDetailAPI validates round detail request log output.
func TestStructuredLogSchemaRoundDetailAPI(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(handler)

	logger.Info("request",
		slog.String("method", "GET"),
		slog.String("path", "/api/v1/rounds/1"),
		slog.Int("status", 200),
		slog.Float64("duration_ms", 45.67),
		slog.Int("year", 2026),
		slog.Int("round", 1),
	)

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("log output is not valid JSON: %v", err)
	}

	if entry["path"] != "/api/v1/rounds/1" {
		t.Errorf("expected path '/api/v1/rounds/1', got %v", entry["path"])
	}
	if entry["year"] != float64(2026) {
		t.Errorf("expected year 2026, got %v", entry["year"])
	}
	if entry["round"] != float64(1) {
		t.Errorf("expected round 1, got %v", entry["round"])
	}
}

// TestStructuredLogSchemaSessionUpsertError validates session upsert error logging.
func TestStructuredLogSchemaSessionUpsertError(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(handler)

	logger.Error("session_upsert_failed",
		slog.String("session_id", "2026-01-race"),
		slog.String("error", "cosmos: conflict"),
	)

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("log output is not valid JSON: %v", err)
	}

	if entry["level"] != "ERROR" {
		t.Errorf("expected level ERROR, got %v", entry["level"])
	}
	if entry["session_id"] != "2026-01-race" {
		t.Errorf("expected session_id '2026-01-race', got %v", entry["session_id"])
	}
}
