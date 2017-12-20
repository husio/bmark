package bmark

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/husio/bmark/pkg/surf"
)

func PagesListHandler(
	store PageStore,
	rend surf.Renderer,
) http.HandlerFunc {
	const pageSize = 25

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		createdLte := time.Now()
		if t, ok := parseTime(r.URL.Query().Get("olderThan")); ok {
			createdLte = t
		}

		pages, err := store.ListPages(ctx, pageSize, createdLte)
		if err != nil {
			surf.Error(ctx, err, "cannot list pages")
			rend.RenderStdResponse(w, http.StatusInternalServerError)
			return
		}

		var next *time.Time
		if len(pages) == pageSize {
			next = &pages[pageSize-1].CreatedAt
		}

		rend.RenderResponse(w, http.StatusOK, "pages_list.tmpl", struct {
			CreatedLte time.Time
			Next       *time.Time
			Pages      []*Page
		}{
			CreatedLte: createdLte,
			Next:       next,
			Pages:      pages,
		})
	}
}

func parseTime(raw string) (time.Time, bool) {
	for _, format := range timeFormats {
		if t, err := time.Parse(format, raw); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

var timeFormats = []string{
	time.RFC3339,
	time.RFC3339Nano,
	"2006-01-02T150405",
	"2006-01-02T15:04:05",
	"2006-01-02T15:04",
	"2006-01-02",
}

func PageHandler(
	store PageStore,
	rend surf.Renderer,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		pageID, _ := strconv.ParseInt(surf.PathArg(r, 0), 10, 64)
		prev, current, next, err := store.PageWithSurrounding(ctx, pageID)
		if err != nil {
			if IsNotFound(err) {
				surf.Info(ctx, "page not found")
				rend.RenderStdResponse(w, http.StatusNotFound)
			} else {
				surf.Error(ctx, err, "cannot get page")
				rend.RenderStdResponse(w, http.StatusInternalServerError)
			}
			return
		}

		rend.RenderResponse(w, http.StatusOK, "page.tmpl", struct {
			NextPage *Page
			Page     *Page
			PrevPage *Page
		}{
			NextPage: next,
			Page:     current,
			PrevPage: prev,
		})
	}
}

func AddPageHandler(
	store PageStore,
	articles ArticleService,
	secretKey string,
	rend surf.Renderer,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		url := strings.TrimSpace(r.FormValue("url"))
		if len(url) == 0 {
			rend.RenderStdResponse(w, http.StatusBadRequest)
			return
		}

		if secretKey != "" && secretKey != r.FormValue("key") {
			surf.Info(ctx, "page add rejected due to invalid key",
				"key", r.FormValue("key"))
			rend.RenderStdResponse(w, http.StatusForbidden)
			return
		}

		article, err := articles.FetchArticle(ctx, url)
		if err != nil {
			surf.Error(ctx, err, "cannot fetch article",
				"url", url)
			rend.RenderStdResponse(w, http.StatusInternalServerError)
			return
		}

		pageID, err := store.AddPage(ctx, article.URL, article.Title, article.Content)
		if err != nil {
			surf.Error(ctx, err, "cannot add page",
				"url", url)
			rend.RenderStdResponse(w, http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, fmt.Sprintf("/p/%d/", pageID), http.StatusSeeOther)
	}
}
