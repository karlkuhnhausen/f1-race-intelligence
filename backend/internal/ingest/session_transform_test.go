package ingest

import (
	"io"
	"log/slog"
	"testing"
	"time"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestTransformSession_StatusDerivedFromDates(t *testing.T) {
	// Choose dates relative to the current wall clock so the assertion
	// holds regardless of when the test runs. TransformSession uses
	// time.Now().UTC() internally.
	now := time.Now().UTC()

	cases := []struct {
		name      string
		dateStart string
		dateEnd   string
		want      string
	}{
		{
			name:      "future session is upcoming",
			dateStart: now.Add(72 * time.Hour).Format(time.RFC3339),
			dateEnd:   now.Add(74 * time.Hour).Format(time.RFC3339),
			want:      "upcoming",
		},
		{
			name:      "past session is completed",
			dateStart: now.Add(-72 * time.Hour).Format(time.RFC3339),
			dateEnd:   now.Add(-70 * time.Hour).Format(time.RFC3339),
			want:      "completed",
		},
		{
			name:      "missing dates default to upcoming",
			dateStart: "",
			dateEnd:   "",
			want:      "upcoming",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := TestTransformSession(1, "Practice 1", 100, tc.dateStart, tc.dateEnd, 2026, 2026, 1)
			if got.Status != tc.want {
				t.Errorf("Status = %q, want %q", got.Status, tc.want)
			}
		})
	}
}

func TestIsFutureSession(t *testing.T) {
	now := time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name      string
		dateStart string
		dateEnd   string
		want      bool
	}{
		{
			name:      "future end → future",
			dateStart: now.Add(48 * time.Hour).Format(time.RFC3339),
			dateEnd:   now.Add(50 * time.Hour).Format(time.RFC3339),
			want:      true,
		},
		{
			name:      "past end → not future",
			dateStart: now.Add(-50 * time.Hour).Format(time.RFC3339),
			dateEnd:   now.Add(-48 * time.Hour).Format(time.RFC3339),
			want:      false,
		},
		{
			name:      "empty date_end falls back to date_start (future)",
			dateStart: now.Add(48 * time.Hour).Format(time.RFC3339),
			dateEnd:   "",
			want:      true,
		},
		{
			name:      "empty date_end falls back to date_start (past)",
			dateStart: now.Add(-1 * time.Hour).Format(time.RFC3339),
			dateEnd:   "",
			want:      false,
		},
		{
			name:      "unparseable date_end falls back to date_start",
			dateStart: now.Add(48 * time.Hour).Format(time.RFC3339),
			dateEnd:   "not-a-date",
			want:      true,
		},
		{
			name:      "neither date usable → defensive future",
			dateStart: "",
			dateEnd:   "",
			want:      true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw := openF1Session{
				SessionKey: 1,
				DateStart:  tc.dateStart,
				DateEnd:    tc.dateEnd,
			}
			got := isFutureSession(raw, now, testLogger())
			if got != tc.want {
				t.Errorf("isFutureSession = %v, want %v", got, tc.want)
			}
		})
	}
}
