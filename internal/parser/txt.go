package parser

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/unalkalkan/TwelveReader/pkg/types"
)

// TXTParser parses plain text files
type TXTParser struct{}

const (
	// paragraphBreakEmptyLines is the number of consecutive empty lines needed to break a paragraph
	paragraphBreakEmptyLines = 1
)

// NewTXTParser creates a new TXT parser
func NewTXTParser() *TXTParser {
	return &TXTParser{}
}

// Parse extracts chapters and text from a TXT file
func (p *TXTParser) Parse(ctx context.Context, data []byte) ([]*types.Chapter, error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))

	chapters := make([]*types.Chapter, 0)
	currentChapter := &types.Chapter{
		ID:         "chapter_001",
		Number:     1,
		Title:      "Main Content",
		TOCPath:    []string{"Main Content"},
		Paragraphs: make([]string, 0),
	}

	var currentParagraph strings.Builder
	lineCount := 0
	emptyLineCount := 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		lineCount++

		// Check if this might be a chapter heading
		if p.isChapterHeading(line) && len(currentChapter.Paragraphs) > 0 {
			// Save current paragraph if any
			if currentParagraph.Len() > 0 {
				currentChapter.Paragraphs = append(currentChapter.Paragraphs, currentParagraph.String())
				currentParagraph.Reset()
			}

			// Save current chapter
			chapters = append(chapters, currentChapter)

			// Start new chapter
			chapterNum := len(chapters) + 1
			currentChapter = &types.Chapter{
				ID:         fmt.Sprintf("chapter_%03d", chapterNum),
				Number:     chapterNum,
				Title:      line,
				TOCPath:    []string{line},
				Paragraphs: make([]string, 0),
			}
			emptyLineCount = 0
			continue
		}

		// Empty line - potential paragraph break
		if line == "" {
			emptyLineCount++
			if currentParagraph.Len() > 0 && emptyLineCount >= paragraphBreakEmptyLines {
				currentChapter.Paragraphs = append(currentChapter.Paragraphs, currentParagraph.String())
				currentParagraph.Reset()
			}
			continue
		}

		// Add line to current paragraph
		emptyLineCount = 0
		if currentParagraph.Len() > 0 {
			currentParagraph.WriteString(" ")
		}
		currentParagraph.WriteString(line)
	}

	// Save last paragraph if any
	if currentParagraph.Len() > 0 {
		currentChapter.Paragraphs = append(currentChapter.Paragraphs, currentParagraph.String())
	}

	// Save last chapter if it has content
	if len(currentChapter.Paragraphs) > 0 {
		chapters = append(chapters, currentChapter)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading text: %w", err)
	}

	// Ensure we have at least one chapter
	if len(chapters) == 0 {
		return nil, fmt.Errorf("no content found in text file")
	}

	return chapters, nil
}

// isChapterHeading checks if a line looks like a chapter heading
func (p *TXTParser) isChapterHeading(line string) bool {
	if len(line) == 0 {
		return false
	}

	lower := strings.ToLower(line)

	// Check for common chapter patterns
	patterns := []string{
		"chapter ",
		"part ",
		"section ",
		"prologue",
		"epilogue",
		"introduction",
	}

	for _, pattern := range patterns {
		if strings.HasPrefix(lower, pattern) {
			return true
		}
	}

	// Check if it's a short line (potential title) - all caps or title case
	if len(line) < 60 && (isAllCaps(line) || isTitleCase(line)) {
		return true
	}

	return false
}

// isAllCaps checks if string is all uppercase (ignoring numbers and punctuation)
func isAllCaps(s string) bool {
	hasLetter := false
	for _, r := range s {
		if r >= 'a' && r <= 'z' {
			return false
		}
		if r >= 'A' && r <= 'Z' {
			hasLetter = true
		}
	}
	return hasLetter
}

// isTitleCase checks if string is in title case
func isTitleCase(s string) bool {
	words := strings.Fields(s)
	if len(words) == 0 {
		return false
	}

	titleCaseCount := 0
	for _, word := range words {
		if len(word) > 0 {
			first := rune(word[0])
			if first >= 'A' && first <= 'Z' {
				titleCaseCount++
			}
		}
	}

	// Most words should be title case
	return float64(titleCaseCount)/float64(len(words)) > 0.7
}

// SupportedFormats returns the formats this parser supports
func (p *TXTParser) SupportedFormats() []string {
	return []string{"txt"}
}
