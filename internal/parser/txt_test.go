package parser

import (
	"context"
	"strings"
	"testing"
)

func TestTXTParser_Parse(t *testing.T) {
	parser := NewTXTParser()
	ctx := context.Background()

	t.Run("Simple text", func(t *testing.T) {
		data := []byte(`This is the first paragraph.

This is the second paragraph with multiple sentences. It continues here.

This is the third paragraph.`)

		chapters, err := parser.Parse(ctx, data)
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}

		if len(chapters) != 1 {
			t.Fatalf("Expected 1 chapter, got %d", len(chapters))
		}

		chapter := chapters[0]
		if len(chapter.Paragraphs) != 3 {
			t.Fatalf("Expected 3 paragraphs, got %d", len(chapter.Paragraphs))
		}

		if !strings.Contains(chapter.Paragraphs[0], "first paragraph") {
			t.Errorf("First paragraph doesn't contain expected text")
		}
	})

	t.Run("Text with chapter headings", func(t *testing.T) {
		data := []byte(`Initial content before chapters.

CHAPTER ONE

This is the first chapter.

CHAPTER TWO

This is the second chapter.`)

		chapters, err := parser.Parse(ctx, data)
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}

		if len(chapters) < 2 {
			t.Fatalf("Expected at least 2 chapters, got %d", len(chapters))
		}

		// Find chapters by searching for content
		foundChapterOne := false
		foundChapterTwo := false
		for _, ch := range chapters {
			if strings.Contains(ch.Title, "CHAPTER ONE") {
				foundChapterOne = true
			}
			if strings.Contains(ch.Title, "CHAPTER TWO") {
				foundChapterTwo = true
			}
		}

		if !foundChapterOne {
			t.Error("CHAPTER ONE not found")
		}
		if !foundChapterTwo {
			t.Error("CHAPTER TWO not found")
		}
	})

	t.Run("Empty file", func(t *testing.T) {
		data := []byte("")

		_, err := parser.Parse(ctx, data)
		if err == nil {
			t.Error("Expected error for empty file")
		}
	})

	t.Run("Multiple consecutive empty lines", func(t *testing.T) {
		data := []byte(`First paragraph.


Second paragraph.`)

		chapters, err := parser.Parse(ctx, data)
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}

		if len(chapters[0].Paragraphs) != 2 {
			t.Fatalf("Expected 2 paragraphs, got %d", len(chapters[0].Paragraphs))
		}
	})
}

func TestTXTParser_isChapterHeading(t *testing.T) {
	parser := NewTXTParser()

	tests := []struct {
		name     string
		line     string
		expected bool
	}{
		{"Chapter with number", "Chapter 1", true},
		{"Chapter uppercase", "CHAPTER ONE", true},
		{"Part heading", "Part I", true},
		{"Section heading", "Section A", true},
		{"Prologue", "Prologue", true},
		{"Epilogue", "EPILOGUE", true},
		{"Introduction", "Introduction", true},
		{"Regular text", "This is a regular sentence.", false},
		{"Empty line", "", false},
		{"Short all caps", "THE END", true},
		{"Title case short", "The Beginning", true},
		{"Long title case", "This Is A Very Long Line That Should Not Be Considered A Title", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.isChapterHeading(tt.line)
			if result != tt.expected {
				t.Errorf("isChapterHeading(%q) = %v, expected %v", tt.line, result, tt.expected)
			}
		})
	}
}

func TestTXTParser_SupportedFormats(t *testing.T) {
	parser := NewTXTParser()
	formats := parser.SupportedFormats()

	if len(formats) != 1 {
		t.Fatalf("Expected 1 format, got %d", len(formats))
	}

	if formats[0] != "txt" {
		t.Errorf("Expected format 'txt', got %q", formats[0])
	}
}
