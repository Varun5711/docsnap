package store

import (
	"context"
	"errors"

	"github.com/docsnap/docsnap/services/api/internal/model"
)

var ErrNotFound = errors.New("not found")

type Repository interface {
	Save(ctx context.Context, evidence model.Evidence) error
	GetEvidence(ctx context.Context, id string) (model.Evidence, error)
	Search(ctx context.Context, params SearchParams) (model.SearchResult, error)
}

type SearchParams struct {
	Query   string
	Company string
	Domain  string
	Status  string
	Limit   int
}
