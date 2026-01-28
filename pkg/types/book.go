package types

import "time"

// Book represents a book being processed
type Book struct {
	ID            string    `json:"id"`
	Title         string    `json:"title"`
	Author        string    `json:"author"`
	Language      string    `json:"language"` // ISO-639-1 code
	UploadedAt    time.Time `json:"uploaded_at"`
	Status        string    `json:"status"`      // "uploaded", "parsing", "segmenting", "voice_mapping", "ready", "error"
	OrigFormat    string    `json:"orig_format"` // "pdf", "epub", "txt"
	Error         string    `json:"error,omitempty"`
	TotalChapters int       `json:"total_chapters"`
	TotalSegments int       `json:"total_segments"`
}

// Chapter represents a chapter in a book
type Chapter struct {
	ID         string   `json:"id"`
	BookID     string   `json:"book_id"`
	Number     int      `json:"number"`
	Title      string   `json:"title"`
	TOCPath    []string `json:"toc_path"` // Hierarchical breadcrumbs
	Paragraphs []string `json:"paragraphs"`
}

// Segment represents a processed text segment with metadata
type Segment struct {
	ID               string          `json:"id"`
	BookID           string          `json:"book_id"`
	Chapter          string          `json:"chapter"`
	TOCPath          []string        `json:"toc_path"`
	Text             string          `json:"text"`
	Language         string          `json:"language"`
	Person           string          `json:"person"`
	VoiceDescription string          `json:"voice_description"`
	VoiceID          string          `json:"voice_id,omitempty"` // Set after voice mapping
	Timestamps       *TimestampData  `json:"timestamps,omitempty"`
	SourceContext    *SourceContext  `json:"source_context,omitempty"`
	Processing       *ProcessingInfo `json:"processing"`
}

// Voice represents a TTS voice with metadata
type Voice struct {
	ID          string   `json:"id"`          // Provider-specific voice ID
	Name        string   `json:"name"`        // Human-readable name
	Languages   []string `json:"languages"`   // Supported language codes (ISO-639-1)
	Gender      string   `json:"gender"`      // "male", "female", "neutral", or empty
	Accent      string   `json:"accent"`      // Regional accent (e.g., "british", "american")
	Description string   `json:"description"` // Additional description
}

// TimestampData holds word-level timestamps
type TimestampData struct {
	Precision string          `json:"precision"` // "word" or "sentence"
	Items     []TimestampItem `json:"items"`
}

// TimestampItem represents timing for a word or sentence
type TimestampItem struct {
	Word  string  `json:"word"`
	Start float64 `json:"start"` // seconds
	End   float64 `json:"end"`   // seconds
}

// SourceContext holds references to neighboring paragraphs
type SourceContext struct {
	PrevParagraphID string `json:"prev_paragraph_id,omitempty"`
	NextParagraphID string `json:"next_paragraph_id,omitempty"`
}

// ProcessingInfo holds audit metadata
type ProcessingInfo struct {
	SegmenterVersion string    `json:"segmenter_version"`
	TTSProvider      string    `json:"tts_provider,omitempty"`
	GeneratedAt      time.Time `json:"generated_at"`
}

// VoiceMap represents persona-to-voice assignments
type VoiceMap struct {
	BookID  string        `json:"book_id"`
	Persons []PersonVoice `json:"persons"`
}

// PersonVoice maps a persona to a provider voice
type PersonVoice struct {
	ID            string `json:"id"`             // Persona identifier
	ProviderVoice string `json:"provider_voice"` // Provider-specific voice ID
}

// ProcessingStatus represents the current state of book processing
type ProcessingStatus struct {
	BookID         string    `json:"book_id"`
	Status         string    `json:"status"`
	Stage          string    `json:"stage"`    // Current processing stage
	Progress       float64   `json:"progress"` // 0-100
	TotalChapters  int       `json:"total_chapters"`
	ParsedChapters int       `json:"parsed_chapters"`
	TotalSegments  int       `json:"total_segments"`
	Error          string    `json:"error,omitempty"`
	UpdatedAt      time.Time `json:"updated_at"`
}
