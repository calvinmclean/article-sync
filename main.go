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
	"slices"

	"github.com/calvinmclean/article-sync/api"
	"github.com/fogleman/gg"
)

// Article is used to show which fields can read/write to local file
type Article struct {
	ID          int      `json:"id"`
	Slug        string   `json:"slug"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	URL         string   `json:"url"`
	Tags        []string `json:"tags"`
	CoverImage  string   `json:"cover_image"`

	Gopher string `json:"gopher"`

	new     bool
	updated bool
}

type commentData struct {
	NewArticles     []*Article
	UpdatedArticles []*Article
}

func main() {
	var apiKey, path, prComment, commit, repositoryName, branch string
	var dryRun, createImage, init bool
	flag.StringVar(&apiKey, "api-key", "", "API key for accessing dev.to")
	flag.StringVar(&path, "path", "./articles", "root path to scan for articles")
	flag.StringVar(&prComment, "pr-comment", "", "file to write the PR comment into")
	flag.StringVar(&commit, "commit", "", "file to write the commit message into")
	flag.StringVar(&repositoryName, "repo", "", "repository name. Used for cover image URL")
	flag.StringVar(&branch, "branch", "", "main branch name. Used for cover image URL")
	flag.BoolVar(&dryRun, "dry-run", false, "dry-run to print which changes will be made without doing them")
	flag.BoolVar(&createImage, "create-image", false, "create gopher cover image even if using dry-run")
	flag.BoolVar(&init, "init", false, "download articles from profile and create directories")
	flag.Parse()

	if apiKey == "" {
		apiKey = os.Getenv("API_KEY")
		if apiKey == "" {
			log.Fatalf("missing required argument --api-key or env var API_KEY")
		}
	}

	client, err := newClient(apiKey, dryRun, createImage)
	if err != nil {
		log.Fatalf("error creating API client: %v", err)
	}

	if init {
		err = client.init(path)
		if err != nil {
			log.Fatalf("error initializing: %v", err)
		}
		return
	}

	if branch != "" || repositoryName != "" {
		client.repositoryName = repositoryName
		client.branch = branch
	}

	var data commentData
	err = client.syncArticlesFromRootDirectory(path, &data)
	if err != nil {
		log.Fatalf("error synchronizing directory: %v", err)
	}

	if prComment != "" {
		err = renderTemplateToFile(prComment, commentTemplate, data)
		if err != nil {
			log.Fatalf("error writing PR comment: %v", err)
		}
	}

	if commit != "" {
		err = renderTemplateToFile(commit, commitTemplate, data)
		if err != nil {
			log.Fatalf("error writing commit: %v", err)
		}
	}
}

type client struct {
	*api.ClientWithResponses
	dryRun, createImage bool
	logger              *slog.Logger

	repositoryName, branch string
}

func newClient(apikey string, dryRun, createImage bool) (*client, error) {
	c, err := api.NewClientWithResponses("https://dev.to", api.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
		req.Header.Add("api-key", apikey)
		return nil
	}))
	if err != nil {
		return nil, fmt.Errorf("error creating client: %w", err)
	}

	return &client{c, dryRun, createImage, slog.New(slog.NewTextHandler(os.Stdout, nil)), "", ""}, nil
}

func (c *client) init(path string) error {
	err := os.MkdirAll(path, 0755)
	if err != nil {
		return fmt.Errorf("error creating directory: %w", err)
	}

	articles, err := c.getPublishedArticles()
	if err != nil {
		return fmt.Errorf("error getting articles: %w", err)
	}
	c.logger.Info("fetched articles", "count", len(articles))

	existingArticles, err := c.getExistingArticleIDs(path)
	if err != nil {
		return fmt.Errorf("error getting existing article IDs: %w", err)
	}
	c.logger.Info("existing articles", "count", len(existingArticles))

	for _, a := range articles {
		logger := c.logger.With("id", a.Id)
		logger.Info("creating article locally")

		_, exists := existingArticles[int(a.Id)]
		if exists {
			logger.Info("article exists")
			continue
		}

		fullArticle, err := c.getArticle(int(a.Id))
		if err != nil {
			logger.Error("error getting article", "error", err)
			continue
		}

		logger.Info("got full article details")

		articleDir := filepath.Join(path, a.Slug)
		err = os.MkdirAll(articleDir, 0755)
		if err != nil {
			return fmt.Errorf("error creating directory: %w", err)
		}

		logger.Info("created directory", "dir", articleDir)

		err = writeArticleFile(articleDir, &Article{
			ID:          int(a.Id),
			Slug:        a.Slug,
			Title:       a.Title,
			Description: a.Description,
			URL:         a.Url,
			Tags:        a.TagList,
		})
		if err != nil {
			return fmt.Errorf("error writing article JSON file: %w", err)
		}

		articleMarkdown, ok := fullArticle["body_markdown"].(string)
		if !ok {
			return fmt.Errorf("error checking body_markdown")
		}

		err = os.WriteFile(filepath.Join(articleDir, "article.md"), []byte(articleMarkdown), 0644)
		if err != nil {
			return fmt.Errorf("error writing article markdown file: %w", err)
		}

		logger.Info("added files", "dir", articleDir)
	}

	return nil
}

func (c *client) getExistingArticleIDs(rootDir string) (map[int]struct{}, error) {
	result := map[int]struct{}{}

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("error accessing path %s: %w", path, err)
		}

		if path == rootDir {
			return nil
		}

		if !d.IsDir() {
			return nil
		}

		c.logger.Info("checking for article", "directory", path)
		data, err := os.ReadFile(filepath.Join(path, "article.json"))
		if err != nil {
			return fmt.Errorf("error reading JSON file: %w", err)
		}

		var article *Article
		err = json.Unmarshal(data, &article)
		if err != nil {
			return fmt.Errorf("error parsing article details: %w", err)
		}

		c.logger.Info("found article", "id", article.ID)

		result[article.ID] = struct{}{}

		return nil
	})

	return result, err
}

func (c *client) syncArticlesFromRootDirectory(rootDir string, data *commentData) error {
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
		article, err := c.syncArticleFromDirectory(path)
		if err != nil {
			return fmt.Errorf("error synchronizing article from path %s: %w", path, err)
		}

		switch {
		case article.new:
			data.NewArticles = append(data.NewArticles, article)
		case article.updated:
			data.UpdatedArticles = append(data.UpdatedArticles, article)
		}

		return nil
	})
}

// syncArticleFromDirectory will read the article files from a directory and:
//   - If no ID is provided, create a new article and record ID
//   - Otherwise, get article by ID and compare text to local text. If the file is
//     recently changed, it will be updated by API
func (c *client) syncArticleFromDirectory(dir string) (*Article, error) {
	markdownBody, err := os.ReadFile(filepath.Join(dir, "article.md"))
	if err != nil {
		return nil, fmt.Errorf("error reading markdown: %w", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "article.json"))
	if err != nil {
		return nil, fmt.Errorf("error reading JSON file: %w", err)
	}

	var article *Article
	err = json.Unmarshal(data, &article)
	if err != nil {
		return nil, fmt.Errorf("error parsing article details: %w", err)
	}

	logger := c.logger.With("directory", dir).With("title", article.Title)

	var respBody []byte
	switch article.ID {
	case 0:
		article.new = true
		logger.Info("creating new article")

		if article.Gopher != "" {
			logger.With("gopher", article.Gopher).Info("creating gopher cover image")
		}
		if article.Gopher != "" && (c.createImage || !c.dryRun) {
			coverImg, err := createCoverImage(article.Gopher, article.Title)
			if err != nil {
				return nil, fmt.Errorf("error creating cover image: %w", err)
			}

			err = gg.SavePNG(filepath.Join(dir, "cover_image.png"), coverImg)
			if err != nil {
				return nil, fmt.Errorf("error saving image: %w", err)
			}
		}
		img := ""
		_, err = os.Stat(filepath.Join(dir, "cover_image.png"))
		if err == nil {
			img = fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/cover_image.png", c.repositoryName, c.branch, dir)
			logger.With("url", img).Info("adding image to article")
		}

		if c.dryRun {
			return article, nil
		}

		respBody, err = c.createArticle(article, string(markdownBody), img)
		if err != nil {
			return nil, fmt.Errorf("error creating article: %w", err)
		}
	default:
		logger = logger.With("id", article.ID)

		shouldUpdate, err := c.shouldUpdateArticle(string(markdownBody), article)
		if err != nil {
			return nil, fmt.Errorf("error checking if article needs update: %w", err)
		}
		if shouldUpdate == "" {
			logger.Info("article is up-to-date")
			return article, nil
		}
		logger.With("reason", shouldUpdate).Info("updating article")
		article.updated = true

		if c.dryRun {
			return article, nil
		}

		respBody, err = c.updateArticle(dir, article, string(markdownBody))
		if err != nil {
			return nil, fmt.Errorf("error updating article: %w", err)
		}
	}

	err = json.Unmarshal(respBody, &article)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response JSON: %w", err)
	}

	// article was created so logger doesn't already have ID
	if article.new {
		logger = logger.With("id", article.ID)
	}

	logger.Info("successfully synchronized article")

	err = writeArticleFile(dir, article)
	if err != nil {
		return nil, fmt.Errorf("error writing article JSON file: %w", err)
	}

	return article, nil
}

func writeArticleFile(path string, article *Article) error {
	data, err := json.MarshalIndent(article, "", "    ")
	if err != nil {
		return fmt.Errorf("error marshaling response JSON to write to file: %w", err)
	}

	err = os.WriteFile(filepath.Join(path, "article.json"), data, 0640)
	if err != nil {
		return fmt.Errorf("error writing JSON file: %w", err)
	}

	return nil
}

func (c *client) shouldUpdateArticle(markdownBody string, article *Article) (string, error) {
	articleData, err := c.getArticle(article.ID)
	if err != nil {
		return "", fmt.Errorf("error getting article: %w", err)
	}

	articleMarkdown, ok := articleData["body_markdown"].(string)
	if !ok {
		return "", fmt.Errorf("error checking body_markdown")
	}

	article.URL, ok = articleData["url"].(string)
	if !ok {
		return "", fmt.Errorf("error getting article url")
	}

	if articleMarkdown != markdownBody {
		return "body changed", nil
	}

	articleTags := articleData["tags"].([]interface{})
	existingTags := []string{}
	for _, tag := range articleTags {
		existingTags = append(existingTags, tag.(string))
	}

	if !slices.Equal[[]string, string](existingTags, article.Tags) {
		return "different tags", nil
	}

	return "", nil
}
