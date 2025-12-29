package parser

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"vitoru.fun/torrents/api/v1alpha1"
)

// ParseResult represents a single torrent found in the indexer results
type ParseResult struct {
	Title       string
	Magnet      string
	Size        string
	Seeders     int
	Leechers    int
	PublishedAt *time.Time
	Indexer     string
}

// ParseHTML parses the HTML content using the indexer definition selector
func ParseHTML(htmlContent string, indexer *v1alpha1.Indexer) ([]ParseResult, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to load HTML: %v", err)
	}

	spec := indexer.Spec
	var results []ParseResult

	if spec.Search == nil || spec.Search.Rows.Selector == "" {
		return nil, fmt.Errorf("indexer %s has no search selectors", indexer.Name)
	}

	// iterate over rows
	fmt.Printf("Parser: URL=%s Selector=%s\n", indexer.Spec.Links[0], spec.Search.Rows.Selector)
	doc.Find(spec.Search.Rows.Selector).Each(func(i int, s *goquery.Selection) {
		fmt.Printf("Parser: Found Row %d\n", i)
		result := ParseResult{
			Indexer: indexer.Name,
		}

		// Parse Title
		if sel, ok := spec.Search.Fields["title"]; ok && sel.Selector != "" {
			titleSel := s.Find(sel.Selector)
			result.Title = strings.TrimSpace(titleSel.Text())
		}

		// Parse Magnet/Download
		if sel, ok := spec.Search.Fields["download"]; ok && sel.Selector != "" {
			dlSel := s.Find(sel.Selector)
			attr := "href"
			if sel.Attribute != "" {
				attr = sel.Attribute
			}
			val, exists := dlSel.Attr(attr)
			if exists {
				// Resolve relative URL
				if !strings.HasPrefix(val, "http") && !strings.HasPrefix(val, "magnet:") {
					// Use indexer link as base.
					// We need base URL here. We have indexer.Spec.Links[0]
					baseURL := strings.TrimRight(indexer.Spec.Links[0], "/")
					if !strings.HasPrefix(val, "/") {
						val = baseURL + "/" + val
					} else {
						val = baseURL + val
					}
				}
				result.Magnet = val
			}
		}

		// Parse Size
		if sel, ok := spec.Search.Fields["size"]; ok && sel.Selector != "" {
			sizeSel := s.Find(sel.Selector)
			result.Size = strings.TrimSpace(sizeSel.Text())
		}

		// Parse Seeders
		if sel, ok := spec.Search.Fields["seeders"]; ok && sel.Selector != "" {
			seedSel := s.Find(sel.Selector)
			txt := strings.TrimSpace(seedSel.Text())
			val, _ := strconv.Atoi(txt)
			result.Seeders = val
		}

		// Parse Leechers
		if sel, ok := spec.Search.Fields["leechers"]; ok && sel.Selector != "" {
			leechSel := s.Find(sel.Selector)
			txt := strings.TrimSpace(leechSel.Text())
			val, _ := strconv.Atoi(txt)
			result.Leechers = val
		}

		// Only add valid results (must have title and magnet)
		fmt.Printf("Parser Row %d: Title='%s' Magnet='%s' Size='%s' Seeders=%d\n", i, result.Title, result.Magnet, result.Size, result.Seeders)
		if result.Title != "" && result.Magnet != "" {
			results = append(results, result)
		}
	})

	return results, nil
}

// ParseDetailsPage parses the details page HTML to find the magnet link
func ParseDetailsPage(htmlContent string, downloadBlock *v1alpha1.DownloadBlock) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return "", fmt.Errorf("failed to load HTML: %v", err)
	}

	for _, selBlock := range downloadBlock.Selectors {
		sel := selBlock.Selector
		attr := "href"
		if selBlock.Attribute != "" {
			attr = selBlock.Attribute
		}

		// Try selector
		s := doc.Find(sel)
		if s.Length() > 0 {
			val, exists := s.Attr(attr)
			if exists && val != "" {
				return val, nil
			}
		}
	}

	return "", fmt.Errorf("magnet link not found in details page")
}
