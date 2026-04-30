package parser

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestPDFParser_SupportedFormats(t *testing.T) {
	p := NewPDFParser()
	formats := p.SupportedFormats()
	if len(formats) != 1 || formats[0] != "pdf" {
		t.Errorf("Expected [pdf], got %v", formats)
	}
}

func TestPDFParser_Parse_SimpleText(t *testing.T) {
	p := NewPDFParser()
	ctx := context.Background()

	pdfData := buildSimplePDF([]pdfPage{
		{textStrings: []string{"Hello World", "Second line"}},
	})

	chapters, err := p.Parse(ctx, pdfData)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(chapters) == 0 {
		t.Fatal("Expected at least 1 chapter")
	}

	allText := strings.Join(chapters[0].Paragraphs, " ")
	if !strings.Contains(allText, "Hello World") {
		t.Errorf("Expected 'Hello World' in extracted text, got: %v", chapters[0].Paragraphs)
	}
	if !strings.Contains(allText, "Second line") {
		t.Errorf("Expected 'Second line' in extracted text, got: %v", chapters[0].Paragraphs)
	}
}

func TestPDFParser_Parse_MultiplePages(t *testing.T) {
	p := NewPDFParser()
	ctx := context.Background()

	pdfData := buildSimplePDF([]pdfPage{
		{textStrings: []string{"Page one text"}},
		{textStrings: []string{"Page two text"}},
	})

	chapters, err := p.Parse(ctx, pdfData)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(chapters) == 0 {
		t.Fatal("Expected at least 1 chapter")
	}

	allText := ""
	for _, ch := range chapters {
		allText += strings.Join(ch.Paragraphs, " ")
	}
	if !strings.Contains(allText, "Page one text") {
		t.Errorf("Expected 'Page one text' in extracted text")
	}
	if !strings.Contains(allText, "Page two text") {
		t.Errorf("Expected 'Page two text' in extracted text")
	}
}

func TestPDFParser_Parse_EmbeddedParentheses(t *testing.T) {
	p := NewPDFParser()
	ctx := context.Background()

	pdfData := buildSimplePDF([]pdfPage{
		{textStrings: []string{"(Hello)"}},
	})

	chapters, err := p.Parse(ctx, pdfData)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(chapters) == 0 {
		t.Fatal("Expected at least 1 chapter")
	}

	allText := strings.Join(chapters[0].Paragraphs, " ")
	if !strings.Contains(allText, "(Hello)") {
		t.Errorf("Expected '(Hello)' with parentheses in extracted text, got: %v", chapters[0].Paragraphs)
	}
}

func TestPDFParser_Parse_EscapedLiteralCharacters(t *testing.T) {
	p := NewPDFParser()
	ctx := context.Background()

	pdfData := buildSimplePDF([]pdfPage{
		{textStrings: []string{"Line one\nLine two\tTabbed\\Slash", "Octal: A"}},
	})

	chapters, err := p.Parse(ctx, pdfData)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	allText := strings.Join(chapters[0].Paragraphs, " ")
	for _, expected := range []string{"Line one\nLine two\tTabbed\\Slash", "Octal: A"} {
		if !strings.Contains(allText, expected) {
			t.Errorf("Expected %q in extracted text, got: %q", expected, allText)
		}
	}
}

func TestPDFParser_Parse_NotPlaceholder(t *testing.T) {
	p := NewPDFParser()
	ctx := context.Background()

	pdfData := buildSimplePDF([]pdfPage{
		{textStrings: []string{"Real text content"}},
	})

	chapters, err := p.Parse(ctx, pdfData)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	for _, ch := range chapters {
		for _, para := range ch.Paragraphs {
			if strings.Contains(para, "not yet implemented") ||
				strings.Contains(para, "Not Yet Implemented") ||
				strings.Contains(para, "requires external libraries") {
				t.Errorf("Paragraph contains placeholder text: %q", para)
			}
		}
	}
}

func TestPDFParser_Parse_InvalidInput(t *testing.T) {
	p := NewPDFParser()
	ctx := context.Background()

	t.Run("nil data", func(t *testing.T) {
		_, err := p.Parse(ctx, nil)
		if err == nil {
			t.Error("Expected error for nil data")
		}
	})

	t.Run("empty data", func(t *testing.T) {
		_, err := p.Parse(ctx, []byte{})
		if err == nil {
			t.Error("Expected error for empty data")
		}
	})

	t.Run("random bytes", func(t *testing.T) {
		_, err := p.Parse(ctx, []byte("this is not a pdf at all"))
		if err == nil {
			t.Error("Expected error for non-PDF data")
		}
	})

	t.Run("PDF header no objects", func(t *testing.T) {
		_, err := p.Parse(ctx, []byte("%PDF-1.4\n%%EOF\n"))
		if err == nil {
			t.Error("Expected error for PDF with no extractable text")
		}
	})
}

func TestPDFParser_Parse_ChapterFields(t *testing.T) {
	p := NewPDFParser()
	ctx := context.Background()

	pdfData := buildSimplePDF([]pdfPage{
		{textStrings: []string{"Some text"}},
	})

	chapters, err := p.Parse(ctx, pdfData)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(chapters) == 0 {
		t.Fatal("Expected at least 1 chapter")
	}

	ch := chapters[0]
	if ch.ID == "" {
		t.Error("Chapter ID must not be empty")
	}
	if ch.Number < 1 {
		t.Errorf("Chapter Number must be >= 1, got %d", ch.Number)
	}
	if len(ch.TOCPath) == 0 {
		t.Error("Chapter TOCPath must not be empty")
	}
}

type pdfPage struct {
	textStrings []string
}

func buildSimplePDF(pages []pdfPage) []byte {
	var b strings.Builder

	b.WriteString("%PDF-1.4\n")

	b.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")

	b.WriteString("2 0 obj\n<< /Type /Pages /Kids [")
	for i := range pages {
		b.WriteString(fmt.Sprintf(" %d 0 R", i+3))
	}
	b.WriteString(" ] /Count ")
	b.WriteString(fmt.Sprintf("%d", len(pages)))
	b.WriteString(" >>\nendobj\n")

	for i, page := range pages {
		contentStr := ""
		for _, ts := range page.textStrings {
			escaped := strings.ReplaceAll(ts, "\\", "\\\\")
			escaped = strings.ReplaceAll(escaped, "\n", "\\n")
			escaped = strings.ReplaceAll(escaped, "\t", "\\t")
			escaped = strings.ReplaceAll(escaped, "Octal: A", "Octal: \\101")
			escaped = strings.ReplaceAll(escaped, "(", "\\(")
			escaped = strings.ReplaceAll(escaped, ")", "\\)")
			contentStr += fmt.Sprintf("BT /F1 12 Tf 100 700 Td (%s) Tj ET\n", escaped)
		}
		streamBytes := []byte(contentStr)

		b.WriteString(fmt.Sprintf("%d 0 obj\n", i+3))
		b.WriteString("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents ")
		b.WriteString(fmt.Sprintf("%d 0 R >>\n", len(pages)+3+i))
		b.WriteString("endobj\n")

		b.WriteString(fmt.Sprintf("%d 0 obj\n", len(pages)+3+i))
		b.WriteString(fmt.Sprintf("<< /Length %d >>\n", len(streamBytes)))
		b.WriteString("stream\n")
		b.Write(streamBytes)
		b.WriteString("\nendstream\nendobj\n")
	}

	b.WriteString(fmt.Sprintf("%d 0 obj\n", len(pages)*2+3))
	b.WriteString("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\nendobj\n")

	objCount := len(pages)*2 + 4
	b.WriteString(fmt.Sprintf("%d 0 obj\n", objCount))
	b.WriteString("<< /Font << /F1 ")
	b.WriteString(fmt.Sprintf("%d 0 R", len(pages)*2+3))
	b.WriteString(" >> >>\nendobj\n")

	b.WriteString("xref\n0 ")
	b.WriteString(fmt.Sprintf("%d\n", objCount+1))
	b.WriteString("0000000000 65535 f \n")
	for i := 0; i < objCount; i++ {
		b.WriteString("0000000000 00000 n \n")
	}

	b.WriteString("trailer\n<< /Size ")
	b.WriteString(fmt.Sprintf("%d", objCount+1))
	b.WriteString(" /Root 1 0 R >>\nstartxref\n0\n%%EOF\n")

	return []byte(b.String())
}
