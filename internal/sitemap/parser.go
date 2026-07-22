package sitemap

import (
	"bytes"
	"encoding/xml"
	"fmt"
)

func Parse(data []byte) (SitemapType, *URLSet, *SitemapIndex, error) {
	sitemapType, err := detectType(data)
	if err != nil {
		return SitemapTypeUnknown, nil, nil, fmt.Errorf("detecting sitemap type: %w", err)
	}
	switch sitemapType {
	case SitemapTypeURLSet:
		var urlSet URLSet
		if err := xml.Unmarshal(data, &urlSet); err != nil {
			return SitemapTypeUnknown, nil, nil, fmt.Errorf("parsing urlset xml: %w", err)

		}
		return SitemapTypeURLSet, &urlSet, nil, nil

	case SitemapTypeIndex:
		var index SitemapIndex
		if err := xml.Unmarshal(data, &index); err != nil {
			return SitemapTypeUnknown, nil, nil, fmt.Errorf("parsing siteindex xml: %w", err)

		}
		return SitemapTypeIndex, nil, &index, nil
	default:
		return SitemapTypeUnknown, nil, nil, fmt.Errorf("unrecognized sitemap root element")

	}

}

func detectType(data []byte) (SitemapType, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))

	for {
		token, err := decoder.Token()
		if err != nil {
			return SitemapTypeUnknown, fmt.Errorf("reading xml tokens: %w", err)
		}

		if start, ok := token.(xml.StartElement); ok {
			switch start.Name.Local {
			case "urlset":
				return SitemapTypeURLSet, nil
			case "sitemapindex":
				return SitemapTypeIndex, nil
			default:
				return SitemapTypeUnknown, fmt.Errorf("unexpected root element <%s>", start.Name.Local)
			}
		}
	}
}
func (u *URLSet) ExtractLocs() []string {
	locs := make([]string, 0, len(u.URLs))
	for _, entry := range u.URLs {
		if entry.Loc != "" {
			locs = append(locs, entry.Loc)
		}
	}
	return locs
}

func (s *SitemapIndex) ExtractLocs() []string {
	locs := make([]string, 0, len(s.Sitemaps))
	for _, entry := range s.Sitemaps {
		if entry.Loc != "" {
			locs = append(locs, entry.Loc)
		}
	}
	return locs
}
