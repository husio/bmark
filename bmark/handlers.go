package bmark

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/husio/bmark/pkg/surf"
)

func LatestPageHandler(
	store PageStore,
	rend surf.Renderer,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch page, err := store.LatestPage(r.Context()); {
		case err == nil:
			http.Redirect(w, r, fmt.Sprintf("/p/%d/", page.PageID), http.StatusSeeOther)
		case IsNotFound(err):
			rend.RenderStdResponse(w, http.StatusNotFound)
		default:
			surf.Error(r.Context(), err, "cannot get latest page")
			rend.RenderStdResponse(w, http.StatusInternalServerError)
		}
	}
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
