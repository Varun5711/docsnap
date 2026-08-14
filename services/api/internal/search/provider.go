package search

import "context"

type Result struct {
	URL     string
	Title   string
	Content string
	Score   float64
}

type Provider interface {
	Search(ctx context.Context, query string) ([]Result, error)
}
