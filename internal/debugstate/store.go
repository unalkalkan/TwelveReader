package debugstate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/unalkalkan/TwelveReader/internal/storage"
	"github.com/unalkalkan/TwelveReader/pkg/types"
)

var unsafeIDChars = regexp.MustCompile(`[^a-zA-Z0-9_.-]+`)

// Store persists debug/observability state using the existing storage adapter.
type Store struct {
	storage storage.Adapter
}

func NewStore(adapter storage.Adapter) *Store {
	return &Store{storage: adapter}
}

func NewID(prefix string) string {
	return fmt.Sprintf("%s_%s", prefix, time.Now().UTC().Format("20060102T150405.000000000Z"))
}

func SafeID(value string) string {
	value = unsafeIDChars.ReplaceAllString(value, "_")
	return strings.Trim(value, "_")
}

func (s *Store) SaveSynthJob(ctx context.Context, job *types.SynthJob) error {
	if job == nil {
		return fmt.Errorf("synth job is nil")
	}
	if job.ID == "" {
		job.ID = fmt.Sprintf("synth_%s_%s", SafeID(job.BookID), SafeID(job.SegmentID))
	}
	if job.UpdatedAt.IsZero() {
		job.UpdatedAt = time.Now().UTC()
	}
	return s.putJSON(ctx, filepath.Join("books", job.BookID, "debug", "synth-jobs", job.ID+".json"), job)
}

func (s *Store) ListSynthJobs(ctx context.Context, bookID string) ([]*types.SynthJob, error) {
	paths, err := s.storage.List(ctx, filepath.Join("books", bookID, "debug", "synth-jobs")+string(filepath.Separator))
	if err != nil {
		return nil, err
	}
	jobs := make([]*types.SynthJob, 0, len(paths))
	for _, path := range paths {
		var job types.SynthJob
		if err := s.getJSON(ctx, path, &job); err == nil {
			jobs = append(jobs, &job)
		}
	}
	sort.Slice(jobs, func(i, j int) bool {
		if jobs[i].SegmentID == jobs[j].SegmentID {
			return jobs[i].UpdatedAt.Before(jobs[j].UpdatedAt)
		}
		return jobs[i].SegmentID < jobs[j].SegmentID
	})
	return jobs, nil
}

func (s *Store) SavePlaybackEvent(ctx context.Context, event *types.PlaybackEvent) error {
	if event == nil {
		return fmt.Errorf("playback event is nil")
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	if event.ID == "" {
		event.ID = NewID("playback")
	}
	if event.UserID == "" {
		event.UserID = "single-user"
	}
	return s.putJSON(ctx, filepath.Join("books", event.BookID, "debug", "playback-events", event.ID+".json"), event)
}

func (s *Store) ListPlaybackEvents(ctx context.Context, bookID string, limit int) ([]*types.PlaybackEvent, error) {
	paths, err := s.storage.List(ctx, filepath.Join("books", bookID, "debug", "playback-events")+string(filepath.Separator))
	if err != nil {
		return nil, err
	}
	events := make([]*types.PlaybackEvent, 0, len(paths))
	for _, path := range paths {
		var event types.PlaybackEvent
		if err := s.getJSON(ctx, path, &event); err == nil {
			events = append(events, &event)
		}
	}
	sort.Slice(events, func(i, j int) bool { return events[i].CreatedAt.After(events[j].CreatedAt) })
	if limit > 0 && len(events) > limit {
		events = events[:limit]
	}
	return events, nil
}

func (s *Store) SaveEvent(ctx context.Context, event *types.DebugEvent) error {
	if event == nil {
		return fmt.Errorf("debug event is nil")
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	if event.ID == "" {
		event.ID = NewID("event")
	}
	if event.Scope == "" {
		event.Scope = "system"
	}
	if event.Severity == "" {
		event.Severity = "info"
	}
	globalPath := filepath.Join("debug", "events", event.ID+".json")
	if err := s.putJSON(ctx, globalPath, event); err != nil {
		return err
	}
	if event.BookID != "" {
		return s.putJSON(ctx, filepath.Join("books", event.BookID, "debug", "events", event.ID+".json"), event)
	}
	return nil
}

func (s *Store) ListEvents(ctx context.Context, bookID string, limit int) ([]*types.DebugEvent, error) {
	prefix := filepath.Join("debug", "events") + string(filepath.Separator)
	if bookID != "" {
		prefix = filepath.Join("books", bookID, "debug", "events") + string(filepath.Separator)
	}
	paths, err := s.storage.List(ctx, prefix)
	if err != nil {
		return nil, err
	}
	events := make([]*types.DebugEvent, 0, len(paths))
	for _, path := range paths {
		var event types.DebugEvent
		if err := s.getJSON(ctx, path, &event); err == nil {
			events = append(events, &event)
		}
	}
	sort.Slice(events, func(i, j int) bool { return events[i].CreatedAt.After(events[j].CreatedAt) })
	if limit > 0 && len(events) > limit {
		events = events[:limit]
	}
	return events, nil
}

func (s *Store) putJSON(ctx context.Context, path string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return s.storage.Put(ctx, path, strings.NewReader(string(data)))
}

func (s *Store) getJSON(ctx context.Context, path string, value interface{}) error {
	reader, err := s.storage.Get(ctx, path)
	if err != nil {
		return err
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, value)
}
