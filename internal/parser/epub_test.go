package parser

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
)

func createEpubZip(files map[string]string) []byte {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for path, content := range files {
		f, _ := w.Create(path)
		f.Write([]byte(content))
	}
	w.Close()
	return buf.Bytes()
}

func TestEPUBParser_SupportedFormats(t *testing.T) {
	p := NewEPUBParser()
	formats := p.SupportedFormats()
	if len(formats) != 1 || formats[0] != "epub" {
		t.Errorf("Expected [epub], got %v", formats)
	}
}

func TestEPUBParser_Parse_ValidEpubWithSpine(t *testing.T) {
	p := NewEPUBParser()
	ctx := context.Background()

	containerXML := `<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`

	opf := `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="uid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Test Book</dc:title>
  </metadata>
  <manifest>
    <item id="ch1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>
    <item id="ch2" href="chapter2.xhtml" media-type="application/xhtml+xml"/>
    <item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/>
  </manifest>
  <spine>
    <itemref idref="ch1"/>
    <itemref idref="ch2"/>
  </spine>
</package>`

	ch1 := `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Chapter One</title></head>
<body>
  <h1>Chapter One</h1>
  <p>First paragraph of chapter one.</p>
  <p>Second paragraph of chapter one.</p>
</body>
</html>`

	ch2 := `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Chapter Two</title></head>
<body>
  <h1>Chapter Two</h1>
  <p>First paragraph of chapter two.</p>
</body>
</html>`

	data := createEpubZip(map[string]string{
		"META-INF/container.xml": containerXML,
		"OEBPS/content.opf":      opf,
		"OEBPS/chapter1.xhtml":   ch1,
		"OEBPS/chapter2.xhtml":   ch2,
	})

	chapters, err := p.Parse(ctx, data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(chapters) < 2 {
		t.Fatalf("Expected at least 2 chapters, got %d", len(chapters))
	}

	if !strings.Contains(chapters[0].Title, "One") {
		t.Errorf("First chapter title should contain 'One', got %q", chapters[0].Title)
	}

	if len(chapters[0].Paragraphs) < 1 {
		t.Errorf("First chapter should have paragraphs")
	}

	foundText := false
	for _, p := range chapters[0].Paragraphs {
		if strings.Contains(p, "First paragraph of chapter one") {
			foundText = true
		}
	}
	if !foundText {
		t.Errorf("Expected to find text content in chapter 1 paragraphs: %v", chapters[0].Paragraphs)
	}

	if chapters[0].ID == "" {
		t.Error("Chapter ID should not be empty")
	}
	if len(chapters[0].TOCPath) == 0 {
		t.Error("Chapter TOCPath should not be empty")
	}
}

func TestEPUBParser_Parse_SpineOrder(t *testing.T) {
	p := NewEPUBParser()
	ctx := context.Background()

	containerXML := `<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`

	opf := `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="uid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Test Book</dc:title>
  </metadata>
  <manifest>
    <item id="ch_alpha" href="alpha.xhtml" media-type="application/xhtml+xml"/>
    <item id="ch_beta" href="beta.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="ch_beta"/>
    <itemref idref="ch_alpha"/>
  </spine>
</package>`

	alpha := `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Alpha</title></head>
<body><p>Alpha content.</p></body>
</html>`

	beta := `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Beta</title></head>
<body><p>Beta content.</p></body>
</html>`

	data := createEpubZip(map[string]string{
		"META-INF/container.xml": containerXML,
		"OEBPS/content.opf":      opf,
		"OEBPS/alpha.xhtml":      alpha,
		"OEBPS/beta.xhtml":       beta,
	})

	chapters, err := p.Parse(ctx, data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(chapters) < 2 {
		t.Fatalf("Expected at least 2 chapters, got %d", len(chapters))
	}

	if !strings.Contains(chapters[0].Title, "Beta") {
		t.Errorf("Spine order: first chapter should be Beta, got %q", chapters[0].Title)
	}
	if !strings.Contains(chapters[1].Title, "Alpha") {
		t.Errorf("Spine order: second chapter should be Alpha, got %q", chapters[1].Title)
	}
}

func TestEPUBParser_Parse_FallbackWithoutSpine(t *testing.T) {
	p := NewEPUBParser()
	ctx := context.Background()

	containerXML := `<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`

	opf := `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="uid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Test Book</dc:title>
  </metadata>
  <manifest>
    <item id="ch1" href="b.xhtml" media-type="application/xhtml+xml"/>
    <item id="ch2" href="a.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
</package>`

	aXhtml := `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>A</title></head>
<body><p>A content.</p></body>
</html>`

	bXhtml := `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>B</title></head>
<body><p>B content.</p></body>
</html>`

	data := createEpubZip(map[string]string{
		"META-INF/container.xml": containerXML,
		"OEBPS/content.opf":      opf,
		"OEBPS/a.xhtml":          aXhtml,
		"OEBPS/b.xhtml":          bXhtml,
	})

	chapters, err := p.Parse(ctx, data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(chapters) < 2 {
		t.Fatalf("Expected at least 2 chapters from fallback, got %d", len(chapters))
	}
}

func TestEPUBParser_Parse_FallbackWithoutContainerXML(t *testing.T) {
	p := NewEPUBParser()
	ctx := context.Background()

	opf := `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="uid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Test Book</dc:title>
  </metadata>
  <manifest>
    <item id="ch1" href="chapter.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="ch1"/>
  </spine>
</package>`

	chXhtml := `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Chapter</title></head>
<body><p>Content here.</p></body>
</html>`

	data := createEpubZip(map[string]string{
		"OEBPS/content.opf":   opf,
		"OEBPS/chapter.xhtml": chXhtml,
	})

	chapters, err := p.Parse(ctx, data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(chapters) < 1 {
		t.Fatalf("Expected at least 1 chapter, got %d", len(chapters))
	}
}

func TestEPUBParser_Parse_StripScriptAndStyle(t *testing.T) {
	p := NewEPUBParser()
	ctx := context.Background()

	containerXML := `<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`

	opf := `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="uid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Test Book</dc:title>
  </metadata>
  <manifest>
    <item id="ch1" href="ch.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="ch1"/>
  </spine>
</package>`

	chXhtml := `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head>
  <title>Test</title>
  <style>body { color: red; }</style>
</head>
<body>
  <script>alert('hello');</script>
  <p>Visible text here.</p>
  <script>var x = 1;</script>
  <p>More visible text.</p>
</body>
</html>`

	data := createEpubZip(map[string]string{
		"META-INF/container.xml": containerXML,
		"OEBPS/content.opf":      opf,
		"OEBPS/ch.xhtml":         chXhtml,
	})

	chapters, err := p.Parse(ctx, data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(chapters) == 0 {
		t.Fatal("Expected at least 1 chapter")
	}

	for _, p := range chapters[0].Paragraphs {
		if strings.Contains(p, "alert") || strings.Contains(p, "color") {
			t.Errorf("Paragraph should not contain script/style content: %q", p)
		}
		if strings.Contains(p, "var x") {
			t.Errorf("Paragraph should not contain script content: %q", p)
		}
	}

	allText := strings.Join(chapters[0].Paragraphs, " ")
	if !strings.Contains(allText, "Visible text here") {
		t.Errorf("Expected 'Visible text here' in paragraphs: %v", chapters[0].Paragraphs)
	}
}

func TestEPUBParser_Parse_EntityDecoding(t *testing.T) {
	p := NewEPUBParser()
	ctx := context.Background()

	containerXML := `<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`

	opf := `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="uid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Test Book</dc:title>
  </metadata>
  <manifest>
    <item id="ch1" href="ch.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="ch1"/>
  </spine>
</package>`

	chXhtml := `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Entities</title></head>
<body>
  <p>He said &amp; she agreed.</p>
  <p>&quot;Hello&amp;quot; he said.</p>
  <p>A &lt;B&gt; is true.</p>
</body>
</html>`

	data := createEpubZip(map[string]string{
		"META-INF/container.xml": containerXML,
		"OEBPS/content.opf":      opf,
		"OEBPS/ch.xhtml":         chXhtml,
	})

	chapters, err := p.Parse(ctx, data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(chapters) == 0 {
		t.Fatal("Expected at least 1 chapter")
	}

	allText := strings.Join(chapters[0].Paragraphs, " ")
	if !strings.Contains(allText, "&") {
		t.Errorf("Expected decoded &amp; entity in: %s", allText)
	}
	if !strings.Contains(allText, `"`) {
		t.Errorf("Expected decoded &quot; entity in: %s", allText)
	}
	if !strings.Contains(allText, "<") {
		t.Errorf("Expected decoded &lt; entity in: %s", allText)
	}
}

func TestEPUBParser_Parse_InvalidInput(t *testing.T) {
	p := NewEPUBParser()
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

	t.Run("invalid zip", func(t *testing.T) {
		_, err := p.Parse(ctx, []byte("this is not a zip"))
		if err == nil {
			t.Error("Expected error for invalid zip data")
		}
	})

	t.Run("valid zip no xhtml", func(t *testing.T) {
		data := createEpubZip(map[string]string{
			"readme.txt": "This is not an EPUB.",
		})
		_, err := p.Parse(ctx, data)
		if err == nil {
			t.Error("Expected error for zip with no XHTML content")
		}
	})
}

func TestEPUBParser_Parse_StableChapterIDs(t *testing.T) {
	p := NewEPUBParser()
	ctx := context.Background()

	containerXML := `<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`

	opf := `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="uid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Test Book</dc:title>
  </metadata>
  <manifest>
    <item id="chap1" href="chap1.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="chap1"/>
  </spine>
</package>`

	ch1 := `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Chap 1</title></head>
<body><p>Content.</p></body>
</html>`

	data := createEpubZip(map[string]string{
		"META-INF/container.xml": containerXML,
		"OEBPS/content.opf":      opf,
		"OEBPS/chap1.xhtml":      ch1,
	})

	chapters, err := p.Parse(ctx, data)
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

func TestEPUBParser_Parse_HTMLFileFallback(t *testing.T) {
	p := NewEPUBParser()
	ctx := context.Background()

	htmlContent := `<html>
<head><title>Legacy Chapter</title></head>
<body>
  <p>This is from an HTML file.</p>
</body>
</html>`

	data := createEpubZip(map[string]string{
		"chapter.html": htmlContent,
	})

	chapters, err := p.Parse(ctx, data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(chapters) < 1 {
		t.Fatalf("Expected at least 1 chapter from HTML fallback, got %d", len(chapters))
	}

	allText := strings.Join(chapters[0].Paragraphs, " ")
	if !strings.Contains(allText, "This is from an HTML file") {
		t.Errorf("Expected HTML content in paragraphs: %v", chapters[0].Paragraphs)
	}
}

func TestEPUBParser_Parse_OPFHrefWithSubdirectory(t *testing.T) {
	p := NewEPUBParser()
	ctx := context.Background()

	containerXML := `<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`

	opf := `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="uid">
  <manifest>
    <item id="ch1" href="text/chapter1.xhtml" media-type="application/xhtml+xml; charset=utf-8"/>
  </manifest>
  <spine>
    <itemref idref="ch1"/>
  </spine>
</package>`

	chapter := `<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Nested Chapter</title></head>
<body><p>Nested path content.</p></body>
</html>`

	data := createEpubZip(map[string]string{
		"META-INF/container.xml":    containerXML,
		"OEBPS/content.opf":         opf,
		"OEBPS/text/chapter1.xhtml": chapter,
	})

	chapters, err := p.Parse(ctx, data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(chapters) != 1 {
		t.Fatalf("Expected 1 chapter, got %d", len(chapters))
	}
	if !strings.Contains(strings.Join(chapters[0].Paragraphs, " "), "Nested path content") {
		t.Errorf("Expected nested href content, got: %v", chapters[0].Paragraphs)
	}
}

func makeMinimalEpub(chapterXHTML string) []byte {
	containerXML := `<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`

	opf := `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="uid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Test Book</dc:title>
  </metadata>
  <manifest>
    <item id="ch1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="ch1"/>
  </spine>
</package>`

	return createEpubZip(map[string]string{
		"META-INF/container.xml": containerXML,
		"OEBPS/content.opf":      opf,
		"OEBPS/chapter1.xhtml": fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Chapter 1</title></head>
<body>%s</body>
</html>`, chapterXHTML),
	})
}
