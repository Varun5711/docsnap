package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"

	"github.com/docsnap/docsnap/services/api/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func randomHex(size int) string {
	bytes := make([]byte, size)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

type Postgres struct {
	pool *pgxpool.Pool
}

func NewPostgres(ctx context.Context, databaseURL string) (*Postgres, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	if _, err := pool.Exec(ctx, schemaSQL); err != nil {
		pool.Close()
		return nil, err
	}
	return &Postgres{pool: pool}, nil
}

func (p *Postgres) Close() {
	p.pool.Close()
}

func (p *Postgres) Save(ctx context.Context, evidence model.Evidence) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO evidence (
			id, url, domain, title, company, case_id, user_id, screenshot_data_url, screenshot_object_key,
			scraped_text, screenshot_hash, scraped_text_hash, metadata_commitment,
			claims_root, evidence_commitment, flare_tx_hash, tee_certificate_hash,
			tee_signature, verification_status, published_by, anchor_submitter, captured_at, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23)
	`, evidence.ID, evidence.URL, evidence.Domain, evidence.Title, evidence.Company, evidence.CaseID, evidence.UserID,
		evidence.ScreenshotDataURL, evidence.ScreenshotObjectKey, evidence.ScrapedText, evidence.ScreenshotHash, evidence.ScrapedTextHash,
		evidence.MetadataCommitment, evidence.ClaimsRoot, evidence.EvidenceCommitment, evidence.FlareTxHash,
		evidence.TEECertificateHash, evidence.TEESignature, evidence.VerificationStatus, evidence.PublishedBy, evidence.AnchorSubmitter,
		evidence.CapturedAt, evidence.CreatedAt)
	if err != nil {
		return err
	}

	for _, claim := range evidence.Claims {
		_, err = tx.Exec(ctx, `
			INSERT INTO claims (id, evidence_id, text, type, confidence, source_excerpt, hash, status, published_by, visibility)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`, claim.ID, claim.EvidenceID, claim.Text, claim.Type, claim.Confidence, claim.SourceExcerpt, claim.Hash, claim.Status, claim.PublishedBy, claim.Visibility)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (p *Postgres) GetEvidence(ctx context.Context, id string) (model.Evidence, error) {
	item, err := p.getEvidenceBase(ctx, id)
	if err != nil {
		return model.Evidence{}, err
	}
	claims, err := p.claimsForEvidence(ctx, []string{id})
	if err != nil {
		return model.Evidence{}, err
	}
	item.Claims = claims[id]
	return item, nil
}

func (p *Postgres) Search(ctx context.Context, params SearchParams) (model.SearchResult, error) {
	if params.Limit <= 0 || params.Limit > 250 {
		params.Limit = 100
	}

	query := strings.TrimSpace(params.Query)
	company := strings.TrimSpace(params.Company)
	domain := strings.TrimSpace(params.Domain)
	status := strings.TrimSpace(params.Status)
	owner := strings.TrimSpace(params.Owner)

	rows, err := p.pool.Query(ctx, `
		SELECT e.id, e.url, e.domain, e.title, e.company, e.case_id, e.user_id,
			e.screenshot_data_url, e.screenshot_object_key, e.scraped_text, e.screenshot_hash, e.scraped_text_hash,
			e.metadata_commitment, e.claims_root, e.evidence_commitment, e.flare_tx_hash,
			e.tee_certificate_hash, e.tee_signature, e.verification_status, e.published_by, e.anchor_submitter, e.captured_at, e.created_at,
			MAX(COALESCE(GREATEST(
				similarity(c.text, $1),
				similarity(c.source_excerpt, $1),
				similarity(e.title, $1),
				similarity(e.url, $1)
			), 0)) AS rank
		FROM evidence e
		LEFT JOIN claims c ON c.evidence_id = e.id
		WHERE ($1 = '' OR c.text ILIKE '%' || $1 || '%' OR c.source_excerpt ILIKE '%' || $1 || '%' OR e.url ILIKE '%' || $1 || '%' OR e.title ILIKE '%' || $1 || '%' OR e.case_id ILIKE '%' || $1 || '%')
			AND ($2 = '' OR e.company ILIKE '%' || $2 || '%')
			AND ($3 = '' OR e.domain ILIKE '%' || $3 || '%')
			AND ($4 = '' OR e.verification_status = $4)
			AND ($5 = '' OR e.published_by = $5)
		GROUP BY e.id
		ORDER BY rank DESC, e.created_at DESC
		LIMIT $6
	`, query, company, domain, status, owner, params.Limit)
	if err != nil {
		return model.SearchResult{}, err
	}
	defer rows.Close()

	items := make([]model.Evidence, 0)
	ids := make([]string, 0)
	for rows.Next() {
		var item model.Evidence
		var rank float64
		err := rows.Scan(
			&item.ID, &item.URL, &item.Domain, &item.Title, &item.Company, &item.CaseID, &item.UserID,
			&item.ScreenshotDataURL, &item.ScreenshotObjectKey, &item.ScrapedText, &item.ScreenshotHash, &item.ScrapedTextHash,
			&item.MetadataCommitment, &item.ClaimsRoot, &item.EvidenceCommitment, &item.FlareTxHash,
			&item.TEECertificateHash, &item.TEESignature, &item.VerificationStatus, &item.PublishedBy, &item.AnchorSubmitter,
			&item.CapturedAt, &item.CreatedAt,
			&rank,
		)
		if err != nil {
			return model.SearchResult{}, err
		}
		items = append(items, item)
		ids = append(ids, item.ID)
	}
	if err := rows.Err(); err != nil {
		return model.SearchResult{}, err
	}

	claimsByEvidence, err := p.claimsForEvidence(ctx, ids)
	if err != nil {
		return model.SearchResult{}, err
	}

	claims := make([]model.Claim, 0)
	for i := range items {
		items[i].Claims = claimsByEvidence[items[i].ID]
		for _, claim := range items[i].Claims {
			if query == "" || containsFold(claim.Text, query) || containsFold(claim.SourceExcerpt, query) || containsFold(claim.Type, query) {
				claims = append(claims, claim)
			}
		}
	}

	return model.SearchResult{Claims: claims, Items: items}, nil
}

func (p *Postgres) GetClaim(ctx context.Context, id string) (model.Claim, error) {
	claim, entitiesJSON, reasoningJSON, err := p.scanClaimRow(ctx, id)
	if err != nil {
		return model.Claim{}, err
	}

	_ = json.Unmarshal([]byte(entitiesJSON), &claim.Entities)
	if claim.InvestigationStatus != "" {
		var reasoning model.ClaimReasoning
		if err := json.Unmarshal([]byte(reasoningJSON), &reasoning); err == nil {
			claim.Reasoning = &reasoning
		}
	}

	sources, err := p.sourcesForClaim(ctx, id)
	if err != nil {
		return model.Claim{}, err
	}
	claim.Sources = sources

	contributions, err := p.contributionsForClaim(ctx, id)
	if err != nil {
		return model.Claim{}, err
	}
	claim.Contributions = contributions

	return claim, nil
}

func (p *Postgres) scanClaimRow(ctx context.Context, id string) (model.Claim, string, string, error) {
	var claim model.Claim
	var entitiesJSON, reasoningJSON string
	var canonicalClaimID *string
	var canonicalClaimSlug *string
	var forkedFromClaimID *string
	row := p.pool.QueryRow(ctx, `
		SELECT c.id, c.evidence_id, c.text, c.type, c.confidence, c.source_excerpt, c.hash, c.status,
			c.subject, c.predicate, c.object, c.claim_date, c.location, c.entities,
			c.investigation_status, c.investigation_confidence, c.reasoning, c.investigated_at,
			c.canonical_claim_id, c.visibility, c.published_by, c.forked_from_claim_id, cc.slug
		FROM claims c
		LEFT JOIN canonical_claims cc ON cc.id = c.canonical_claim_id
		WHERE c.id = $1
	`, id)
	err := row.Scan(
		&claim.ID, &claim.EvidenceID, &claim.Text, &claim.Type, &claim.Confidence, &claim.SourceExcerpt, &claim.Hash, &claim.Status,
		&claim.Subject, &claim.Predicate, &claim.Object, &claim.ClaimDate, &claim.Location, &entitiesJSON,
		&claim.InvestigationStatus, &claim.InvestigationConfidence, &reasoningJSON, &claim.InvestigatedAt,
		&canonicalClaimID, &claim.Visibility, &claim.PublishedBy, &forkedFromClaimID, &canonicalClaimSlug,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Claim{}, "", "", ErrNotFound
	}
	if err != nil {
		return model.Claim{}, "", "", err
	}
	if canonicalClaimID != nil {
		claim.CanonicalClaimID = *canonicalClaimID
	}
	if canonicalClaimSlug != nil {
		claim.CanonicalClaimSlug = *canonicalClaimSlug
	}
	if forkedFromClaimID != nil {
		claim.ForkedFromClaimID = *forkedFromClaimID
	}
	return claim, entitiesJSON, reasoningJSON, nil
}

func (p *Postgres) sourcesForClaim(ctx context.Context, claimID string) ([]model.Source, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT id, claim_id, url, name, source_type, star_rating, relationship, relevance, captured_at
		FROM sources
		WHERE claim_id = $1
		ORDER BY relevance DESC
	`, claimID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sources := make([]model.Source, 0)
	for rows.Next() {
		var s model.Source
		if err := rows.Scan(&s.ID, &s.ClaimID, &s.URL, &s.Name, &s.SourceType, &s.StarRating, &s.Relationship, &s.Relevance, &s.CapturedAt); err != nil {
			return nil, err
		}
		sources = append(sources, s)
	}
	return sources, rows.Err()
}

func (p *Postgres) contributionsForClaim(ctx context.Context, claimID string) ([]model.EvidenceContribution, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT id, claim_id, contributor_id, type, url, note, created_at
		FROM evidence_contributions
		WHERE claim_id = $1
		ORDER BY created_at DESC
	`, claimID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	contributions := make([]model.EvidenceContribution, 0)
	for rows.Next() {
		var c model.EvidenceContribution
		if err := rows.Scan(&c.ID, &c.ClaimID, &c.ContributorID, &c.Type, &c.URL, &c.Note, &c.CreatedAt); err != nil {
			return nil, err
		}
		contributions = append(contributions, c)
	}
	return contributions, rows.Err()
}

func (p *Postgres) SaveInvestigation(ctx context.Context, claimID string, v Investigation) error {
	reasoningJSON, err := json.Marshal(v.Reasoning)
	if err != nil {
		return err
	}

	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		UPDATE claims
		SET investigation_status = $1, investigation_confidence = $2, reasoning = $3, investigated_at = now()
		WHERE id = $4
	`, v.Status, v.Confidence, string(reasoningJSON), claimID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	if _, err := tx.Exec(ctx, `DELETE FROM sources WHERE claim_id = $1`, claimID); err != nil {
		return err
	}
	for _, s := range v.Sources {
		id := s.ID
		if id == "" {
			id = "src_" + randomHex(10)
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO sources (id, claim_id, url, name, source_type, star_rating, relationship, relevance)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, id, claimID, s.URL, s.Name, s.SourceType, s.StarRating, s.Relationship, s.Relevance)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (p *Postgres) UpdateVerificationStatus(ctx context.Context, id string, status string) error {
	tag, err := p.pool.Exec(ctx, `UPDATE evidence SET verification_status = $1 WHERE id = $2`, status, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *Postgres) getEvidenceBase(ctx context.Context, id string) (model.Evidence, error) {
	row := p.pool.QueryRow(ctx, `
		SELECT id, url, domain, title, company, case_id, user_id, screenshot_data_url,
			screenshot_object_key, scraped_text, screenshot_hash, scraped_text_hash, metadata_commitment,
			claims_root, evidence_commitment, flare_tx_hash, tee_certificate_hash,
			tee_signature, verification_status, published_by, anchor_submitter, captured_at, created_at
		FROM evidence
		WHERE id = $1
	`, id)

	item, err := scanEvidence(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Evidence{}, ErrNotFound
	}
	if err != nil {
		return model.Evidence{}, err
	}
	return item, nil
}

func (p *Postgres) claimsForEvidence(ctx context.Context, evidenceIDs []string) (map[string][]model.Claim, error) {
	result := map[string][]model.Claim{}
	if len(evidenceIDs) == 0 {
		return result, nil
	}

	rows, err := p.pool.Query(ctx, `
		SELECT id, evidence_id, text, type, confidence, source_excerpt, hash, status
		FROM claims
		WHERE evidence_id = ANY($1)
		ORDER BY confidence DESC, id ASC
	`, evidenceIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var claim model.Claim
		if err := rows.Scan(&claim.ID, &claim.EvidenceID, &claim.Text, &claim.Type, &claim.Confidence, &claim.SourceExcerpt, &claim.Hash, &claim.Status); err != nil {
			return nil, err
		}
		result[claim.EvidenceID] = append(result[claim.EvidenceID], claim)
	}
	return result, rows.Err()
}

type evidenceScanner interface {
	Scan(dest ...any) error
}

func scanEvidence(row evidenceScanner) (model.Evidence, error) {
	var item model.Evidence
	err := row.Scan(
		&item.ID,
		&item.URL,
		&item.Domain,
		&item.Title,
		&item.Company,
		&item.CaseID,
		&item.UserID,
		&item.ScreenshotDataURL,
		&item.ScreenshotObjectKey,
		&item.ScrapedText,
		&item.ScreenshotHash,
		&item.ScrapedTextHash,
		&item.MetadataCommitment,
		&item.ClaimsRoot,
		&item.EvidenceCommitment,
		&item.FlareTxHash,
		&item.TEECertificateHash,
		&item.TEESignature,
		&item.VerificationStatus,
		&item.PublishedBy,
		&item.AnchorSubmitter,
		&item.CapturedAt,
		&item.CreatedAt,
	)
	return item, err
}

func (p *Postgres) DomainTrust(ctx context.Context, domain string) (model.DomainTrust, error) {
	var total, contradicted, supported int
	err := p.pool.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE c.investigation_status <> ''),
			COUNT(*) FILTER (WHERE c.investigation_status IN ('CONTRADICTED', 'LIKELY_CONTRADICTED')),
			COUNT(*) FILTER (WHERE c.investigation_status IN ('SUPPORTED', 'LIKELY_SUPPORTED'))
		FROM claims c
		JOIN evidence e ON e.id = c.evidence_id
		WHERE e.domain = $1
	`, domain).Scan(&total, &contradicted, &supported)
	if err != nil {
		return model.DomainTrust{}, err
	}
	return model.NewDomainTrust(domain, total, contradicted, supported), nil
}

func (p *Postgres) UpdateAnchor(ctx context.Context, id string, result model.AnchorResult, submitter string) error {
	_, err := p.pool.Exec(ctx, `
		UPDATE evidence
		SET flare_tx_hash = $2, tee_certificate_hash = $3, verification_status = $4, anchor_submitter = $5
		WHERE id = $1
	`, id, result.TxHash, result.TEECertificateHash, result.Status, submitter)
	return err
}

func containsFold(value string, query string) bool {
	return strings.Contains(strings.ToLower(value), strings.ToLower(query))
}
