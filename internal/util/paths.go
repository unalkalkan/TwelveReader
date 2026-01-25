package util

import (
	"fmt"
	"path/filepath"
)

// GetAudioPath returns the storage path for a segment's audio file
func GetAudioPath(bookID, segmentID, format string) string {
	return filepath.Join("books", bookID, "audio", fmt.Sprintf("%s.%s", segmentID, format))
}

// AudioFormats returns the list of supported audio formats to try
func AudioFormats() []string {
	return []string{"wav", "mp3", "ogg", "flac"}
}
