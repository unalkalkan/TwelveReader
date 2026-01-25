package parser

import (
	"context"
	"fmt"

	"github.com/unalkalkan/TwelveReader/pkg/types"
)

// EPUBParser parses ePUB files
type EPUBParser struct{}

// NewEPUBParser creates a new ePUB parser
func NewEPUBParser() *EPUBParser {
	return &EPUBParser{}
}

// Parse extracts chapters and text from an ePUB file
// This is a stub implementation - a real implementation would use an ePUB library
func (p *EPUBParser) Parse(ctx context.Context, data []byte) ([]*types.Chapter, error) {
	// For now, return a stub chapter indicating ePUB parsing is not yet fully implemented
	// In a real implementation, we would use a library like go-epub or bmaupin/go-epub

	chapter := &types.Chapter{
		ID:      "chapter_001",
		Number:  1,
		Title:   "ePUB Content (Parsing Not Yet Implemented)",
		TOCPath: []string{"ePUB Content"},
		Paragraphs: []string{
			"ePUB parsing requires external libraries and is not yet implemented.",
			"Future implementation will use libraries like github.com/bmaupin/go-epub or similar.",
			fmt.Sprintf("ePUB file size: %d bytes", len(data)),
		},
	}

	return []*types.Chapter{chapter}, nil
}

// SupportedFormats returns the formats this parser supports
func (p *EPUBParser) SupportedFormats() []string {
	return []string{"epub"}
}
