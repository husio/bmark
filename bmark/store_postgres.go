package bmark

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

func NewPostgresPageStore(db *sql.DB) (PageStore, error) {
	store := &pgPageStore{
		db: db,
	}
	return store, store.ensureSchema()
}

type pgPageStore struct {
	db *sql.DB
}

func (s *pgPageStore) ensureSchema() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS pages (
			page_id SERIAL PRIMARY KEY,
			url TEXT UNIQUE NOT NULL,
			title TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL
		)
	`)
	return err
}

func (s *pgPageStore) AddPage(ctx context.Context, url, title, content string) (int64, error) {
	var pageID int64
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO pages (url, title, content, created_at)
		VALUES ($1, $2, $3, $4)
		RETURNING page_id
	`, url, title, content, time.Now()).Scan(&pageID)
	if err != nil {
		return 0, err
	}
	return pageID, nil
}

func (s *pgPageStore) DelPage(ctx context.Context, pageID int64) error {
	res, err := s.db.Exec(`
		DELETE FROM pages
		WHERE page_id = $1
	`, pageID)
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrNotFound("page not found")
	}
	return nil
}

func (s *pgPageStore) PageWithSurrounding(ctx context.Context, pageID int64) (*Page, *Page, *Page, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, nil, nil, ErrTransactionBegin(err)
	}
	defer tx.Rollback()

	var prev, current, next *Page

	current = &Page{}
	err = tx.QueryRowContext(ctx, `
		SELECT page_id, url, title, content, created_at
		FROM pages
		WHERE page_id = $1
		LIMIT 1
	`, pageID).Scan(&current.PageID, &current.URL, &current.Title, &current.Content, &current.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, nil, ErrNotFound(fmt.Sprintf("no page with id=%d", pageID))
		}
		return nil, nil, nil, err
	}

	next = &Page{}
	err = tx.QueryRowContext(ctx, `
		SELECT page_id, url, title, content, created_at
		FROM pages
		WHERE created_at > $1
		ORDER BY created_at ASC
		LIMIT 1
	`, current.CreatedAt).Scan(&next.PageID, &next.URL, &next.Title, &next.Content, &next.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			next = nil
		} else {
			return nil, nil, nil, err
		}
	}

	prev = &Page{}
	err = tx.QueryRowContext(ctx, `
		SELECT page_id, url, title, content, created_at
		FROM pages
		WHERE created_at < $1
		ORDER BY created_at DESC
		LIMIT 1
	`, current.CreatedAt).Scan(&prev.PageID, &prev.URL, &prev.Title, &prev.Content, &prev.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			prev = nil
		} else {
			return nil, nil, nil, err
		}
	}

	return prev, current, next, nil
}

func (s *pgPageStore) ListPages(ctx context.Context, limit int, createdLte time.Time) ([]*Page, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT page_id, url, title, created_at
		FROM pages
		WHERE created_at <= $1
		ORDER BY created_at DESC
		LIMIT $2
	`, createdLte, limit)
	if err != nil {
		return nil, fmt.Errorf("cannot query pages: %s", err)
	}
	defer rows.Close()

	pages := make([]*Page, 0, limit)
	for rows.Next() {
		var p Page
		if err := rows.Scan(&p.PageID, &p.URL, &p.Title, &p.CreatedAt); err != nil {
			rows.Close()
			return pages, fmt.Errorf("cannot scan page: %s", err)
		}
		pages = append(pages, &p)
	}
	return pages, rows.Close()
}
