package parser

import (
	"fmt"
	"strings"
)

// DefaultFactory creates parsers for supported formats
type DefaultFactory struct {
	parsers map[string]Parser
}

// NewFactory creates a new parser factory with default parsers
func NewFactory() Factory {
	f := &DefaultFactory{
		parsers: make(map[string]Parser),
	}
	
	// Register default parsers
	f.registerParser(NewTXTParser())
	f.registerParser(NewPDFParser())
	f.registerParser(NewEPUBParser())
	
	return f
}

// registerParser registers a parser for its supported formats
func (f *DefaultFactory) registerParser(p Parser) {
	for _, format := range p.SupportedFormats() {
		f.parsers[strings.ToLower(format)] = p
	}
}

// GetParser returns a parser for the given format
func (f *DefaultFactory) GetParser(format string) (Parser, error) {
	format = strings.ToLower(format)
	parser, ok := f.parsers[format]
	if !ok {
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
	return parser, nil
}
