package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage"
)

// openF1RaceControlMsg is the raw upstream shape from OpenF1 /v1/race_control.
type openF1RaceControlMsg struct {
	Category   string `json:"category"`
	Flag       string `json:"flag"`
	Message    string `json:"message"`
	LapNumber  int    `json:"lap_number"`
	SessionKey int    `json:"session_key"`
}

// FetchRaceControlMsgs fetches race control messages for a session from OpenF1.
// The caller is responsible for rate-limiting before calling this function.
func FetchRaceControlMsgs(ctx context.Context, client *http.Client, sessionKey int) ([]openF1RaceControlMsg, error) {
	url := fmt.Sprintf("%s/race_control?session_key=%d", openF1BaseURL, sessionKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("race_control: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("race_control: request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("race_control: unexpected status %d", resp.StatusCode)
	}

	var msgs []openF1RaceControlMsg
	if err := json.NewDecoder(resp.Body).Decode(&msgs); err != nil {
		return nil, fmt.Errorf("race_control: decode: %w", err)
	}
	return msgs, nil
}

// eventAccumulator tracks distinct lap-number activations of a race control event type.
type eventAccumulator struct {
	lapsSeen map[int]bool
	firstLap int // -1 means no activations recorded yet
}

func newEventAccumulator() *eventAccumulator {
	return &eventAccumulator{lapsSeen: make(map[int]bool), firstLap: -1}
}

func (a *eventAccumulator) add(lap int) {
	if !a.lapsSeen[lap] {
		a.lapsSeen[lap] = true
		if a.firstLap == -1 {
			a.firstLap = lap
		}
	}
}

func (a *eventAccumulator) count() int { return len(a.lapsSeen) }

func (a *eventAccumulator) firstLapNumber() int {
	if a.firstLap == -1 {
		return 0
	}
	return a.firstLap
}

// SummarizeRaceControl aggregates raw race control messages into a summary.
// Messages are deduplicated by activation type + lap_number (multiple messages
// for the same event on the same lap are counted as one activation).
// NotableEvents are returned in priority order: red_flag > safety_car > vsc > investigation.
func SummarizeRaceControl(msgs []openF1RaceControlMsg) storage.RaceControlSummary {
	rfAcc := newEventAccumulator()
	scAcc := newEventAccumulator()
	vscAcc := newEventAccumulator()
	invAcc := newEventAccumulator()

	for _, msg := range msgs {
		switch {
		case msg.Flag == "RED":
			rfAcc.add(msg.LapNumber)
		case strings.HasPrefix(msg.Message, "SAFETY CAR DEPLOYED"):
			scAcc.add(msg.LapNumber)
		case strings.HasPrefix(msg.Message, "VIRTUAL SAFETY CAR DEPLOYED"):
			vscAcc.add(msg.LapNumber)
		case msg.Category == "Other" && strings.Contains(msg.Message, "UNDER INVESTIGATION"):
			invAcc.add(msg.LapNumber)
		}
	}

	// Build notable events in priority order, only including types with at least one activation.
	type eventSpec struct {
		eventType string
		acc       *eventAccumulator
	}
	specs := []eventSpec{
		{"red_flag", rfAcc},
		{"safety_car", scAcc},
		{"vsc", vscAcc},
		{"investigation", invAcc},
	}

	var notableEvents []storage.NotableEvent
	for _, spec := range specs {
		if spec.acc.count() > 0 {
			notableEvents = append(notableEvents, storage.NotableEvent{
				EventType: spec.eventType,
				LapNumber: spec.acc.firstLapNumber(),
				Count:     spec.acc.count(),
			})
		}
	}

	return storage.RaceControlSummary{
		RedFlagCount:   rfAcc.count(),
		SafetyCarCount: scAcc.count(),
		VSCCount:       vscAcc.count(),
		NotableEvents:  notableEvents,
		FetchedAtUTC:   time.Now().UTC(),
	}
}

// RaceControlHydrator fetches race control data for a session from OpenF1,
// persists it to Cosmos DB, and returns the summary. Implements the
// rounds.RaceControlHydrator interface.
type RaceControlHydrator struct {
	repo   storage.SessionRepository
	client *http.Client
	logger *slog.Logger
}

// NewRaceControlHydrator creates a new RaceControlHydrator.
func NewRaceControlHydrator(repo storage.SessionRepository, logger *slog.Logger) *RaceControlHydrator {
	return &RaceControlHydrator{
		repo:   repo,
		client: &http.Client{Timeout: 30 * time.Second},
		logger: logger,
	}
}

// Hydrate fetches race control data for the session, persists it to Cosmos,
// and returns the summary. On error the caller should log and degrade gracefully.
func (h *RaceControlHydrator) Hydrate(ctx context.Context, sess storage.Session) (*storage.RaceControlSummary, error) {
	msgs, err := FetchRaceControlMsgs(ctx, h.client, sess.SessionKey)
	if err != nil {
		return nil, fmt.Errorf("hydrate %s: fetch race control: %w", sess.ID, err)
	}

	summary := SummarizeRaceControl(msgs)
	sess.RaceControlSummary = &summary

	if err := h.repo.UpsertSession(ctx, sess); err != nil {
		return nil, fmt.Errorf("hydrate %s: upsert session: %w", sess.ID, err)
	}

	h.logger.Info("race control hydrated", "session_id", sess.ID)
	return &summary, nil
}
