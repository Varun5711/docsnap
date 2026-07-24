package httpapi

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/docsnap/docsnap/services/api/internal/ai"
	"github.com/docsnap/docsnap/services/api/internal/config"
	"github.com/docsnap/docsnap/services/api/internal/evidence"
	"github.com/docsnap/docsnap/services/api/internal/flare"
	"github.com/docsnap/docsnap/services/api/internal/model"
	"github.com/docsnap/docsnap/services/api/internal/storage"
	"github.com/docsnap/docsnap/services/api/internal/store"
	"github.com/docsnap/docsnap/services/api/internal/tee"
)

type Server struct {
	cfg       config.Config
	store     store.Repository
	extractor ai.Extractor
	hasher    evidence.Hasher
	flare     flare.Client
	storage   storage.Store
	tee       tee.Certifier
}

func NewServer(cfg config.Config, store store.Repository, extractor ai.Extractor, hasher evidence.Hasher, flare flare.Client, objectStore storage.Store, certifier tee.Certifier) Server {
	return Server{cfg: cfg, store: store, extractor: extractor, hasher: hasher, flare: flare, storage: objectStore, tee: certifier}
}

func (s Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("POST /api/captures", s.capture)
	mux.HandleFunc("GET /api/claims", s.search)
	mux.HandleFunc("GET /api/evidence/{id}/screenshot", s.getScreenshot)
	mux.HandleFunc("GET /api/evidence/", s.getEvidence)
	mux.HandleFunc("POST /api/verify", s.verify)
	return s.cors(mux)
}

func (s Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s Server) capture(w http.ResponseWriter, r *http.Request) {
	var req model.CaptureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	if strings.TrimSpace(req.URL) == "" {
		writeError(w, http.StatusBadRequest, "url is required")
		return
	}
	if strings.TrimSpace(req.ScreenshotDataURL) == "" && strings.TrimSpace(req.ScrapedText) == "" {
		writeError(w, http.StatusBadRequest, "screenshotDataUrl or scrapedText is required")
		return
	}
	if req.CapturedAt.IsZero() {
		req.CapturedAt = time.Now().UTC()
	}

	claims, err := s.extractor.Extract(req)
	if err != nil {
		writeError(w, http.StatusBadGateway, "claim extraction failed")
		return
	}

	id := "ev_" + randomHex(12)
	for i := range claims {
		claims[i].ID = "cl_" + randomHex(10)
		claims[i].EvidenceID = id
	}

	screenshotHash, textHash, metadataCommitment, claimsRoot, commitment, claims := s.hasher.Evidence(evidence.HashInput{
		URL:               req.URL,
		Company:           req.Company,
		CaseID:            req.CaseID,
		UserID:            req.UserID,
		ScreenshotDataURL: req.ScreenshotDataURL,
		ScrapedText:       req.ScrapedText,
		CapturedAt:        req.CapturedAt,
		Claims:            claims,
	})

	screenshotObjectKey := ""
	if strings.TrimSpace(req.ScreenshotDataURL) != "" {
		key, err := s.storage.PutDataURL(r.Context(), id, req.ScreenshotDataURL)
		if err != nil {
			writeError(w, http.StatusBadRequest, "screenshot storage failed")
			return
		}
		screenshotObjectKey = key
	}

	teeResult, err := s.tee.Certify(r.Context(), model.TEECertifyRequest{
		EvidenceID:         id,
		EvidenceCommitment: commitment,
		ScreenshotHash:     screenshotHash,
		ScrapedTextHash:    textHash,
		MetadataCommitment: metadataCommitment,
		ClaimsRoot:         claimsRoot,
		SubmittedAt:        req.CapturedAt.UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		writeError(w, http.StatusBadGateway, "tee certification failed")
		return
	}

	anchor, err := s.flare.Anchor(model.AnchorRequest{
		EvidenceID:         id,
		EvidenceCommitment: commitment,
		ScreenshotHash:     screenshotHash,
		ScrapedTextHash:    textHash,
		MetadataCommitment: metadataCommitment,
		ClaimsRoot:         claimsRoot,
		TEECertificateHash: teeResult.CertificateHash,
		Submitter:          req.UserID,
	})
	if err != nil {
		writeError(w, http.StatusBadGateway, "flare anchoring failed")
		return
	}

	item := model.Evidence{
		ID:                  id,
		URL:                 req.URL,
		Domain:              evidence.Domain(req.URL),
		Title:               req.Title,
		Company:             req.Company,
		CaseID:              req.CaseID,
		UserID:              req.UserID,
		ScreenshotObjectKey: screenshotObjectKey,
		ScreenshotDataURL:   req.ScreenshotDataURL,
		ScrapedText:         req.ScrapedText,
		ScreenshotHash:      screenshotHash,
		ScrapedTextHash:     textHash,
		MetadataCommitment:  metadataCommitment,
		ClaimsRoot:          claimsRoot,
		EvidenceCommitment:  commitment,
		FlareTxHash:         anchor.TxHash,
		TEECertificateHash:  anchor.TEECertificateHash,
		TEESignature:        teeResult.Signature,
		VerificationStatus:  anchor.Status,
		CapturedAt:          req.CapturedAt,
		CreatedAt:           time.Now().UTC(),
		Claims:              claims,
	}

	saveItem := item
	saveItem.ScreenshotDataURL = ""
	if err := s.store.Save(r.Context(), saveItem); err != nil {
		writeError(w, http.StatusInternalServerError, "save failed")
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s Server) search(w http.ResponseWriter, r *http.Request) {
	result, err := s.store.Search(r.Context(), store.SearchParams{
		Query:   r.URL.Query().Get("q"),
		Company: r.URL.Query().Get("company"),
		Domain:  r.URL.Query().Get("domain"),
		Status:  r.URL.Query().Get("status"),
		Limit:   100,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "search failed")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s Server) getEvidence(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/evidence/")
	item, err := s.store.GetEvidence(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "evidence not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "read failed")
		return
	}
	item.ScreenshotDataURL = ""
	writeJSON(w, http.StatusOK, item)
}

func (s Server) getScreenshot(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	item, err := s.store.GetEvidence(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "evidence not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "read failed")
		return
	}
	mediaType, dataURL, err := s.storage.ReadDataURL(r.Context(), item.ScreenshotObjectKey)
	if err != nil {
		writeError(w, http.StatusNotFound, "screenshot not found")
		return
	}
	parts := strings.SplitN(dataURL, ",", 2)
	if len(parts) != 2 {
		writeError(w, http.StatusInternalServerError, "screenshot decode failed")
		return
	}
	body, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		writeError(w, http.StatusInternalServerError, "screenshot decode failed")
		return
	}
	w.Header().Set("Content-Type", mediaType)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func (s Server) verify(w http.ResponseWriter, r *http.Request) {
	var req model.VerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	item, err := s.store.GetEvidence(r.Context(), req.EvidenceID)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "evidence not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "read failed")
		return
	}

	if req.ScreenshotDataURL == "" {
		req.ScreenshotDataURL = item.ScreenshotDataURL
	}
	if req.ScreenshotDataURL == "" && item.ScreenshotObjectKey != "" {
		_, dataURL, err := s.storage.ReadDataURL(r.Context(), item.ScreenshotObjectKey)
		if err != nil {
			writeError(w, http.StatusNotFound, "stored screenshot not found")
			return
		}
		req.ScreenshotDataURL = dataURL
	}
	if req.ScrapedText == "" {
		req.ScrapedText = item.ScrapedText
	}
	if len(req.Claims) == 0 {
		req.Claims = item.Claims
	}

	_, _, _, _, actual, _ := s.hasher.Evidence(evidence.HashInput{
		URL:               item.URL,
		Company:           item.Company,
		CaseID:            item.CaseID,
		UserID:            item.UserID,
		ScreenshotDataURL: req.ScreenshotDataURL,
		ScrapedText:       req.ScrapedText,
		CapturedAt:        item.CapturedAt,
		Claims:            req.Claims,
	})

	verified := actual == item.EvidenceCommitment
	status := "tampered"
	if verified {
		status = "verified"
	}

	writeJSON(w, http.StatusOK, model.VerifyResult{
		EvidenceID:         item.ID,
		Verified:           verified,
		ExpectedCommitment: item.EvidenceCommitment,
		ActualCommitment:   actual,
		Status:             status,
	})
}

func (s Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", s.cfg.AppOrigin)
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func randomHex(size int) string {
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format(time.RFC3339Nano)))[:size]
	}
	return hex.EncodeToString(bytes)
}
