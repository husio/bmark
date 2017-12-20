package bmark

import (
	"context"
	"strings"
	"time"
)

type Page struct {
	PageID    int64
	URL       string
	Title     string
	Content   string
	CreatedAt time.Time
}

type PageStore interface {
	ListPages(ctx context.Context, limit int, createdLte time.Time) ([]*Page, error)
	AddPage(ctx context.Context, url, title, content string) (int64, error)
	DelPage(ctx context.Context, pageID int64) error
	PageWithSurrounding(ctx context.Context, pageID int64) (prev, current, next *Page, err error)
}

func ErrNotFound(message string) error {
	return &storageError{
		kind:    kindNotFound,
		message: message,
	}
}

func IsNotFound(err error) bool {
	if e, ok := err.(*storageError); ok {
		return e.kind == kindNotFound
	}
	return false
}

func ErrConflict(original error, message string) error {
	return &storageError{
		original: original,
		kind:     kindConflict,
		message:  message,
	}
}

func IsConflict(err error) bool {
	if e, ok := err.(*storageError); ok {
		return e.kind == kindConflict
	}
	return false
}

type storageError struct {
	original error
	kind     uint16
	message  string
}

func ErrTransactionBegin(original error) error {
	return &storageError{
		original: original,
		kind:     kindTxBegin,
	}
}

func IsTransactionBegin(err error) bool {
	if e, ok := err.(*storageError); ok {
		return e.kind == kindTxBegin
	}
	return false
}

func ErrTransactionEnd(original error) error {
	return &storageError{
		original: original,
		kind:     kindTxEnd,
	}
}

func IsTransactionEnd(err error) bool {
	if e, ok := err.(*storageError); ok {
		return e.kind == kindTxEnd
	}
	return false
}

func (e *storageError) Error() string {
	var chunks []string

	switch e.kind {
	case kindNotFound:
		chunks = append(chunks, "not found: ")
	case kindConflict:
		chunks = append(chunks, "conflict: ")
	case kindTxBegin:
		chunks = append(chunks, "tx begin: ")
	case kindTxEnd:
		chunks = append(chunks, "tx end: ")
	}

	if len(e.message) != 0 {
		chunks = append(chunks, e.message, ": ")
	}

	if e.original != nil {
		chunks = append(chunks, e.original.Error())
	}

	return strings.Join(chunks, "")
}

const (
	_ = iota
	kindNotFound
	kindConflict
	kindTxBegin
	kindTxEnd
)
