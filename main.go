package main

import (
	"context"
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"os"

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
		"safehtml": func(s string) template.HTML {
			return template.HTML(s)
		},
	})

	rt := surf.NewRouter()
	rt.Get(`/`, bmark.PagesListHandler(pageStore, rend))
	rt.Get(`/p/<page-id:\d+>/`, bmark.PageHandler(pageStore, rend))
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
