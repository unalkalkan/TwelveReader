package parser

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
	"regexp"
	"sort"
	"strings"

	"github.com/unalkalkan/TwelveReader/pkg/types"
)

type EPUBParser struct{}

const (
	epubMaxFileSize  = 10 << 20
	epubMaxTotalSize = 50 << 20
)

var (
	epubTitleRe      = regexp.MustCompile(`(?is)<title>(.*?)</title>`)
	epubHeadingRe    = regexp.MustCompile(`(?is)<h[1-6][^>]*>(.*?)</h[1-6]>`)
	epubBodyRe       = regexp.MustCompile(`(?is)<body[^>]*>(.*?)</body>`)
	epubScriptRe     = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	epubStyleRe      = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	epubBreakRe      = regexp.MustCompile(`(?is)<br\s*/?>`)
	epubBlockCloseRe = regexp.MustCompile(`(?is)</(p|div|li|blockquote|dt|dd|td|tr|br|h[1-6])>`)
	epubTagRe        = regexp.MustCompile(`<[^>]+>`)
	errEPUBSizeLimit = errors.New("epub: extracted content exceeds safety limit")
)

func NewEPUBParser() *EPUBParser {
	return &EPUBParser{}
}

func (p *EPUBParser) Parse(ctx context.Context, data []byte) ([]*types.Chapter, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("epub: empty data")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("epub: invalid zip: %w", err)
	}

	fileMap := make(map[string]string, len(r.File))
	totalSize := int64(0)
	for _, f := range r.File {
		if f.UncompressedSize64 > epubMaxFileSize {
			return nil, fmt.Errorf("%w: %s", errEPUBSizeLimit, f.Name)
		}
		totalSize += int64(f.UncompressedSize64)
		if totalSize > epubMaxTotalSize {
			return nil, errEPUBSizeLimit
		}
		rc, err := f.Open()
		if err != nil {
			continue
		}
		content, err := io.ReadAll(io.LimitReader(rc, epubMaxFileSize+1))
		rc.Close()
		if err != nil {
			continue
		}
		if len(content) > epubMaxFileSize {
			return nil, fmt.Errorf("%w: %s", errEPUBSizeLimit, f.Name)
		}
		fileMap[f.Name] = string(content)
	}

	opfPath := findOPFPath(fileMap)

	opfDir := ""
	var spine []string
	manifest := make(map[string]string)

	if opfPath != "" {
		opfContent, ok := fileMap[opfPath]
		if ok {
			if idx := strings.LastIndex(opfPath, "/"); idx >= 0 {
				opfDir = opfPath[:idx+1]
			}
			spine, manifest = parseOPF(opfContent)
		}
	}

	type htmlEntry struct {
		href  string
		path  string
		title string
	}

	var chapters []*types.Chapter

	if len(spine) > 0 {
		for i, idref := range spine {
			href, ok := manifest[idref]
			if !ok {
				continue
			}
			fullPath := opfDir + href
			htmlContent, ok := fileMap[fullPath]
			if !ok {
				continue
			}
			title := extractTitle(htmlContent)
			if title == "" {
				title = fmt.Sprintf("Chapter %d", i+1)
			}
			paragraphs := extractParagraphs(htmlContent)
			if len(paragraphs) == 0 {
				continue
			}
			chapters = append(chapters, &types.Chapter{
				ID:         fmt.Sprintf("chapter_%03d", i+1),
				Number:     i + 1,
				Title:      title,
				TOCPath:    []string{title},
				Paragraphs: paragraphs,
			})
		}
	}

	if len(chapters) == 0 {
		var xhtmlFiles []string
		for path := range fileMap {
			lower := strings.ToLower(path)
			if strings.HasSuffix(lower, ".xhtml") || strings.HasSuffix(lower, ".html") ||
				strings.HasSuffix(lower, ".htm") {
				if strings.Contains(strings.ToLower(path), "toc") ||
					strings.Contains(strings.ToLower(path), "nav") {
					continue
				}
				xhtmlFiles = append(xhtmlFiles, path)
			}
		}
		sort.Strings(xhtmlFiles)

		for i, path := range xhtmlFiles {
			htmlContent := fileMap[path]
			title := extractTitle(htmlContent)
			if title == "" {
				title = fmt.Sprintf("Chapter %d", i+1)
			}
			paragraphs := extractParagraphs(htmlContent)
			if len(paragraphs) == 0 {
				continue
			}
			chapters = append(chapters, &types.Chapter{
				ID:         fmt.Sprintf("chapter_%03d", i+1),
				Number:     i + 1,
				Title:      title,
				TOCPath:    []string{title},
				Paragraphs: paragraphs,
			})
		}
	}

	if len(chapters) == 0 {
		return nil, fmt.Errorf("epub: no readable chapters found")
	}

	return chapters, nil
}

func findOPFPath(fileMap map[string]string) string {
	if containerXML, ok := fileMap["META-INF/container.xml"]; ok {
		var container struct {
			XMLName   xml.Name `xml:"container"`
			RootFiles []struct {
				FullPath string `xml:"full-path,attr"`
			} `xml:"rootfiles>rootfile"`
		}
		if err := xml.Unmarshal([]byte(containerXML), &container); err == nil {
			for _, rf := range container.RootFiles {
				if rf.FullPath != "" {
					return rf.FullPath
				}
			}
		}
	}

	for path := range fileMap {
		if strings.HasSuffix(strings.ToLower(path), ".opf") {
			return path
		}
	}
	return ""
}

func parseOPF(content string) (spine []string, manifest map[string]string) {
	manifest = make(map[string]string)

	var pkg struct {
		Manifest struct {
			Items []struct {
				ID        string `xml:"id,attr"`
				Href      string `xml:"href,attr"`
				MediaType string `xml:"media-type,attr"`
			} `xml:"item"`
		} `xml:"manifest"`
		Spine struct {
			ItemRefs []struct {
				IDRef string `xml:"idref,attr"`
			} `xml:"itemref"`
		} `xml:"spine"`
	}

	if err := xml.Unmarshal([]byte(content), &pkg); err != nil {
		return spine, manifest
	}

	for _, item := range pkg.Manifest.Items {
		mediaType := strings.ToLower(strings.TrimSpace(strings.Split(item.MediaType, ";")[0]))
		if item.MediaType == "" ||
			mediaType == "application/xhtml+xml" ||
			mediaType == "text/html" ||
			mediaType == "application/xml" {
			manifest[item.ID] = item.Href
		}
	}

	for _, ref := range pkg.Spine.ItemRefs {
		spine = append(spine, ref.IDRef)
	}

	return spine, manifest
}

func extractTitle(htmlContent string) string {
	matches := epubTitleRe.FindStringSubmatch(htmlContent)
	if len(matches) > 1 {
		title := strings.TrimSpace(html.UnescapeString(stripTags(matches[1])))
		if title != "" {
			return title
		}
	}

	matches2 := epubHeadingRe.FindStringSubmatch(htmlContent)
	if len(matches2) > 1 {
		heading := strings.TrimSpace(html.UnescapeString(stripTags(matches2[1])))
		if heading != "" {
			return heading
		}
	}

	return ""
}

func extractParagraphs(htmlContent string) []string {
	bodyMatch := epubBodyRe.FindStringSubmatch(htmlContent)
	body := htmlContent
	if len(bodyMatch) > 1 {
		body = bodyMatch[1]
	}

	body = removeScriptStyle(body)
	body = addBlockBreaks(body)
	text := stripTags(body)
	text = html.UnescapeString(text)
	return splitParagraphs(text)
}

func removeScriptStyle(htmlContent string) string {
	htmlContent = epubScriptRe.ReplaceAllString(htmlContent, "")
	htmlContent = epubStyleRe.ReplaceAllString(htmlContent, "")
	return htmlContent
}

func addBlockBreaks(htmlContent string) string {
	htmlContent = epubBreakRe.ReplaceAllString(htmlContent, "\n")
	htmlContent = epubBlockCloseRe.ReplaceAllString(htmlContent, "\n\n")
	return htmlContent
}

func stripTags(s string) string {
	return epubTagRe.ReplaceAllString(s, "")
}

func splitParagraphs(text string) []string {
	lines := strings.Split(text, "\n")
	var paragraphs []string
	var buf strings.Builder
	for _, line := range lines {
		line = strings.Join(strings.Fields(line), " ")
		if line == "" {
			if buf.Len() > 0 {
				paragraphs = append(paragraphs, strings.TrimSpace(buf.String()))
				buf.Reset()
			}
			continue
		}
		if buf.Len() > 0 {
			buf.WriteString(" ")
		}
		buf.WriteString(line)
	}
	if buf.Len() > 0 {
		paragraphs = append(paragraphs, strings.TrimSpace(buf.String()))
	}
	return paragraphs
}

func (p *EPUBParser) SupportedFormats() []string {
	return []string{"epub"}
}
