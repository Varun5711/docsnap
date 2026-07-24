package flare

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/docsnap/docsnap/services/api/internal/model"
)

type Client interface {
	Anchor(req model.AnchorRequest) (model.AnchorResult, error)
}

type SimulatedClient struct{}

func NewSimulatedClient() SimulatedClient {
	return SimulatedClient{}
}

func (SimulatedClient) Anchor(req model.AnchorRequest) (model.AnchorResult, error) {
	seed := req.EvidenceID + "|" + req.EvidenceCommitment + "|" + req.ClaimsRoot + "|" + time.Now().UTC().Format(time.RFC3339Nano)
	return model.AnchorResult{
		TxHash:             "0x" + digest(seed+"|tx"),
		TEECertificateHash: "0x" + digest(seed+"|tee-cert"),
		TEESignature:       "0x" + digest(seed+"|tee-signature") + digest(seed+"|tee-signature-2"),
		Status:             "certified",
	}, nil
}

func digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

