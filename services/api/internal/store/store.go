package store

import (
	"context"
	"errors"
	"time"

	"github.com/docsnap/docsnap/services/api/internal/model"
)

var ErrNotFound = errors.New("not found")

type Repository interface {
	Save(ctx context.Context, evidence model.Evidence) error
	GetEvidence(ctx context.Context, id string) (model.Evidence, error)
	Search(ctx context.Context, params SearchParams) (model.SearchResult, error)
	UpdateVerificationStatus(ctx context.Context, id string, status string) error
	UpdateAnchor(ctx context.Context, id string, result model.AnchorResult, submitter string) error
	GetClaim(ctx context.Context, id string) (model.Claim, error)
	SaveInvestigation(ctx context.Context, claimID string, v Investigation) error

	CreateUser(ctx context.Context, user model.User, passwordHash string) error
	GetUserByEmail(ctx context.Context, email string) (model.User, string, error)
	GetUserByID(ctx context.Context, id string) (model.User, error)
	CreateSession(ctx context.Context, token, userID string, expiresAt time.Time) error
	GetSessionUser(ctx context.Context, token string) (model.User, error)
	DeleteSession(ctx context.Context, token string) error

	FindSimilarCanonicalClaims(ctx context.Context, text string, limit int) ([]model.CanonicalClaim, error)
	CreateCanonicalClaim(ctx context.Context, cc model.CanonicalClaim) error
	GetCanonicalClaimBySlug(ctx context.Context, slug string) (model.CanonicalClaim, error)
	PublishClaim(ctx context.Context, claimID, canonicalClaimID, visibility, publishedBy string) error
	ForkClaim(ctx context.Context, parentID, newClaimID, ownerID string) (model.Claim, error)
	AddEvidenceContribution(ctx context.Context, contribution model.EvidenceContribution) error
	ReportContribution(ctx context.Context, contributionID, reporterID string) error
	Discover(ctx context.Context) (recent []model.Claim, trending []model.Claim, err error)
	DomainTrust(ctx context.Context, domain string) (model.DomainTrust, error)
}

type Investigation struct {
	Status     string
	Confidence float64
	Reasoning  model.ClaimReasoning
	Sources    []model.Source
}

type SearchParams struct {
	Query   string
	Company string
	Domain  string
	Status  string

	Owner string
	Limit int
}
