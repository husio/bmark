package main

import (
	"context"
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/husio/bmark/bmark"
	"github.com/husio/bmark/pkg/surf"
	"github.com/husio/envconf"
)

func main() {
	conf := configuration{
		Postgres:  "dbname=postgres user=postgres sslmode=disable",
		HTTPPort:  8000,
		SecretKey: "",
	}
	envconf.Parse(&conf)

	db, err := sql.Open("postgres", conf.Postgres)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	logger := surf.NewLogger(os.Stderr)

	pageStore, err := bmark.NewPostgresPageStore(db)
	if err != nil {
		panic(err)
	}

	articles := bmark.NewMercuryArticleService(conf.MercuryApiKey)

	rend := surf.NewHTMLRenderer("**/templates/**.tmpl", template.FuncMap{
		"timeago": timeago,
		"slugify": slugify,
		"safehtml": func(s string) template.HTML {
			return template.HTML(s)
		},
	})

	rt := surf.NewRouter()
	rt.Get(`/`, bmark.PagesListHandler(pageStore, rend))
	rt.Get(`/p<page-id:\d+>/.*`, bmark.PageHandler(pageStore, rend))
	rt.Any(`/add/`, bmark.AddPageHandler(pageStore, articles, conf.SecretKey, rend))

	app := surf.NewHTTPApplication(rt, false, logger)

	addr := fmt.Sprintf("0.0.0.0:%d", conf.HTTPPort)
	logger.Info(context.Background(), "starting HTTP server",
		"address", addr)
	if err := http.ListenAndServe(addr, app); err != nil {
		panic(err)
	}
}

type configuration struct {
	MercuryApiKey string
	HTTPPort      int    `envconf:"PORT"`
	Postgres      string `envconf:"DATABASE_URL"`
	SecretKey     string
}

func timeago(t time.Time) string {
	d := time.Now().Sub(t)
	if d > 24*time.Hour {
		days := int(d / (24 * time.Hour))
		if days == 1 {
			return "1 day"
		}
		return fmt.Sprintf("%d days", days)
	}
	if d > time.Hour {
		hours := int(d / time.Hour)
		if hours == 1 {
			return "1 hour"
		}
		return fmt.Sprintf("%d hours", hours)
	}
	if d > time.Minute {
		mins := int(d / time.Minute)
		if mins == 1 {
			return "1 minute"
		}
		return fmt.Sprintf("%d minutes", mins)
	}
	return "just now"
}

func slugify(s string) string {
	return strings.Trim(slugifyrx.ReplaceAllString(strings.ToLower(s), "-"), "-")
}

var slugifyrx = regexp.MustCompile("[^a-z0-9]+")
