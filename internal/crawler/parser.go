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
	return strings.Contains(strings.ToLower(contentType), "text/html")

}
