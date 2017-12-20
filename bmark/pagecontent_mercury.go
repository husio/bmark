package bmark

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
)

func NewMercuryArticleService(apiKey string) ArticleService {
	return &mercuryArticleService{
		apiUrl: "https://mercury.postlight.com",
		apiKey: apiKey,
		client: http.DefaultClient,
	}
}

type mercuryArticleService struct {
	apiUrl string
	apiKey string
	client *http.Client
}

func (s *mercuryArticleService) FetchArticle(ctx context.Context, articleUrl string) (*Article, error) {
	fetchUrl := s.apiUrl + "/parser?url=" + url.QueryEscape(articleUrl)
	req, err := http.NewRequest("GET", fetchUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot create request: %s", err)
	}
	req = req.WithContext(ctx)
	req.Header.Set("x-api-key", s.apiKey)
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot execute request: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := ioutil.ReadAll(io.LimitReader(resp.Body, 1e5))
		return nil, fmt.Errorf("invalid response %d: %s", resp.StatusCode, string(b))
	}

	var article Article
	if err := json.NewDecoder(resp.Body).Decode(&article); err != nil {
		return nil, fmt.Errorf("cannot decode response body: %s", err)
	}
	return &article, nil
}
