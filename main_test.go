package main

import (
	"bytes"
	"testing"
)

func TestTemplates(t *testing.T) {
	tests := []struct {
		name            string
		input           commentData
		expectedComment string
		expectedCommit  string
	}{
		{
			"NoChangesOrNew",
			commentData{},
			`## Article Sync Summary

After merge, 0 new article will be created and 0 existing article will be updated.`,
			`completed sync: 0 new, 0 updated
`,
		},
		{
			"OneNewArticle",
			commentData{
				NewArticles: []*Article{{
					Title: "My New Article",
					URL:   "dev.to",
				}},
			},
			`## Article Sync Summary

After merge, 1 new article will be created and 0 existing article will be updated.

### New Articles
- My New Article`,
			`completed sync: 1 new, 0 updated

- new: My New Article (dev.to)`,
		},
		{
			"TwoNewArticles",
			commentData{
				NewArticles: []*Article{
					{
						Title: "My New Article",
						URL:   "dev.to",
					},
					{
						Title: "My Other New Article",
						URL:   "dev.to",
					}},
			},
			`## Article Sync Summary

After merge, 2 new article will be created and 0 existing article will be updated.

### New Articles
- My New Article
- My Other New Article`,
			`completed sync: 2 new, 0 updated

- new: My New Article (dev.to)
- new: My Other New Article (dev.to)`,
		},
		{
			"OneUpdatedArticle",
			commentData{
				UpdatedArticles: []*Article{{
					Title: "My Updated Article",
					URL:   "dev.to",
				}},
			},
			`## Article Sync Summary

After merge, 0 new article will be created and 1 existing article will be updated.

### Updated Articles
- [My Updated Article](dev.to)`,
			`completed sync: 0 new, 1 updated

- updated: My Updated Article (dev.to)`,
		},
		{
			"TwoUpdatedArticles",
			commentData{
				UpdatedArticles: []*Article{
					{
						Title: "My Updated Article",
						URL:   "dev.to",
					},
					{
						Title: "My Other Updated Article",
						URL:   "dev.to",
					},
				},
			},
			`## Article Sync Summary

After merge, 0 new article will be created and 2 existing article will be updated.

### Updated Articles
- [My Updated Article](dev.to)
- [My Other Updated Article](dev.to)`,
			`completed sync: 0 new, 2 updated

- updated: My Updated Article (dev.to)
- updated: My Other Updated Article (dev.to)`,
		},
		{
			"UpdatedAndNewArticles",
			commentData{
				NewArticles: []*Article{{
					Title: "My New Article",
					URL:   "dev.to",
				}},
				UpdatedArticles: []*Article{{
					Title: "My Updated Article",
					URL:   "dev.to",
				}},
			},
			`## Article Sync Summary

After merge, 1 new article will be created and 1 existing article will be updated.

### New Articles
- My New Article

### Updated Articles
- [My Updated Article](dev.to)`,
			`completed sync: 1 new, 1 updated

- new: My New Article (dev.to)
- updated: My Updated Article (dev.to)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+"Comment", func(t *testing.T) {
			var result bytes.Buffer
			err := renderTemplate(commentTemplate, tt.input, &result)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.String() != tt.expectedComment {
				t.Fatalf("unexpected result: %s", result.String())
			}
		})
		t.Run(tt.name+"Commit", func(t *testing.T) {
			var result bytes.Buffer
			err := renderTemplate(commitTemplate, tt.input, &result)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.String() != tt.expectedCommit {
				t.Fatalf("unexpected result: %s", result.String())
			}
		})
	}
}
