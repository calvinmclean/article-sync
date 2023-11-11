package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/calvinmclean/article-sync/api"
)

func (c *client) getPublishedArticles() ([]api.ArticleIndex, error) {
	resp, err := doWithRetry(func() (*api.GetUserPublishedArticlesResponse, error) {
		return c.GetUserPublishedArticlesWithResponse(context.Background(), nil)
	}, 5, 1*time.Second)
	if err != nil {
		return nil, fmt.Errorf("error getting articles: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("unexpected status getting articles: %d %s", resp.StatusCode(), string(resp.Body))
	}

	return *resp.JSON200, nil
}

func (c *client) updateArticle(dir string, article *Article, markdownBody string) ([]byte, error) {
	published := true
	articleBody := api.Article{}
	articleBody.Article = &struct {
		BodyMarkdown   *string   "json:\"body_markdown,omitempty\""
		CanonicalUrl   *string   "json:\"canonical_url\""
		Description    *string   "json:\"description,omitempty\""
		MainImage      *string   "json:\"main_image\""
		OrganizationId *int      "json:\"organization_id\""
		Published      *bool     "json:\"published,omitempty\""
		Series         *string   "json:\"series\""
		Tags           *[]string "json:\"tags,omitempty\""
		Title          *string   "json:\"title,omitempty\""
	}{
		Title:        &article.Title,
		Description:  &article.Description,
		BodyMarkdown: &markdownBody,
		Published:    &published,
		Tags:         &article.Tags,
	}

	resp, err := doWithRetry(func() (*api.UpdateArticleResponse, error) {
		return c.UpdateArticleWithResponse(context.Background(), int32(article.ID), articleBody)
	}, 5, 1*time.Second)
	if err != nil {
		return nil, fmt.Errorf("error updating article: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("unexpected status updating article %d: %d %s", article.ID, resp.StatusCode(), string(resp.Body))
	}

	return resp.Body, nil
}

func (c *client) getArticle(id int) (map[string]interface{}, error) {
	resp, err := doWithRetry(func() (*api.GetArticleByIdResponse, error) {
		return c.GetArticleByIdWithResponse(context.Background(), id)
	}, 5, 1*time.Second)
	if err != nil {
		return nil, fmt.Errorf("error getting article %d: %w", id, err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("unexpected status getting article %d: %d %s", id, resp.StatusCode(), string(resp.Body))
	}

	return *resp.JSON200, nil
}

func (c *client) createArticle(article *Article, body, img string) ([]byte, error) {
	published := true
	articleBody := api.Article{}
	articleBody.Article = &struct {
		BodyMarkdown   *string   "json:\"body_markdown,omitempty\""
		CanonicalUrl   *string   "json:\"canonical_url\""
		Description    *string   "json:\"description,omitempty\""
		MainImage      *string   "json:\"main_image\""
		OrganizationId *int      "json:\"organization_id\""
		Published      *bool     "json:\"published,omitempty\""
		Series         *string   "json:\"series\""
		Tags           *[]string "json:\"tags,omitempty\""
		Title          *string   "json:\"title,omitempty\""
	}{
		Title:        &article.Title,
		Description:  &article.Description,
		BodyMarkdown: &body,
		Published:    &published,
		Tags:         &article.Tags,
		MainImage:    &img,
	}

	resp, err := doWithRetry(func() (*api.CreateArticleResponse, error) {
		return c.CreateArticleWithResponse(context.Background(), articleBody)
	}, 5, 1*time.Second)
	if err != nil {
		return nil, fmt.Errorf("error creating article: %w", err)
	}

	if resp.StatusCode() != http.StatusCreated {
		return nil, fmt.Errorf("unexpected status creating article: %d %s", resp.StatusCode(), string(resp.Body))
	}

	return resp.Body, nil
}

type response interface {
	StatusCode() int
}

func doWithRetry[T response](f func() (T, error), numRetries int, initialWait time.Duration) (T, error) {
	for i := 1; i <= numRetries; i++ {
		result, err := f()
		if err != nil {
			return *new(T), err
		}

		if result.StatusCode() == http.StatusTooManyRequests {
			time.Sleep(initialWait * time.Duration(i))
			continue
		}

		return result, err
	}

	return *new(T), fmt.Errorf("exhausted retry limit %d", numRetries)
}
