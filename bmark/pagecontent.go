package bmark

import "context"

type ArticleService interface {
	FetchArticle(ctx context.Context, url string) (*Article, error)
}

type Article struct {
	URL     string
	Title   string
	Content string
}
