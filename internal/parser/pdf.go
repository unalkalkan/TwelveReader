package parser

import (
	"bytes"
	"compress/zlib"
	"context"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/unalkalkan/TwelveReader/pkg/types"
)

type PDFParser struct{}

var (
	pdfStreamRe       = regexp.MustCompile(`(?m)(?:^|\r?\n)stream\r?\n`)
	pdfEndStreamRe    = regexp.MustCompile(`\r?\nendstream`)
	flateDecodeRe     = regexp.MustCompile(`/FlateDecode`)
	filterRe          = regexp.MustCompile(`/Filter\s*/FlateDecode`)
	chapterHeadingRe  = regexp.MustCompile(`(?i)^chapter\s+([0-9]+|[ivxlcdm]+|one|two|three|four|five|six|seven|eight|nine|ten|eleven|twelve)\b`)
	numberedHeadingRe = regexp.MustCompile(`^\d{1,3}\.\s+\S`)
)

func NewPDFParser() *PDFParser {
	return &PDFParser{}
}

func (p *PDFParser) Parse(ctx context.Context, data []byte) ([]*types.Chapter, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("pdf: empty data")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if !bytes.HasPrefix(data, []byte("%PDF")) {
		return nil, fmt.Errorf("pdf: invalid header, not a PDF file")
	}

	streams := extractStreams(data)
	if len(streams) == 0 {
		return nil, fmt.Errorf("pdf: no content streams found")
	}

	var allText []string
	for _, stream := range streams {
		texts := extractTextFromStream(stream)
		allText = append(allText, texts...)
	}

	if len(allText) == 0 {
		return nil, fmt.Errorf("pdf: no extractable text found")
	}
	allText = reflowExtractedPDFParagraphs(allText)

	chapter := &types.Chapter{
		ID:         "chapter_001",
		Number:     1,
		Title:      "PDF Content",
		TOCPath:    []string{"PDF Content"},
		Paragraphs: allText,
	}

	return []*types.Chapter{chapter}, nil
}

func extractStreams(data []byte) [][]byte {
	var streams [][]byte

	positions := pdfStreamRe.FindAllIndex(data, -1)
	for _, pos := range positions {
		start := pos[1]
		endMatch := pdfEndStreamRe.FindIndex(data[start:])
		if endMatch == nil || endMatch[0] == 0 {
			continue
		}
		end := start + endMatch[0]
		rawStream := data[start:end]

		// Check if this stream uses FlateDecode by looking at the
		// stream dictionary (the object bytes before the "stream" keyword)
		streamDictStart := pos[0] - 500
		if streamDictStart < 0 {
			streamDictStart = 0
		}
		streamDict := data[streamDictStart:pos[0]]

		if usesFlateDecode(streamDict) {
			decompressed, err := decompressFlate(rawStream)
			if err == nil && len(decompressed) > 0 {
				streams = append(streams, decompressed)
				continue
			}
			// On decompression failure, include raw data as fallback
		}

		streams = append(streams, rawStream)
	}

	return streams
}

// usesFlateDecode checks if the stream dictionary contains a FlateDecode filter
func usesFlateDecode(dict []byte) bool {
	return filterRe.Match(dict) || flateDecodeRe.Match(dict)
}

// decompressFlate decompresses a zlib/deflate (FlateDecode) compressed stream
func decompressFlate(data []byte) ([]byte, error) {
	reader, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("zlib decompression failed: %w", err)
	}
	defer reader.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		return nil, fmt.Errorf("reading decompressed data failed: %w", err)
	}
	return buf.Bytes(), nil
}

func extractTextFromStream(stream []byte) []string {
	content := string(stream)

	var paragraphs []string
	var currentLine strings.Builder

	inTextBlock := false
	i := 0
	for i < len(content) {
		if hasPDFOperatorAt(content, i, "BT") {
			inTextBlock = true
			i += 2
			continue
		}
		if hasPDFOperatorAt(content, i, "ET") {
			inTextBlock = false
			if currentLine.Len() > 0 {
				text := strings.TrimSpace(currentLine.String())
				if text != "" {
					paragraphs = append(paragraphs, text)
				}
				currentLine.Reset()
			}
			i += 2
			continue
		}

		if inTextBlock && i < len(content) && content[i] == '(' {
			text, endIdx := parseLiteralString(content, i)
			if text != "" {
				if currentLine.Len() > 0 {
					currentLine.WriteString(" ")
				}
				currentLine.WriteString(text)
			}
			i = endIdx
			continue
		}

		if inTextBlock && content[i] == 'T' && i+1 < len(content) {
			if content[i+1] == 'j' || content[i+1] == 'J' {
				i += 2
				continue
			}
			if content[i+1] == 'd' || content[i+1] == 'D' || content[i+1] == 'm' || content[i+1] == 'M' || content[i+1] == 'f' || content[i+1] == 'F' {
				if currentLine.Len() > 0 {
					text := strings.TrimSpace(currentLine.String())
					if text != "" {
						paragraphs = append(paragraphs, text)
					}
					currentLine.Reset()
				}
				i += 2
				continue
			}
		}

		i++
	}

	if currentLine.Len() > 0 {
		text := strings.TrimSpace(currentLine.String())
		if text != "" {
			paragraphs = append(paragraphs, text)
		}
	}

	return paragraphs
}

func reflowExtractedPDFParagraphs(lines []string) []string {
	reflowed := make([]string, 0, len(lines))
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}

		if len(reflowed) == 0 || shouldStartNewPDFParagraph(reflowed[len(reflowed)-1], line) {
			reflowed = append(reflowed, line)
			continue
		}

		reflowed[len(reflowed)-1] = joinPDFWrappedLines(reflowed[len(reflowed)-1], line)
	}
	return reflowed
}

func shouldStartNewPDFParagraph(previous, current string) bool {
	previous = strings.TrimSpace(previous)
	current = strings.TrimSpace(current)
	if previous == "" || current == "" {
		return true
	}
	if hasUnclosedQuote(previous) {
		return false
	}
	if isPDFHeadingLine(previous) || isPDFHeadingLine(current) {
		return true
	}
	if endsWithSentenceBoundary(previous) {
		return true
	}
	return false
}

func joinPDFWrappedLines(previous, current string) string {
	previous = strings.TrimSpace(previous)
	current = strings.TrimSpace(current)
	if previous == "" {
		return current
	}
	if current == "" {
		return previous
	}
	if strings.HasSuffix(previous, "-") && !strings.HasSuffix(previous, "--") {
		return strings.TrimSuffix(previous, "-") + current
	}
	return previous + " " + current
}

func isPDFHeadingLine(line string) bool {
	line = strings.TrimSpace(line)
	if line == "" {
		return false
	}
	if chapterHeadingRe.MatchString(line) || numberedHeadingRe.MatchString(line) {
		return true
	}
	wordCount := len(strings.Fields(line))
	if wordCount > 0 && wordCount <= 8 && !endsWithSentenceBoundary(line) && isMostlyTitleCase(line) {
		return true
	}
	return false
}

func isMostlyTitleCase(line string) bool {
	words := strings.Fields(line)
	if len(words) == 0 {
		return false
	}

	casedWords := 0
	titleLikeWords := 0
	for _, word := range words {
		trimmed := strings.Trim(word, `"'“”‘’()[]{}.,:;!?`)
		if trimmed == "" {
			continue
		}
		runes := []rune(trimmed)
		firstLetter := rune(0)
		for _, r := range runes {
			if unicode.IsLetter(r) {
				firstLetter = r
				break
			}
		}
		if firstLetter == 0 {
			continue
		}
		casedWords++
		if unicode.IsUpper(firstLetter) || len([]rune(trimmed)) == 1 {
			titleLikeWords++
		}
	}
	return casedWords > 0 && titleLikeWords*100/casedWords >= 70
}

func endsWithSentenceBoundary(line string) bool {
	line = strings.TrimSpace(line)
	if line == "" {
		return false
	}

	for len(line) > 0 {
		r, size := lastRune(line)
		if r == '"' || r == '\'' || r == '”' || r == '’' || r == ')' || r == ']' || r == '}' {
			line = strings.TrimSpace(line[:len(line)-size])
			continue
		}
		break
	}
	if line == "" {
		return false
	}
	r, _ := lastRune(line)
	return r == '.' || r == '!' || r == '?' || r == ':'
}

func hasUnclosedQuote(line string) bool {
	straightDoubleQuotes := 0
	leftDoubleQuotes := 0
	rightDoubleQuotes := 0
	for _, r := range line {
		switch r {
		case '"':
			straightDoubleQuotes++
		case '“':
			leftDoubleQuotes++
		case '”':
			rightDoubleQuotes++
		}
	}
	if straightDoubleQuotes%2 == 1 {
		return true
	}
	return leftDoubleQuotes > rightDoubleQuotes
}

func lastRune(s string) (rune, int) {
	for i := len(s); i > 0; {
		r, size := rune(s[i-1]), 1
		if r >= utf8.RuneSelf {
			r, size = utf8.DecodeLastRuneInString(s[:i])
		}
		return r, size
	}
	return 0, 0
}

func hasPDFOperatorAt(content string, index int, op string) bool {
	if index < 0 || index+len(op) > len(content) || content[index:index+len(op)] != op {
		return false
	}
	beforeOK := index == 0 || isPDFDelimiter(content[index-1])
	afterIndex := index + len(op)
	afterOK := afterIndex == len(content) || isPDFDelimiter(content[afterIndex])
	return beforeOK && afterOK
}

func isPDFDelimiter(ch byte) bool {
	switch ch {
	case ' ', '\t', '\n', '\r', '\f', '[', ']', '<', '>', '/', '(', ')':
		return true
	default:
		return false
	}
}

func parseLiteralString(content string, start int) (string, int) {
	if content[start] != '(' {
		return "", start + 1
	}

	var buf strings.Builder
	parenDepth := 1
	i := start + 1

	for i < len(content) && parenDepth > 0 {
		ch := content[i]
		if ch == '\\' && i+1 < len(content) {
			next := content[i+1]
			switch next {
			case 'n':
				buf.WriteByte('\n')
				i += 2
			case 'r':
				buf.WriteByte('\r')
				i += 2
			case 't':
				buf.WriteByte('\t')
				i += 2
			case 'b':
				buf.WriteByte('\b')
				i += 2
			case 'f':
				buf.WriteByte('\f')
				i += 2
			case '(':
				buf.WriteByte('(')
				i += 2
			case ')':
				buf.WriteByte(')')
				i += 2
			case '\\':
				buf.WriteByte('\\')
				i += 2
			default:
				if next >= '0' && next <= '7' {
					end := i + 1
					for end < len(content) && end < i+4 && content[end] >= '0' && content[end] <= '7' {
						end++
					}
					if value, err := strconv.ParseInt(content[i+1:end], 8, 32); err == nil {
						buf.WriteRune(rune(value))
					}
					i = end
				} else {
					buf.WriteByte(next)
					i += 2
				}
			}
			continue
		}
		if ch == '(' {
			parenDepth++
			buf.WriteByte(ch)
			i++
			continue
		}
		if ch == ')' {
			parenDepth--
			if parenDepth == 0 {
				i++
				break
			}
			buf.WriteByte(ch)
			i++
			continue
		}
		buf.WriteByte(ch)
		i++
	}

	return buf.String(), i
}

func (p *PDFParser) SupportedFormats() []string {
	return []string{"pdf"}
}
