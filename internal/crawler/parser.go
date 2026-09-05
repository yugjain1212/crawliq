package crawler

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type ParsedPage struct {
	Title string
}

func ParseHTML(body []byte) (*ParsedPage, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("Parsing HTML document: %w", err)

	}
	title := strings.TrimSpace(doc.Find("title").First().Text())

	return &ParsedPage{
		Title: title,
	}, nil

}
func IsHTML(contentType string) bool {
	ct := strings.ToLower(contentType)
	// text/html is the standard; application/xhtml+xml is what strict
	// XHTML servers send. The substring match also catches the
	// common "; charset=utf-8" suffix.
	return strings.Contains(ct, "text/html") || strings.Contains(ct, "application/xhtml+xml")
}
