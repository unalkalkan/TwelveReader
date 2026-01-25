package parser

import (
	"context"

	"github.com/unalkalkan/TwelveReader/pkg/types"
)

// Parser defines the interface for document parsers
type Parser interface {
	// Parse extracts chapters and text from the document
	Parse(ctx context.Context, data []byte) ([]*types.Chapter, error)

	// SupportedFormats returns the file formats this parser supports
	SupportedFormats() []string
}

// Factory creates parsers for different formats
type Factory interface {
	// GetParser returns a parser for the given format
	GetParser(format string) (Parser, error)
}
