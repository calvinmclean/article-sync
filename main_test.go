package main

import (
	"bytes"
	"testing"
)

func TestCommentTemplates(t *testing.T) {
	tests := []struct {
		name     string
		input    commentData
		expected string
	}{
		{
			"NoChangesOrNew",
			commentData{},
			`## Article Sync Summary

After merge, 0 new article will be created and 0 existing article will be updated.`,
		},
		{
			"OneNewArticle",
			commentData{
				NewArticles: []*Article{{
					Title: "My New Article",
				}},
			},
			`## Article Sync Summary

After merge, 1 new article will be created and 0 existing article will be updated.

### New Articles
- My New Article`,
		},
		{
			"TwoNewArticles",
			commentData{
				NewArticles: []*Article{
					{
						Title: "My New Article",
					},
					{
						Title: "My Other New Article",
					}},
			},
			`## Article Sync Summary

After merge, 2 new article will be created and 0 existing article will be updated.

### New Articles
- My New Article
- My Other New Article`,
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
		},
		{
			"UpdatedAndNewArticles",
			commentData{
				NewArticles: []*Article{{
					Title: "My New Article",
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
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result bytes.Buffer
			err := renderCommentTemplate(tt.input, &result)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.String() != tt.expected {
				t.Fatalf("unexpected result: %s", result.String())
			}
		})
	}
}
