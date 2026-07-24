package extension

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

type CertifyRequest struct {
	EvidenceID         string `json:"evidenceId"`
	EvidenceCommitment string `json:"evidenceCommitment"`
	ScreenshotHash     string `json:"screenshotHash"`
	ScrapedTextHash    string `json:"scrapedTextHash"`
	MetadataCommitment string `json:"metadataCommitment"`
	ClaimsRoot         string `json:"claimsRoot"`
	SubmittedAt        string `json:"submittedAt"`
}

type CertifyResult struct {
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

func Certify(req CertifyRequest) (CertifyResult, error) {
	if strings.TrimSpace(req.EvidenceID) == "" {
		return CertifyResult{}, errors.New("evidence id is required")
	}
	if !isHash(req.EvidenceCommitment) || !isHash(req.ScreenshotHash) || !isHash(req.ScrapedTextHash) || !isHash(req.MetadataCommitment) || !isHash(req.ClaimsRoot) {
		return CertifyResult{}, errors.New("invalid commitment")
	}

	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return CertifyResult{}, err
	}

	certifiedAt := time.Now().UTC().Format(time.RFC3339Nano)
	payload := strings.Join([]string{
		req.EvidenceID,
		req.EvidenceCommitment,
		req.ScreenshotHash,
		req.ScrapedTextHash,
		req.MetadataCommitment,
		req.ClaimsRoot,
		certifiedAt,
	}, "|")
	hash := digest(payload)
	signature := ed25519.Sign(privateKey, []byte(hash))

	return CertifyResult{
		EvidenceID:         req.EvidenceID,
		Accepted:           true,
		CertificateHash:    hash,
		Signature:          hex.EncodeToString(signature),
		PublicKey:          hex.EncodeToString(publicKey),
		EvidenceCommitment: req.EvidenceCommitment,
		MetadataCommitment: req.MetadataCommitment,
		ClaimsRoot:         req.ClaimsRoot,
		CertifiedAt:        certifiedAt,
	}, nil
}

func isHash(value string) bool {
	value = strings.TrimPrefix(value, "0x")
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
