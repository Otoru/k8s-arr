package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"vitoru.fun/torrents/api/v1alpha1"
)

func TestParseHTML(t *testing.T) {
	// Mock Indexer
	indexer := &v1alpha1.Indexer{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-indexer",
		},
		Spec: v1alpha1.IndexerSpec{
			Links: []string{"https://example.com"},
			Search: &v1alpha1.Search{
				Rows: v1alpha1.RowsBlock{
					Selector: "tr.result",
				},
				Fields: v1alpha1.FieldsBlock{
					"title":    v1alpha1.SelectorBlock{Selector: ".title"},
					"download": v1alpha1.SelectorBlock{Selector: ".dl", Attribute: "href"},
					"size":     v1alpha1.SelectorBlock{Selector: ".size"},
					"seeders":  v1alpha1.SelectorBlock{Selector: ".seeds"},
					"leechers": v1alpha1.SelectorBlock{Selector: ".leechs"},
				},
			},
		},
	}

	tests := []struct {
		name     string
		html     string
		expected []ParseResult
	}{
		{
			name: "Valid Search Result",
			html: `
				<html>
					<body>
						<table>
							<tr class="result">
								<td class="title">Ubuntu ISO</td>
								<td><a class="dl" href="magnet:?xt=urn:btih:123">Download</a></td>
								<td class="size">2.5 GB</td>
								<td class="seeds">100</td>
								<td class="leechs">10</td>
							</tr>
						</table>
					</body>
				</html>
			`,
			expected: []ParseResult{
				{
					Title:    "Ubuntu ISO",
					Magnet:   "magnet:?xt=urn:btih:123",
					Size:     "2.5 GB",
					Seeders:  100,
					Leechers: 10,
				},
			},
		},
		{
			name: "Relative URL Resolution",
			html: `
				<html>
					<body>
						<table>
							<tr class="result">
								<td class="title">Debian ISO</td>
								<td><a class="dl" href="/download/debian.torrent">Download</a></td>
								<td class="size">600 MB</td>
								<td class="seeds">50</td>
								<td class="leechs">5</td>
							</tr>
						</table>
					</body>
				</html>
			`,
			expected: []ParseResult{
				{
					Title:    "Debian ISO",
					Magnet:   "https://example.com/download/debian.torrent", // Expect resolved URL
					Size:     "600 MB",
					Seeders:  50,
					Leechers: 5,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := ParseHTML(tt.html, indexer)
			assert.NoError(t, err)
			assert.Equal(t, len(tt.expected), len(results))
			if len(results) > 0 {
				assert.Equal(t, tt.expected[0].Title, results[0].Title)
				assert.Equal(t, tt.expected[0].Magnet, results[0].Magnet)
				assert.Equal(t, tt.expected[0].Seeders, results[0].Seeders)
			}
		})
	}
}
