package tee

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/docsnap/docsnap/services/api/internal/model"
)

type Certifier interface {
	Certify(ctx context.Context, req model.TEECertifyRequest) (model.TEECertifyResult, error)
}

type LocalCertifier struct{}

func NewLocalCertifier() LocalCertifier {
	return LocalCertifier{}
}

func (LocalCertifier) Certify(ctx context.Context, req model.TEECertifyRequest) (model.TEECertifyResult, error) {
	select {
	case <-ctx.Done():
		return model.TEECertifyResult{}, ctx.Err()
	default:
	}
	if req.EvidenceID == "" || !isHash(req.EvidenceCommitment) || !isHash(req.ScreenshotHash) || !isHash(req.ScrapedTextHash) || !isHash(req.MetadataCommitment) || !isHash(req.ClaimsRoot) {
		return model.TEECertifyResult{}, errors.New("invalid tee request")
	}
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return model.TEECertifyResult{}, err
	}
	certifiedAt := time.Now().UTC().Format(time.RFC3339Nano)
	payload := strings.Join([]string{req.EvidenceID, req.EvidenceCommitment, req.ScreenshotHash, req.ScrapedTextHash, req.MetadataCommitment, req.ClaimsRoot, certifiedAt}, "|")
	hash := digest(payload)
	signature := ed25519.Sign(privateKey, []byte(hash))
	return model.TEECertifyResult{
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

type HTTPClient struct {
	url    string
	client *http.Client
}

func NewHTTPClient(url string) HTTPClient {
	return HTTPClient{url: strings.TrimRight(url, "/") + "/certify", client: &http.Client{Timeout: 30 * time.Second}}
}

func (h HTTPClient) Certify(ctx context.Context, req model.TEECertifyRequest) (model.TEECertifyResult, error) {
	var buffer bytes.Buffer
	if err := json.NewEncoder(&buffer).Encode(req); err != nil {
		return model.TEECertifyResult{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, h.url, &buffer)
	if err != nil {
		return model.TEECertifyResult{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(httpReq)
	if err != nil {
		return model.TEECertifyResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return model.TEECertifyResult{}, errors.New("tee certification failed")
	}
	var result model.TEECertifyResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return model.TEECertifyResult{}, err
	}
	if !result.Accepted || !isHash(result.CertificateHash) {
		return model.TEECertifyResult{}, errors.New("tee rejected evidence")
	}
	return result, nil
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
