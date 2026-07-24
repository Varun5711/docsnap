package store

import (
	"context"
	"errors"
	"strings"

	"github.com/docsnap/docsnap/services/api/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

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
			tee_signature, verification_status, captured_at, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
	`, evidence.ID, evidence.URL, evidence.Domain, evidence.Title, evidence.Company, evidence.CaseID, evidence.UserID,
		evidence.ScreenshotDataURL, evidence.ScreenshotObjectKey, evidence.ScrapedText, evidence.ScreenshotHash, evidence.ScrapedTextHash,
		evidence.MetadataCommitment, evidence.ClaimsRoot, evidence.EvidenceCommitment, evidence.FlareTxHash,
		evidence.TEECertificateHash, evidence.TEESignature, evidence.VerificationStatus, evidence.CapturedAt, evidence.CreatedAt)
	if err != nil {
		return err
	}

	for _, claim := range evidence.Claims {
		_, err = tx.Exec(ctx, `
			INSERT INTO claims (id, evidence_id, text, type, confidence, source_excerpt, hash, status)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, claim.ID, claim.EvidenceID, claim.Text, claim.Type, claim.Confidence, claim.SourceExcerpt, claim.Hash, claim.Status)
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

	rows, err := p.pool.Query(ctx, `
		SELECT DISTINCT e.id, e.url, e.domain, e.title, e.company, e.case_id, e.user_id,
			e.screenshot_data_url, e.screenshot_object_key, e.scraped_text, e.screenshot_hash, e.scraped_text_hash,
			e.metadata_commitment, e.claims_root, e.evidence_commitment, e.flare_tx_hash,
			e.tee_certificate_hash, e.tee_signature, e.verification_status, e.captured_at, e.created_at
		FROM evidence e
		LEFT JOIN claims c ON c.evidence_id = e.id
		WHERE ($1 = '' OR c.text ILIKE '%' || $1 || '%' OR c.source_excerpt ILIKE '%' || $1 || '%' OR e.url ILIKE '%' || $1 || '%' OR e.title ILIKE '%' || $1 || '%' OR e.case_id ILIKE '%' || $1 || '%')
			AND ($2 = '' OR e.company ILIKE '%' || $2 || '%')
			AND ($3 = '' OR e.domain ILIKE '%' || $3 || '%')
			AND ($4 = '' OR e.verification_status = $4)
		ORDER BY e.created_at DESC
		LIMIT $5
	`, query, company, domain, status, params.Limit)
	if err != nil {
		return model.SearchResult{}, err
	}
	defer rows.Close()

	items := make([]model.Evidence, 0)
	ids := make([]string, 0)
	for rows.Next() {
		item, err := scanEvidence(rows)
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

func (p *Postgres) getEvidenceBase(ctx context.Context, id string) (model.Evidence, error) {
	row := p.pool.QueryRow(ctx, `
		SELECT id, url, domain, title, company, case_id, user_id, screenshot_data_url,
			screenshot_object_key, scraped_text, screenshot_hash, scraped_text_hash, metadata_commitment,
			claims_root, evidence_commitment, flare_tx_hash, tee_certificate_hash,
			tee_signature, verification_status, captured_at, created_at
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
		&item.CapturedAt,
		&item.CreatedAt,
	)
	return item, err
}

func containsFold(value string, query string) bool {
	return strings.Contains(strings.ToLower(value), strings.ToLower(query))
}
