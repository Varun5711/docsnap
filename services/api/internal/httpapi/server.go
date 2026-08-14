package httpapi

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/docsnap/docsnap/services/api/internal/ai"
	"github.com/docsnap/docsnap/services/api/internal/config"
	"github.com/docsnap/docsnap/services/api/internal/evidence"
	"github.com/docsnap/docsnap/services/api/internal/flare"
	"github.com/docsnap/docsnap/services/api/internal/model"
	"github.com/docsnap/docsnap/services/api/internal/search"
	"github.com/docsnap/docsnap/services/api/internal/storage"
	"github.com/docsnap/docsnap/services/api/internal/store"
	"github.com/docsnap/docsnap/services/api/internal/tee"
	"github.com/docsnap/docsnap/services/api/internal/verdict"
)

type Server struct {
	cfg            config.Config
	store          store.Repository
	extractor      ai.Extractor
	hasher         evidence.Hasher
	flare          flare.Client
	storage        storage.Store
	tee            tee.Certifier
	searchProvider search.Provider
	verdictGen     verdict.Generator
}

func NewServer(cfg config.Config, store store.Repository, extractor ai.Extractor, hasher evidence.Hasher, flare flare.Client, objectStore storage.Store, certifier tee.Certifier, searchProvider search.Provider, verdictGenerator verdict.Generator) Server {
	return Server{cfg: cfg, store: store, extractor: extractor, hasher: hasher, flare: flare, storage: objectStore, tee: certifier, searchProvider: searchProvider, verdictGen: verdictGenerator}
}

func (s Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("POST /api/captures", s.capture)
	mux.HandleFunc("GET /api/claims", s.search)
	mux.HandleFunc("GET /api/evidence/{id}/screenshot", s.getScreenshot)
	mux.HandleFunc("GET /api/evidence/", s.getEvidence)
	mux.HandleFunc("POST /api/verify", s.verify)
	mux.HandleFunc("POST /api/claims/{id}/investigate", s.investigate)
	mux.HandleFunc("GET /api/investigations/{id}", s.getInvestigation)
	mux.HandleFunc("GET /api/proof/{id}", s.getProof)

	mux.HandleFunc("POST /api/auth/signup", s.signup)
	mux.HandleFunc("POST /api/auth/login", s.login)
	mux.HandleFunc("POST /api/auth/logout", s.logout)
	mux.HandleFunc("GET /api/auth/me", s.me)

	mux.HandleFunc("GET /api/discover", s.discover)
	mux.HandleFunc("GET /api/search", s.searchCanonicalClaims)
	mux.HandleFunc("GET /api/claims/similar", s.similarClaims)
	mux.HandleFunc("GET /api/claim/{slug}", s.getCanonicalClaim)
	mux.HandleFunc("POST /api/claims/{id}/evidence", s.addEvidence)
	mux.HandleFunc("POST /api/contributions/{id}/report", s.reportContribution)
	mux.HandleFunc("POST /api/claims/{id}/fork", s.forkClaim)
	mux.HandleFunc("POST /api/claims/{id}/publish", s.publishClaim)

	mux.HandleFunc("POST /api/evidence/{id}/anchor/prepare", s.prepareAnchor)
	mux.HandleFunc("POST /api/evidence/{id}/anchor/confirm", s.confirmAnchor)

	mux.HandleFunc("GET /api/domain/{domain}/trust", s.domainTrust)

	return s.cors(s.auth(mux))
}

var publicPathPrefixes = []string{"/api/proof/", "/api/investigations/", "/api/discover", "/api/search", "/api/claim/", "/api/claims/similar", "/api/domain/"}

func (s Server) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.cfg.APIKey == "" || r.Method == http.MethodOptions || r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}
		for _, prefix := range publicPathPrefixes {
			if strings.HasPrefix(r.URL.Path, prefix) {
				next.ServeHTTP(w, r)
				return
			}
		}
		if strings.HasPrefix(r.URL.Path, "/api/auth/") {
			next.ServeHTTP(w, r)
			return
		}

		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if subtle.ConstantTimeCompare([]byte(token), []byte(s.cfg.APIKey)) == 1 {
			next.ServeHTTP(w, r)
			return
		}
		if token != "" {
			if _, err := s.store.GetSessionUser(r.Context(), token); err == nil {
				next.ServeHTTP(w, r)
				return
			}
		}
		writeError(w, http.StatusUnauthorized, "unauthorized")
	})
}

func (s Server) resolveViewer(r *http.Request) (fullAccess bool, viewerID string) {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if token == "" {
		return false, ""
	}
	if s.cfg.APIKey != "" && subtle.ConstantTimeCompare([]byte(token), []byte(s.cfg.APIKey)) == 1 {
		return true, ""
	}
	user, err := s.store.GetSessionUser(r.Context(), token)
	if err != nil {
		return false, ""
	}
	return false, user.ID
}

func canView(claim model.Claim, fullAccess bool, viewerID string) bool {
	if fullAccess || claim.Visibility != "private" {
		return true
	}
	return claim.PublishedBy != "" && claim.PublishedBy == viewerID
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
		log.Printf("capture: claim extraction failed: %v", err)
		writeError(w, http.StatusBadGateway, "claim extraction failed")
		return
	}
	claims = dropOversizedClaims(claims)

	_, ownerID := s.resolveViewer(r)
	visibility := "private"
	if ownerID == "" {
		visibility = "unlisted"
	}

	id := "ev_" + randomHex(12)
	for i := range claims {
		claims[i].ID = "cl_" + randomHex(10)
		claims[i].EvidenceID = id
		claims[i].PublishedBy = ownerID
		claims[i].Visibility = visibility
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
			log.Printf("capture: screenshot storage failed: %v", err)
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
		log.Printf("capture: tee certification failed: %v", err)
		writeError(w, http.StatusBadGateway, "tee certification failed")
		return
	}

	var anchor model.AnchorResult
	if ownerID != "" {
		anchor = model.AnchorResult{TEECertificateHash: teeResult.CertificateHash, Status: model.StatusPendingWalletAnchor}
	} else {
		anchor, err = s.flare.Anchor(model.AnchorRequest{
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
			log.Printf("capture: flare anchoring failed: %v", err)
			writeError(w, http.StatusBadGateway, "flare anchoring failed")
			return
		}
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
		PublishedBy:         ownerID,
		CapturedAt:          req.CapturedAt,
		CreatedAt:           time.Now().UTC(),
		Claims:              claims,
	}

	saveItem := item
	saveItem.ScreenshotDataURL = ""
	if err := s.store.Save(r.Context(), saveItem); err != nil {
		log.Printf("capture: save failed: %v", err)
		writeError(w, http.StatusInternalServerError, "save failed")
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s Server) search(w http.ResponseWriter, r *http.Request) {

	_, ownerID := s.resolveViewer(r)
	result, err := s.store.Search(r.Context(), store.SearchParams{
		Query:   r.URL.Query().Get("q"),
		Company: r.URL.Query().Get("company"),
		Domain:  r.URL.Query().Get("domain"),
		Status:  r.URL.Query().Get("status"),
		Owner:   ownerID,
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

	pureCheck := req.ScreenshotDataURL == "" && req.ScrapedText == "" && len(req.Claims) == 0

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

	if pureCheck {
		if err := s.store.UpdateVerificationStatus(r.Context(), item.ID, status); err != nil {
			log.Printf("verify: persisting status failed: %v", err)
		}
	}

	writeJSON(w, http.StatusOK, model.VerifyResult{
		EvidenceID:         item.ID,
		Verified:           verified,
		ExpectedCommitment: item.EvidenceCommitment,
		ActualCommitment:   actual,
		Status:             status,
	})
}

func (s Server) investigate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	claim, err := s.store.GetClaim(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "claim not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "read failed")
		return
	}

	results, err := s.searchProvider.Search(r.Context(), claim.Text)
	if err != nil {
		log.Printf("investigate: search failed: %v", err)
		writeError(w, http.StatusBadGateway, "evidence search failed")
		return
	}

	v, err := s.verdictGen.Generate(r.Context(), claim.Text, results)
	if err != nil {
		log.Printf("investigate: verdict generation failed: %v", err)
		writeError(w, http.StatusBadGateway, "verdict generation failed")
		return
	}

	if err := s.store.SaveInvestigation(r.Context(), id, store.Investigation{
		Status:     v.Status,
		Confidence: v.Confidence,
		Reasoning:  v.Reasoning,
		Sources:    v.Sources,
	}); err != nil {
		log.Printf("investigate: save failed: %v", err)
		writeError(w, http.StatusInternalServerError, "save failed")
		return
	}

	updated, err := s.store.GetClaim(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "read failed")
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s Server) getInvestigation(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	claim, err := s.store.GetClaim(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "claim not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "read failed")
		return
	}
	if fullAccess, viewerID := s.resolveViewer(r); !canView(claim, fullAccess, viewerID) {
		writeError(w, http.StatusNotFound, "claim not found")
		return
	}
	ev, err := s.store.GetEvidence(r.Context(), claim.EvidenceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "read failed")
		return
	}
	ev.ScreenshotDataURL = ""
	// Best-effort attribution — a fork should say whose investigation it
	// built on, not just "an earlier investigation". Never fails the
	// request over a display nicety: if the parent or its owner can't be
	// resolved, ForkedFromOwnerName just stays blank.
	if claim.ForkedFromClaimID != "" {
		if parent, err := s.store.GetClaim(r.Context(), claim.ForkedFromClaimID); err == nil && parent.PublishedBy != "" {
			if owner, err := s.store.GetUserByID(r.Context(), parent.PublishedBy); err == nil {
				claim.ForkedFromOwnerName = owner.DisplayName
			}
		}
	}
	writeJSON(w, http.StatusOK, model.Investigation{Claim: claim, Evidence: ev})
}

func (s Server) getProof(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ev, err := s.store.GetEvidence(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "evidence not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "read failed")
		return
	}
	writeJSON(w, http.StatusOK, model.Proof{
		EvidenceID:         ev.ID,
		URL:                ev.URL,
		ScreenshotHash:     ev.ScreenshotHash,
		ScrapedTextHash:    ev.ScrapedTextHash,
		MetadataCommitment: ev.MetadataCommitment,
		ClaimsRoot:         ev.ClaimsRoot,
		EvidenceCommitment: ev.EvidenceCommitment,
		FlareTxHash:        ev.FlareTxHash,
		TEECertificateHash: ev.TEECertificateHash,
		VerificationStatus: ev.VerificationStatus,
		CapturedAt:         ev.CapturedAt,
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

const maxClaimTextLength = 240

func dropOversizedClaims(claims []model.Claim) []model.Claim {
	kept := claims[:0]
	for _, claim := range claims {
		if len(claim.Text) > maxClaimTextLength {
			continue
		}
		kept = append(kept, claim)
	}
	return kept
}

func randomHex(size int) string {
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format(time.RFC3339Nano)))[:size]
	}
	return hex.EncodeToString(bytes)
}
