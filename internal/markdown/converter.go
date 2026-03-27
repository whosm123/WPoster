package markdown

import (
	"bytes"
	"io"
	"os"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

// ConvertMarkdownToHTML converts markdown content to HTML
func ConvertMarkdownToHTML(markdownContent []byte) []byte {
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	p := parser.NewWithExtensions(extensions)

	htmlFlags := html.CommonFlags
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	return markdown.ToHTML(markdownContent, p, renderer)
}

// ConvertMarkdownFile reads a markdown file and converts it to HTML
func ConvertMarkdownFile(filePath string) ([]byte, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return ConvertMarkdownToHTML(content), nil
}

// ConvertMarkdownReader reads markdown from an io.Reader and converts it to HTML
func ConvertMarkdownReader(reader io.Reader) ([]byte, error) {
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		return nil, err
	}

	return ConvertMarkdownToHTML(buf.Bytes()), nil
}
