package httpapi

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/docsnap/docsnap/services/api/internal/model"
)

func (s Server) prepareAnchor(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}

	item, err := s.store.GetEvidence(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "evidence not found")
		return
	}
	if item.PublishedBy != user.ID {
		writeError(w, http.StatusForbidden, "only the owner can anchor this evidence")
		return
	}
	if item.VerificationStatus != model.StatusPendingWalletAnchor {
		writeError(w, http.StatusConflict, "already anchored")
		return
	}

	calldata, err := s.flare.BuildSubmitCalldata(model.AnchorRequest{
		EvidenceID:         item.ID,
		EvidenceCommitment: item.EvidenceCommitment,
		ScreenshotHash:     item.ScreenshotHash,
		ScrapedTextHash:    item.ScrapedTextHash,
		MetadataCommitment: item.MetadataCommitment,
		ClaimsRoot:         item.ClaimsRoot,
		TEECertificateHash: item.TEECertificateHash,
	})
	if err != nil {
		log.Printf("prepareAnchor: build calldata failed: %v", err)
		writeError(w, http.StatusInternalServerError, "couldn't prepare transaction")
		return
	}
	writeJSON(w, http.StatusOK, calldata)
}

type confirmAnchorRequest struct {
	TxHash string `json:"txHash"`
}

func (s Server) confirmAnchor(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}

	item, err := s.store.GetEvidence(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "evidence not found")
		return
	}
	if item.PublishedBy != user.ID {
		writeError(w, http.StatusForbidden, "only the owner can anchor this evidence")
		return
	}
	if item.VerificationStatus != model.StatusPendingWalletAnchor {
		writeError(w, http.StatusConflict, "already anchored")
		return
	}

	var req confirmAnchorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TxHash == "" {
		writeError(w, http.StatusBadRequest, "txHash is required")
		return
	}

	result, submitter, err := s.flare.VerifySubmission(r.Context(), req.TxHash, model.AnchorRequest{
		EvidenceID:         item.ID,
		EvidenceCommitment: item.EvidenceCommitment,
		ScreenshotHash:     item.ScreenshotHash,
		ScrapedTextHash:    item.ScrapedTextHash,
		MetadataCommitment: item.MetadataCommitment,
		ClaimsRoot:         item.ClaimsRoot,
		TEECertificateHash: item.TEECertificateHash,
	})
	if err != nil {
		log.Printf("confirmAnchor: verification failed: %v", err)
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	if err := s.store.UpdateAnchor(r.Context(), item.ID, result, submitter); err != nil {
		log.Printf("confirmAnchor: save failed: %v", err)
		writeError(w, http.StatusInternalServerError, "save failed")
		return
	}

	item.FlareTxHash = result.TxHash
	item.TEECertificateHash = result.TEECertificateHash
	item.VerificationStatus = result.Status
	item.AnchorSubmitter = submitter
	writeJSON(w, http.StatusOK, item)
}
