package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/docsnap/docsnap/services/api/internal/model"
	"github.com/jackc/pgx/v5"
)

func (p *Postgres) CreateUser(ctx context.Context, user model.User, passwordHash string) error {
	_, err := p.pool.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, display_name, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, user.ID, user.Email, passwordHash, user.DisplayName, user.CreatedAt)
	return err
}

func (p *Postgres) GetUserByEmail(ctx context.Context, email string) (model.User, string, error) {
	var user model.User
	var passwordHash string
	err := p.pool.QueryRow(ctx, `
		SELECT id, email, display_name, created_at, password_hash FROM users WHERE email = $1
	`, email).Scan(&user.ID, &user.Email, &user.DisplayName, &user.CreatedAt, &passwordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.User{}, "", ErrNotFound
	}
	if err != nil {
		return model.User{}, "", err
	}
	return user, passwordHash, nil
}

func (p *Postgres) GetUserByID(ctx context.Context, id string) (model.User, error) {
	var user model.User
	err := p.pool.QueryRow(ctx, `
		SELECT id, email, display_name, created_at FROM users WHERE id = $1
	`, id).Scan(&user.ID, &user.Email, &user.DisplayName, &user.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.User{}, ErrNotFound
	}
	return user, err
}

func (p *Postgres) CreateSession(ctx context.Context, token, userID string, expiresAt time.Time) error {
	_, err := p.pool.Exec(ctx, `
		INSERT INTO sessions (token, user_id, expires_at) VALUES ($1, $2, $3)
	`, token, userID, expiresAt)
	return err
}

func (p *Postgres) GetSessionUser(ctx context.Context, token string) (model.User, error) {
	var user model.User
	err := p.pool.QueryRow(ctx, `
		SELECT u.id, u.email, u.display_name, u.created_at
		FROM sessions s JOIN users u ON u.id = s.user_id
		WHERE s.token = $1 AND s.expires_at > now()
	`, token).Scan(&user.ID, &user.Email, &user.DisplayName, &user.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.User{}, ErrNotFound
	}
	return user, err
}

func (p *Postgres) DeleteSession(ctx context.Context, token string) error {
	_, err := p.pool.Exec(ctx, `DELETE FROM sessions WHERE token = $1`, token)
	return err
}

func (p *Postgres) FindSimilarCanonicalClaims(ctx context.Context, text string, limit int) ([]model.CanonicalClaim, error) {
	if limit <= 0 || limit > 20 {
		limit = 5
	}
	rows, err := p.pool.Query(ctx, `
		SELECT id, slug, text, created_at
		FROM canonical_claims
		WHERE similarity(text, $1) > 0.3
		ORDER BY similarity(text, $1) DESC
		LIMIT $2
	`, text, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]model.CanonicalClaim, 0)
	for rows.Next() {
		var cc model.CanonicalClaim
		if err := rows.Scan(&cc.ID, &cc.Slug, &cc.Text, &cc.CreatedAt); err != nil {
			return nil, err
		}
		results = append(results, cc)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range results {
		claims, err := p.claimsForCanonical(ctx, results[i].ID)
		if err != nil {
			return nil, err
		}
		results[i].Claims = claims
	}
	return results, nil
}

func (p *Postgres) CreateCanonicalClaim(ctx context.Context, cc model.CanonicalClaim) error {
	_, err := p.pool.Exec(ctx, `
		INSERT INTO canonical_claims (id, slug, text, created_at) VALUES ($1, $2, $3, $4)
	`, cc.ID, cc.Slug, cc.Text, cc.CreatedAt)
	return err
}

func (p *Postgres) GetCanonicalClaimBySlug(ctx context.Context, slug string) (model.CanonicalClaim, error) {
	var cc model.CanonicalClaim
	err := p.pool.QueryRow(ctx, `
		SELECT id, slug, text, created_at FROM canonical_claims WHERE slug = $1
	`, slug).Scan(&cc.ID, &cc.Slug, &cc.Text, &cc.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.CanonicalClaim{}, ErrNotFound
	}
	if err != nil {
		return model.CanonicalClaim{}, err
	}

	claims, err := p.claimsForCanonical(ctx, cc.ID)
	if err != nil {
		return model.CanonicalClaim{}, err
	}
	cc.Claims = claims
	return cc, nil
}

func (p *Postgres) claimsForCanonical(ctx context.Context, canonicalClaimID string) ([]model.Claim, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT id, evidence_id, text, type, confidence, source_excerpt, hash, status,
			investigation_status, investigation_confidence, investigated_at,
			visibility, published_by, forked_from_claim_id
		FROM claims
		WHERE canonical_claim_id = $1
		ORDER BY created_at DESC
	`, canonicalClaimID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	claims := make([]model.Claim, 0)
	for rows.Next() {
		var c model.Claim
		var forkedFrom *string
		if err := rows.Scan(
			&c.ID, &c.EvidenceID, &c.Text, &c.Type, &c.Confidence, &c.SourceExcerpt, &c.Hash, &c.Status,
			&c.InvestigationStatus, &c.InvestigationConfidence, &c.InvestigatedAt,
			&c.Visibility, &c.PublishedBy, &forkedFrom,
		); err != nil {
			return nil, err
		}
		if forkedFrom != nil {
			c.ForkedFromClaimID = *forkedFrom
		}
		c.CanonicalClaimID = canonicalClaimID
		claims = append(claims, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	ids := make([]string, len(claims))
	for i, c := range claims {
		ids[i] = c.ID
	}
	sourcesByClaim, err := p.sourcesForClaims(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range claims {
		claims[i].Sources = sourcesByClaim[claims[i].ID]
	}
	return claims, nil
}

func (p *Postgres) PublishClaim(ctx context.Context, claimID, canonicalClaimID, visibility, publishedBy string) error {
	tag, err := p.pool.Exec(ctx, `
		UPDATE claims SET canonical_claim_id = $1, visibility = $2, published_by = $3 WHERE id = $4
	`, canonicalClaimID, visibility, publishedBy, claimID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *Postgres) ForkClaim(ctx context.Context, parentID, newClaimID, ownerID string) (model.Claim, error) {
	parent, err := p.GetClaim(ctx, parentID)
	if err != nil {
		return model.Claim{}, err
	}

	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return model.Claim{}, err
	}
	defer tx.Rollback(ctx)

	entitiesJSON, _ := marshalOrEmpty(parent.Entities, "[]")

	var canonicalClaimID *string
	if parent.CanonicalClaimID != "" {
		canonicalClaimID = &parent.CanonicalClaimID
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO claims (
			id, evidence_id, text, type, confidence, source_excerpt, hash, status,
			subject, predicate, object, claim_date, location, entities,
			canonical_claim_id, visibility, published_by, forked_from_claim_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, 'private', $16, $17)
	`, newClaimID, parent.EvidenceID, parent.Text, parent.Type, parent.Confidence, parent.SourceExcerpt, parent.Hash, parent.Status,
		parent.Subject, parent.Predicate, parent.Object, parent.ClaimDate, parent.Location, entitiesJSON,
		canonicalClaimID, ownerID, parentID)
	if err != nil {
		return model.Claim{}, err
	}

	for _, s := range parent.Sources {
		_, err := tx.Exec(ctx, `
			INSERT INTO sources (id, claim_id, url, name, source_type, star_rating, relationship, relevance)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, "src_"+randomHex(10), newClaimID, s.URL, s.Name, s.SourceType, s.StarRating, s.Relationship, s.Relevance)
		if err != nil {
			return model.Claim{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return model.Claim{}, err
	}

	return p.GetClaim(ctx, newClaimID)
}

func (p *Postgres) AddEvidenceContribution(ctx context.Context, c model.EvidenceContribution) error {
	_, err := p.pool.Exec(ctx, `
		INSERT INTO evidence_contributions (id, claim_id, contributor_id, type, url, note, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, c.ID, c.ClaimID, c.ContributorID, c.Type, c.URL, c.Note, c.CreatedAt)
	return err
}

// ReportContribution is idempotent — the (contribution_id, reporter_id)
// unique constraint means a repeated report from the same person is a
// silent no-op rather than an error the caller has to special-case.
func (p *Postgres) ReportContribution(ctx context.Context, contributionID, reporterID string) error {
	_, err := p.pool.Exec(ctx, `
		INSERT INTO contribution_reports (id, contribution_id, reporter_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (contribution_id, reporter_id) DO NOTHING
	`, "rep_"+randomHex(10), contributionID, reporterID)
	return err
}

func (p *Postgres) Discover(ctx context.Context) ([]model.Claim, []model.Claim, error) {
	recent, err := p.publicClaims(ctx, `
		SELECT c.id, c.evidence_id, c.text, c.type, c.confidence, c.source_excerpt, c.hash, c.status,
			c.investigation_status, c.investigation_confidence, c.investigated_at,
			c.visibility, c.published_by, c.canonical_claim_id
		FROM claims c
		WHERE c.visibility = 'public' AND c.investigation_status != ''
		ORDER BY c.investigated_at DESC
		LIMIT 12
	`)
	if err != nil {
		return nil, nil, err
	}

	trending, err := p.publicClaims(ctx, `
		SELECT c.id, c.evidence_id, c.text, c.type, c.confidence, c.source_excerpt, c.hash, c.status,
			c.investigation_status, c.investigation_confidence, c.investigated_at,
			c.visibility, c.published_by, c.canonical_claim_id
		FROM claims c
		WHERE c.visibility = 'public' AND c.investigation_status != ''
		ORDER BY (SELECT count(*) FROM sources WHERE sources.claim_id = c.id) DESC, c.investigated_at DESC
		LIMIT 8
	`)
	if err != nil {
		return nil, nil, err
	}

	return recent, trending, nil
}

func (p *Postgres) publicClaims(ctx context.Context, query string) ([]model.Claim, error) {
	rows, err := p.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	claims := make([]model.Claim, 0)
	ids := make([]string, 0)
	for rows.Next() {
		var c model.Claim
		var canonicalClaimID *string
		if err := rows.Scan(
			&c.ID, &c.EvidenceID, &c.Text, &c.Type, &c.Confidence, &c.SourceExcerpt, &c.Hash, &c.Status,
			&c.InvestigationStatus, &c.InvestigationConfidence, &c.InvestigatedAt,
			&c.Visibility, &c.PublishedBy, &canonicalClaimID,
		); err != nil {
			return nil, err
		}
		if canonicalClaimID != nil {
			c.CanonicalClaimID = *canonicalClaimID
		}
		claims = append(claims, c)
		ids = append(ids, c.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	sourcesByClaim, err := p.sourcesForClaims(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range claims {
		claims[i].Sources = sourcesByClaim[claims[i].ID]
	}
	return claims, nil
}

// sourcesForClaims batches sourcesForClaim across many claims in one query
// — Discover's "0 sources" everywhere was this exact gap: publicClaims()
// never fetched sources at all, so every claim showed an empty list
// regardless of how many sources it actually had.
func (p *Postgres) sourcesForClaims(ctx context.Context, claimIDs []string) (map[string][]model.Source, error) {
	result := map[string][]model.Source{}
	if len(claimIDs) == 0 {
		return result, nil
	}
	rows, err := p.pool.Query(ctx, `
		SELECT id, claim_id, url, name, source_type, star_rating, relationship, relevance, captured_at
		FROM sources
		WHERE claim_id = ANY($1)
		ORDER BY relevance DESC
	`, claimIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var s model.Source
		if err := rows.Scan(&s.ID, &s.ClaimID, &s.URL, &s.Name, &s.SourceType, &s.StarRating, &s.Relationship, &s.Relevance, &s.CapturedAt); err != nil {
			return nil, err
		}
		result[s.ClaimID] = append(result[s.ClaimID], s)
	}
	return result, rows.Err()
}

func marshalOrEmpty(v any, fallback string) (string, error) {
	b, err := json.Marshal(v)
	if err != nil || string(b) == "null" {
		return fallback, nil
	}
	return string(b), nil
}
