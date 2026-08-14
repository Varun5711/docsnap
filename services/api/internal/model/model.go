package model

import "time"

type CaptureRequest struct {
	URL               string    `json:"url"`
	Title             string    `json:"title"`
	Company           string    `json:"company"`
	CaseID            string    `json:"caseId"`
	UserID            string    `json:"userId"`
	ScreenshotDataURL string    `json:"screenshotDataUrl"`
	ScrapedText       string    `json:"scrapedText"`
	CapturedAt        time.Time `json:"capturedAt"`
}

type Evidence struct {
	ID                  string    `json:"id"`
	URL                 string    `json:"url"`
	Domain              string    `json:"domain"`
	Title               string    `json:"title"`
	Company             string    `json:"company"`
	CaseID              string    `json:"caseId"`
	UserID              string    `json:"userId"`
	ScreenshotObjectKey string    `json:"screenshotObjectKey"`
	ScreenshotDataURL   string    `json:"screenshotDataUrl"`
	ScrapedText         string    `json:"scrapedText"`
	ScreenshotHash      string    `json:"screenshotHash"`
	ScrapedTextHash     string    `json:"scrapedTextHash"`
	MetadataCommitment  string    `json:"metadataCommitment"`
	ClaimsRoot          string    `json:"claimsRoot"`
	EvidenceCommitment  string    `json:"evidenceCommitment"`
	FlareTxHash         string    `json:"flareTxHash"`
	TEECertificateHash  string    `json:"teeCertificateHash"`
	TEESignature        string    `json:"teeSignature"`
	VerificationStatus  string    `json:"verificationStatus"`
	PublishedBy         string    `json:"publishedBy"`
	AnchorSubmitter     string    `json:"anchorSubmitter"`
	CapturedAt          time.Time `json:"capturedAt"`
	CreatedAt           time.Time `json:"createdAt"`
	Claims              []Claim   `json:"claims"`
}

const StatusPendingWalletAnchor = "pending_wallet_anchor"

type SubmitCalldata struct {
	To      string `json:"to"`
	Data    string `json:"data"`
	ChainID int64  `json:"chainId"`
}

type DomainTrust struct {
	Domain            string  `json:"domain"`
	TotalInvestigated int     `json:"totalInvestigated"`
	Contradicted      int     `json:"contradicted"`
	Supported         int     `json:"supported"`
	FalseRatio        float64 `json:"falseRatio"`
	Label             string  `json:"label"`
}

const (
	DomainTrustLowTrust     = "low_trust"
	DomainTrustInconsistent = "inconsistent"
	DomainTrustNone         = "none"

	domainTrustMinSample = 3

	domainTrustFalseRatioThreshold = 0.5
)

func NewDomainTrust(domain string, total, contradicted, supported int) DomainTrust {
	dt := DomainTrust{Domain: domain, TotalInvestigated: total, Contradicted: contradicted, Supported: supported, Label: DomainTrustNone}
	if total < domainTrustMinSample {
		return dt
	}
	dt.FalseRatio = float64(contradicted) / float64(total)
	switch {
	case dt.FalseRatio > domainTrustFalseRatioThreshold:
		dt.Label = DomainTrustLowTrust
	case supported > 0 && contradicted > 0:
		dt.Label = DomainTrustInconsistent
	}
	return dt
}

type Claim struct {
	ID            string  `json:"id"`
	EvidenceID    string  `json:"evidenceId"`
	Text          string  `json:"text"`
	Type          string  `json:"type"`
	Confidence    float64 `json:"confidence"`
	SourceExcerpt string  `json:"sourceExcerpt"`
	Hash          string  `json:"hash"`
	Status        string  `json:"status"`

	Subject   string   `json:"subject,omitempty"`
	Predicate string   `json:"predicate,omitempty"`
	Object    string   `json:"object,omitempty"`
	ClaimDate string   `json:"claimDate,omitempty"`
	Location  string   `json:"location,omitempty"`
	Entities  []string `json:"entities,omitempty"`

	InvestigationStatus     string          `json:"investigationStatus,omitempty"`
	InvestigationConfidence float64         `json:"investigationConfidence,omitempty"`
	Reasoning               *ClaimReasoning `json:"reasoning,omitempty"`
	InvestigatedAt          *time.Time      `json:"investigatedAt,omitempty"`
	Sources                 []Source        `json:"sources,omitempty"`

	CanonicalClaimID   string `json:"canonicalClaimId,omitempty"`
	CanonicalClaimSlug string `json:"canonicalClaimSlug,omitempty"`
	Visibility         string `json:"visibility,omitempty"`
	PublishedBy        string `json:"publishedBy,omitempty"`
	ForkedFromClaimID  string `json:"forkedFromClaimId,omitempty"`
	// ForkedFromOwnerName is resolved server-side (getInvestigation) purely
	// for display — who published the investigation this was built on.
	ForkedFromOwnerName string                 `json:"forkedFromOwnerName,omitempty"`
	Contributions       []EvidenceContribution `json:"contributions,omitempty"`
}

type EvidenceContribution struct {
	ID            string `json:"id"`
	ClaimID       string `json:"claimId"`
	ContributorID string `json:"contributorId"`
	// ContributorName is resolved server-side (join on users) — accountability
	// only means something if the reader can actually see whose name is on it.
	ContributorName string `json:"contributorName,omitempty"`
	Type            string `json:"type"`
	URL             string `json:"url"`
	Note            string `json:"note"`
	// Flagged is true once independent reports cross a minimum-sample
	// threshold — same "don't react to one report" guard as domain trust.
	// The raw count is deliberately not exposed: it's a signal to read the
	// contribution skeptically, not a vote tally to game.
	Flagged   bool      `json:"flagged"`
	CreatedAt time.Time `json:"createdAt"`
}

type CanonicalClaim struct {
	ID        string    `json:"id"`
	Slug      string    `json:"slug"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"createdAt"`
	Claims    []Claim   `json:"claims"`
}

type User struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"displayName"`
	CreatedAt   time.Time `json:"createdAt"`
}

type ClaimReasoning struct {
	Knowns    []string `json:"knowns"`
	Unknowns  []string `json:"unknowns"`
	Conflicts []string `json:"conflicts"`
}

type Source struct {
	ID           string    `json:"id"`
	ClaimID      string    `json:"claimId"`
	URL          string    `json:"url"`
	Name         string    `json:"name"`
	SourceType   string    `json:"sourceType"`
	StarRating   int       `json:"starRating"`
	Relationship string    `json:"relationship"`
	Relevance    float64   `json:"relevance"`
	CapturedAt   time.Time `json:"capturedAt"`
}

type Investigation struct {
	Claim    Claim    `json:"claim"`
	Evidence Evidence `json:"evidence"`
}

type Proof struct {
	EvidenceID         string    `json:"evidenceId"`
	URL                string    `json:"url"`
	ScreenshotHash     string    `json:"screenshotHash"`
	ScrapedTextHash    string    `json:"scrapedTextHash"`
	MetadataCommitment string    `json:"metadataCommitment"`
	ClaimsRoot         string    `json:"claimsRoot"`
	EvidenceCommitment string    `json:"evidenceCommitment"`
	FlareTxHash        string    `json:"flareTxHash"`
	TEECertificateHash string    `json:"teeCertificateHash"`
	VerificationStatus string    `json:"verificationStatus"`
	CapturedAt         time.Time `json:"capturedAt"`
	// ScreenshotDataURL: the proof page is already fully public and
	// unauthenticated once someone has the link — a hash-only page is only
	// useful to someone who already has an independent copy to compare
	// against, which defeats "share this as evidence". Best-effort: left
	// blank if the screenshot can't be read, never fails the whole request.
	ScreenshotDataURL string `json:"screenshotDataUrl,omitempty"`
}

type AnchorRequest struct {
	EvidenceID         string
	EvidenceCommitment string
	ScreenshotHash     string
	ScrapedTextHash    string
	MetadataCommitment string
	ClaimsRoot         string
	TEECertificateHash string
	Submitter          string
}

type AnchorResult struct {
	TxHash             string `json:"txHash"`
	TEECertificateHash string `json:"teeCertificateHash"`
	Status             string `json:"status"`
}

type TEECertifyRequest struct {
	EvidenceID         string `json:"evidenceId"`
	EvidenceCommitment string `json:"evidenceCommitment"`
	ScreenshotHash     string `json:"screenshotHash"`
	ScrapedTextHash    string `json:"scrapedTextHash"`
	MetadataCommitment string `json:"metadataCommitment"`
	ClaimsRoot         string `json:"claimsRoot"`
	SubmittedAt        string `json:"submittedAt"`
}

type TEECertifyResult struct {
	EvidenceID         string `json:"evidenceId"`
	Accepted           bool   `json:"accepted"`
	CertificateHash    string `json:"certificateHash"`
	Signature          string `json:"signature"`
	PublicKey          string `json:"publicKey"`
	EvidenceCommitment string `json:"evidenceCommitment"`
	MetadataCommitment string `json:"metadataCommitment"`
	ClaimsRoot         string `json:"claimsRoot"`
	CertifiedAt        string `json:"certifiedAt"`
}

type SearchResult struct {
	Claims []Claim    `json:"claims"`
	Items  []Evidence `json:"items"`
}

type VerifyRequest struct {
	EvidenceID        string  `json:"evidenceId"`
	ScreenshotDataURL string  `json:"screenshotDataUrl"`
	ScrapedText       string  `json:"scrapedText"`
	Claims            []Claim `json:"claims"`
}

type VerifyResult struct {
	EvidenceID         string `json:"evidenceId"`
	Verified           bool   `json:"verified"`
	ExpectedCommitment string `json:"expectedCommitment"`
	ActualCommitment   string `json:"actualCommitment"`
	Status             string `json:"status"`
}
