package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/calvinmclean/article-sync/api"
)

func main() {
	var apiKey, path string
	var dryRun bool
	flag.StringVar(&apiKey, "api-key", "", "API key for accessing dev.to")
	flag.StringVar(&path, "path", "./articles", "root path to scan for articles")
	flag.BoolVar(&dryRun, "dry-run", false, "dry-run to print which changes will be made without doing them")
	flag.Parse()

	if apiKey == "" {
		apiKey = os.Getenv("API_KEY")
		if apiKey == "" {
			log.Fatalf("missing required argument --api-key or env var API_KEY")
		}
	}

	client, err := newClient(apiKey, dryRun)
	if err != nil {
		log.Fatalf("error creating API client: %v", err)
	}

	err = client.syncArticlesFromRootDirectory(path)
	if err != nil {
		log.Fatalf("error synchronizing directory: %v", err)
	}
}

type client struct {
	*api.ClientWithResponses
	dryRun bool
	logger *slog.Logger
}

func newClient(apikey string, dryRun bool) (*client, error) {
	c, err := api.NewClientWithResponses("https://dev.to", api.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
		req.Header.Add("api-key", apikey)
		return nil
	}))
	if err != nil {
		return nil, fmt.Errorf("error creating client: %w", err)
	}

	return &client{c, dryRun, slog.New(slog.NewTextHandler(os.Stdout, nil))}, nil
}

// Article is used to show which fields can read/write to local file
type Article struct {
	ID          int    `json:"id"`
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

func (c *client) syncArticlesFromRootDirectory(rootDir string) error {
	return filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("error accessing path %s: %w", path, err)
		}

		if path == rootDir {
			return nil
		}

		if !d.IsDir() {
			return nil
		}

		c.logger.Info("sychronizing article", "directory", path)
		err = c.syncArticleFromDirectory(path)
		if err != nil {
			return fmt.Errorf("error synchronizing article from path %s: %w", path, err)
		}
		return nil
	})
}

// syncArticleFromDirectory will read the article files from a directory and:
//   - If no ID is provided, create a new article and record ID
//   - Otherwise, get article by ID and compare text to local text. If the file is
//     recently changed, it will be updated by API
func (c *client) syncArticleFromDirectory(dir string) error {
	markdownBody, err := os.ReadFile(filepath.Join(dir, "article.md"))
	if err != nil {
		return fmt.Errorf("error reading markdown: %w", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "article.json"))
	if err != nil {
		return fmt.Errorf("error reading JSON file: %w", err)
	}

	var article Article
	err = json.Unmarshal(data, &article)
	if err != nil {
		return fmt.Errorf("error parsing article details: %w", err)
	}

	logger := c.logger.With("directory", dir).With("title", article.Title)

	var respBody []byte
	var created bool
	switch article.ID {
	case 0:
		created = true
		logger.Info("creating new article")
		if c.dryRun {
			return nil
		}
		respBody, err = c.createArticle(article, string(markdownBody))
		if err != nil {
			return fmt.Errorf("error creating article: %w", err)
		}
	default:
		logger = logger.With("id", article.ID)

		shouldUpdate, err := c.shouldUpdateArticle(string(markdownBody), article.ID)
		if err != nil {
			return fmt.Errorf("error checking if article needs update: %w", err)
		}
		if !shouldUpdate {
			logger.Info("article is up-to-date")
			return nil
		}
		logger.Info("updating article with new body")

		if c.dryRun {
			return nil
		}

		respBody, err = c.updateArticle(dir, article, string(markdownBody))
		if err != nil {
			return fmt.Errorf("error updating article: %w", err)
		}
	}

	err = json.Unmarshal(respBody, &article)
	if err != nil {
		return fmt.Errorf("error unmarshaling response JSON: %w", err)
	}

	// article was created so logger doesn't already have ID
	if created {
		logger = logger.With("id", article.ID)
	}

	logger.Info("successfully synchronized article")

	newData, err := json.MarshalIndent(article, "", "    ")
	if err != nil {
		return fmt.Errorf("error marshaling response JSON to write to file: %w", err)
	}

	err = os.WriteFile(filepath.Join(dir, "article.json"), newData, 0640)
	if err != nil {
		return fmt.Errorf("error writing JSON file: %w", err)
	}

	return nil
}

func (c *client) shouldUpdateArticle(markdownBody string, id int) (bool, error) {
	article, err := c.getArticle(id)
	if err != nil {
		return false, fmt.Errorf("error getting article: %w", err)
	}

	articleMarkdown, ok := article["body_markdown"].(string)
	if !ok {
		return false, fmt.Errorf("error checking body_markdown")
	}

	return articleMarkdown != markdownBody, nil
}

func (c *client) updateArticle(dir string, article Article, markdownBody string) ([]byte, error) {
	published := true
	articleBody := api.Article{}
	articleBody.Article = &struct {
		BodyMarkdown   *string "json:\"body_markdown,omitempty\""
		CanonicalUrl   *string "json:\"canonical_url\""
		Description    *string "json:\"description,omitempty\""
		MainImage      *string "json:\"main_image\""
		OrganizationId *int    "json:\"organization_id\""
		Published      *bool   "json:\"published,omitempty\""
		Series         *string "json:\"series\""
		Tags           *string "json:\"tags,omitempty\""
		Title          *string "json:\"title,omitempty\""
	}{
		Title:        &article.Title,
		Description:  &article.Description,
		BodyMarkdown: &markdownBody,
		Published:    &published,
	}

	resp, err := c.UpdateArticleWithResponse(context.Background(), int32(article.ID), articleBody)
	if err != nil {
		return nil, fmt.Errorf("error updating article: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("unexpected status updating article %d: %d %s", article.ID, resp.StatusCode(), string(resp.Body))
	}

	return resp.Body, nil
}

func (c *client) getArticle(id int) (map[string]interface{}, error) {
	resp, err := c.GetArticleByIdWithResponse(context.Background(), id)
	if err != nil {
		return nil, fmt.Errorf("error getting article %d: %w", id, err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("unexpected status getting article %d: %d %s", id, resp.StatusCode(), string(resp.Body))
	}

	return *resp.JSON200, nil
}

func (c *client) createArticle(article Article, body string) ([]byte, error) {
	published := true
	articleBody := api.Article{}
	articleBody.Article = &struct {
		BodyMarkdown   *string "json:\"body_markdown,omitempty\""
		CanonicalUrl   *string "json:\"canonical_url\""
		Description    *string "json:\"description,omitempty\""
		MainImage      *string "json:\"main_image\""
		OrganizationId *int    "json:\"organization_id\""
		Published      *bool   "json:\"published,omitempty\""
		Series         *string "json:\"series\""
		Tags           *string "json:\"tags,omitempty\""
		Title          *string "json:\"title,omitempty\""
	}{
		Title:        &article.Title,
		Description:  &article.Description,
		BodyMarkdown: &body,
		Published:    &published,
	}

	resp, err := c.CreateArticleWithResponse(context.Background(), articleBody)
	if err != nil {
		return nil, fmt.Errorf("error creating article: %w", err)
	}

	if resp.StatusCode() != http.StatusCreated {
		return nil, fmt.Errorf("unexpected status creating article: %d %s", resp.StatusCode(), string(resp.Body))
	}

	return resp.Body, nil
}
