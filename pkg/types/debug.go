package types

import "time"

// SynthJob exposes per-segment TTS/synthesis state for debug dashboards.
type SynthJob struct {
	ID               string     `json:"id"`
	BookID           string     `json:"book_id"`
	SegmentID        string     `json:"segment_id"`
	Status           string     `json:"status"` // queued, running, completed, failed, retrying, cancelled, exhausted, derived
	Provider         string     `json:"provider,omitempty"`
	VoiceID          string     `json:"voice_id,omitempty"`
	VoiceDescription string     `json:"voice_description,omitempty"`
	QueuedAt         *time.Time `json:"queued_at,omitempty"`
	StartedAt        *time.Time `json:"started_at,omitempty"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
	DurationMS       int64      `json:"duration_ms,omitempty"`
	OutputPath       string     `json:"output_path,omitempty"`
	OutputFormat     string     `json:"output_format,omitempty"`
	OutputBytes      int64      `json:"output_bytes,omitempty"`
	RetryCount       int        `json:"retry_count"`
	Error            string     `json:"error,omitempty"`
	Worker           string     `json:"worker,omitempty"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// PlaybackEvent records read/listen/playback telemetry from the client.
type PlaybackEvent struct {
	ID                  string    `json:"id"`
	BookID              string    `json:"book_id"`
	SegmentID           string    `json:"segment_id,omitempty"`
	UserID              string    `json:"user_id"`
	EventType           string    `json:"event_type"` // book_opened, segment_opened, read, play, pause, complete, failed, skipped
	PlaybackPositionSec float64   `json:"playback_position_sec,omitempty"`
	DurationSec         float64   `json:"duration_sec,omitempty"`
	Success             bool      `json:"success,omitempty"`
	Error               string    `json:"error,omitempty"`
	Client              string    `json:"client,omitempty"`
	Device              string    `json:"device,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
}

// AudioArtifactValidation reports whether a segment audio artifact exists and looks usable.
type AudioArtifactValidation struct {
	BookID      string    `json:"book_id"`
	SegmentID   string    `json:"segment_id"`
	Status      string    `json:"status"` // attached, missing, stale, invalid
	Format      string    `json:"format,omitempty"`
	Path        string    `json:"path,omitempty"`
	Bytes       int64     `json:"bytes,omitempty"`
	ContentType string    `json:"content_type,omitempty"`
	Error       string    `json:"error,omitempty"`
	CheckedAt   time.Time `json:"checked_at"`
}

// UserProgress summarizes the current single-user journey through a book.
type UserProgress struct {
	BookID                  string    `json:"book_id"`
	UserID                  string    `json:"user_id"`
	JourneyState            string    `json:"journey_state"` // not_started, opened, reading, listening, stuck, abandoned, completed
	CanRead                 bool      `json:"can_read"`
	CanListenAll            bool      `json:"can_listen_all"`
	LastOpenedSegmentID     string    `json:"last_opened_segment_id,omitempty"`
	LastReadSegmentID       string    `json:"last_read_segment_id,omitempty"`
	LastListenedSegmentID   string    `json:"last_listened_segment_id,omitempty"`
	StuckSegmentID          string    `json:"stuck_segment_id,omitempty"`
	PlaybackFailures        int       `json:"playback_failures"`
	CompletedListenSegments int       `json:"completed_listen_segments"`
	TotalSegments           int       `json:"total_segments"`
	UpdatedAt               time.Time `json:"updated_at"`
}

// DebugEvent is the unified operational event shape used by polling and SSE endpoints.
type DebugEvent struct {
	ID        string    `json:"id"`
	BookID    string    `json:"book_id,omitempty"`
	SegmentID string    `json:"segment_id,omitempty"`
	Scope     string    `json:"scope"`    // system, book, segment, synth, user, health
	Severity  string    `json:"severity"` // info, success, warning, danger
	Title     string    `json:"title"`
	Detail    string    `json:"detail,omitempty"`
	Source    string    `json:"source,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
