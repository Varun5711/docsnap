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
	CapturedAt          time.Time `json:"capturedAt"`
	CreatedAt           time.Time `json:"createdAt"`
	Claims              []Claim   `json:"claims"`
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
