package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"strings"
	"time"

	"github.com/docsnap/docsnap/services/api/internal/ai"
	"github.com/docsnap/docsnap/services/api/internal/config"
	"github.com/docsnap/docsnap/services/api/internal/evidence"
	"github.com/docsnap/docsnap/services/api/internal/flare"
	"github.com/docsnap/docsnap/services/api/internal/model"
	"github.com/docsnap/docsnap/services/api/internal/search"
	"github.com/docsnap/docsnap/services/api/internal/store"
	"github.com/docsnap/docsnap/services/api/internal/tee"
	"github.com/docsnap/docsnap/services/api/internal/verdict"
)

type memoryRepo struct {
	items           map[string]model.Evidence
	users           map[string]model.User
	usersByEmail    map[string]string
	passwords       map[string]string
	sessions        map[string]string
	canonical       map[string]model.CanonicalClaim
	canonicalBySlug map[string]string
	contributions   map[string][]model.EvidenceContribution
}

func newMemoryRepo() *memoryRepo {
	return &memoryRepo{
		items:           map[string]model.Evidence{},
		users:           map[string]model.User{},
		usersByEmail:    map[string]string{},
		passwords:       map[string]string{},
		sessions:        map[string]string{},
		canonical:       map[string]model.CanonicalClaim{},
		canonicalBySlug: map[string]string{},
		contributions:   map[string][]model.EvidenceContribution{},
	}
}

func (m *memoryRepo) Save(ctx context.Context, item model.Evidence) error {
	m.items[item.ID] = item
	return nil
}

func (m *memoryRepo) GetEvidence(ctx context.Context, id string) (model.Evidence, error) {
	item, ok := m.items[id]
	if !ok {
		return model.Evidence{}, store.ErrNotFound
	}
	return item, nil
}

func (m *memoryRepo) Search(ctx context.Context, params store.SearchParams) (model.SearchResult, error) {
	items := make([]model.Evidence, 0, len(m.items))
	for _, item := range m.items {
		if params.Owner != "" && item.PublishedBy != params.Owner {
			continue
		}
		items = append(items, item)
	}
	return model.SearchResult{Items: items}, nil
}

func (m *memoryRepo) UpdateVerificationStatus(ctx context.Context, id string, status string) error {
	item, ok := m.items[id]
	if !ok {
		return store.ErrNotFound
	}
	item.VerificationStatus = status
	m.items[id] = item
	return nil
}

func (m *memoryRepo) DomainTrust(ctx context.Context, domain string) (model.DomainTrust, error) {
	var total, contradicted, supported int
	for _, item := range m.items {
		if item.Domain != domain {
			continue
		}
		for _, claim := range item.Claims {
			if claim.InvestigationStatus == "" {
				continue
			}
			total++
			switch claim.InvestigationStatus {
			case "CONTRADICTED", "LIKELY_CONTRADICTED":
				contradicted++
			case "SUPPORTED", "LIKELY_SUPPORTED":
				supported++
			}
		}
	}
	return model.NewDomainTrust(domain, total, contradicted, supported), nil
}

func (m *memoryRepo) UpdateAnchor(ctx context.Context, id string, result model.AnchorResult, submitter string) error {
	item, ok := m.items[id]
	if !ok {
		return store.ErrNotFound
	}
	item.FlareTxHash = result.TxHash
	item.TEECertificateHash = result.TEECertificateHash
	item.VerificationStatus = result.Status
	item.AnchorSubmitter = submitter
	m.items[id] = item
	return nil
}

func (m *memoryRepo) GetClaim(ctx context.Context, id string) (model.Claim, error) {
	for _, item := range m.items {
		for _, claim := range item.Claims {
			if claim.ID == id {
				claim.Contributions = m.contributions[id]
				return claim, nil
			}
		}
	}
	return model.Claim{}, store.ErrNotFound
}

func (m *memoryRepo) SaveInvestigation(ctx context.Context, claimID string, v store.Investigation) error {
	for evidenceID, item := range m.items {
		for i, claim := range item.Claims {
			if claim.ID != claimID {
				continue
			}
			claim.InvestigationStatus = v.Status
			claim.InvestigationConfidence = v.Confidence
			claim.Reasoning = &v.Reasoning
			claim.Sources = v.Sources
			item.Claims[i] = claim
			m.items[evidenceID] = item
			return nil
		}
	}
	return store.ErrNotFound
}

func (m *memoryRepo) findClaim(id string) (string, model.Claim, bool) {
	for evidenceID, item := range m.items {
		for _, claim := range item.Claims {
			if claim.ID == id {
				return evidenceID, claim, true
			}
		}
	}
	return "", model.Claim{}, false
}

func (m *memoryRepo) putClaim(evidenceID string, claim model.Claim) {
	item := m.items[evidenceID]
	for i, c := range item.Claims {
		if c.ID == claim.ID {
			item.Claims[i] = claim
			m.items[evidenceID] = item
			return
		}
	}
	item.Claims = append(item.Claims, claim)
	m.items[evidenceID] = item
}

func (m *memoryRepo) CreateUser(ctx context.Context, user model.User, passwordHash string) error {
	if _, exists := m.usersByEmail[user.Email]; exists {
		return errors.New("email taken")
	}
	m.users[user.ID] = user
	m.usersByEmail[user.Email] = user.ID
	m.passwords[user.ID] = passwordHash
	return nil
}

func (m *memoryRepo) GetUserByEmail(ctx context.Context, email string) (model.User, string, error) {
	id, ok := m.usersByEmail[email]
	if !ok {
		return model.User{}, "", store.ErrNotFound
	}
	return m.users[id], m.passwords[id], nil
}

func (m *memoryRepo) GetUserByID(ctx context.Context, id string) (model.User, error) {
	user, ok := m.users[id]
	if !ok {
		return model.User{}, store.ErrNotFound
	}
	return user, nil
}

func (m *memoryRepo) CreateSession(ctx context.Context, token, userID string, expiresAt time.Time) error {
	m.sessions[token] = userID
	return nil
}

func (m *memoryRepo) GetSessionUser(ctx context.Context, token string) (model.User, error) {
	userID, ok := m.sessions[token]
	if !ok {
		return model.User{}, store.ErrNotFound
	}
	return m.GetUserByID(ctx, userID)
}

func (m *memoryRepo) DeleteSession(ctx context.Context, token string) error {
	delete(m.sessions, token)
	return nil
}

func (m *memoryRepo) FindSimilarCanonicalClaims(ctx context.Context, text string, limit int) ([]model.CanonicalClaim, error) {
	results := make([]model.CanonicalClaim, 0)
	for _, cc := range m.canonical {
		if strings.Contains(strings.ToLower(cc.Text), strings.ToLower(text)) || strings.Contains(strings.ToLower(text), strings.ToLower(cc.Text)) {
			cc.Claims = m.claimsForCanonical(cc.ID)
			results = append(results, cc)
		}
	}
	return results, nil
}

func (m *memoryRepo) claimsForCanonical(canonicalID string) []model.Claim {
	claims := make([]model.Claim, 0)
	for _, item := range m.items {
		for _, c := range item.Claims {
			if c.CanonicalClaimID == canonicalID {
				claims = append(claims, c)
			}
		}
	}
	return claims
}

func (m *memoryRepo) CreateCanonicalClaim(ctx context.Context, cc model.CanonicalClaim) error {
	m.canonical[cc.ID] = cc
	m.canonicalBySlug[cc.Slug] = cc.ID
	return nil
}

func (m *memoryRepo) GetCanonicalClaimBySlug(ctx context.Context, slug string) (model.CanonicalClaim, error) {
	id, ok := m.canonicalBySlug[slug]
	if !ok {
		return model.CanonicalClaim{}, store.ErrNotFound
	}
	cc := m.canonical[id]
	cc.Claims = m.claimsForCanonical(id)
	return cc, nil
}

func (m *memoryRepo) PublishClaim(ctx context.Context, claimID, canonicalClaimID, visibility, publishedBy string) error {
	evidenceID, claim, ok := m.findClaim(claimID)
	if !ok {
		return store.ErrNotFound
	}
	claim.CanonicalClaimID = canonicalClaimID
	claim.Visibility = visibility
	claim.PublishedBy = publishedBy
	m.putClaim(evidenceID, claim)
	return nil
}

func (m *memoryRepo) ForkClaim(ctx context.Context, parentID, newClaimID, ownerID string) (model.Claim, error) {
	evidenceID, parent, ok := m.findClaim(parentID)
	if !ok {
		return model.Claim{}, store.ErrNotFound
	}
	forked := parent
	forked.ID = newClaimID
	forked.Visibility = "private"
	forked.PublishedBy = ownerID
	forked.ForkedFromClaimID = parentID
	m.putClaim(evidenceID, forked)
	return forked, nil
}

func (m *memoryRepo) AddEvidenceContribution(ctx context.Context, contribution model.EvidenceContribution) error {
	m.contributions[contribution.ClaimID] = append(m.contributions[contribution.ClaimID], contribution)
	return nil
}

func (m *memoryRepo) Discover(ctx context.Context) ([]model.Claim, []model.Claim, error) {
	public := make([]model.Claim, 0)
	for _, item := range m.items {
		for _, c := range item.Claims {
			if c.Visibility == "public" {
				public = append(public, c)
			}
		}
	}
	return public, public, nil
}

type memoryStore struct {
	objects map[string]string
}

func newMemoryStore() *memoryStore {
	return &memoryStore{objects: map[string]string{}}
}

func (m *memoryStore) PutDataURL(ctx context.Context, evidenceID string, dataURL string) (string, error) {
	key := evidenceID + ".png"
	m.objects[key] = dataURL
	return key, nil
}

func (m *memoryStore) ReadDataURL(ctx context.Context, key string) (string, string, error) {
	dataURL, ok := m.objects[key]
	if !ok {
		return "", "", errors.New("not found")
	}
	return "image/png", dataURL, nil
}

func newTestServer() Server {
	return newTestServerWithKey("")
}

func newTestServerWithKey(apiKey string) Server {
	return NewServer(
		config.Config{AppOrigin: "http://localhost:3000", APIKey: apiKey},
		newMemoryRepo(),
		ai.NewRuleExtractor(),
		evidence.NewHasher(),
		flare.NewSimulatedClient(),
		newMemoryStore(),
		tee.NewLocalCertifier(),
		search.NewTavilyProvider(""),
		verdict.NewGenerator("", "", ""),
	)
}

func doRequest(t *testing.T, handler http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	return doAuthedRequest(t, handler, method, path, body, "")
}

func doAuthedRequest(t *testing.T, handler http.Handler, method, path string, body any, authHeader string) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestCaptureThenVerifyRoundTrip(t *testing.T) {
	handler := newTestServer().Routes()

	rec := doRequest(t, handler, http.MethodPost, "/api/captures", map[string]any{
		"url":         "https://example.com/pricing",
		"title":       "Pricing",
		"company":     "ExampleCo",
		"caseId":      "CASE-1",
		"userId":      "tester",
		"scrapedText": "Our plan costs $99 per month and is fully SOC 2 compliant.",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("capture status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var captured model.Evidence
	if err := json.Unmarshal(rec.Body.Bytes(), &captured); err != nil {
		t.Fatalf("decode capture response: %v", err)
	}
	if captured.EvidenceCommitment == "" {
		t.Fatal("expected a non-empty evidence commitment")
	}

	verifyRec := doRequest(t, handler, http.MethodPost, "/api/verify", map[string]any{
		"evidenceId": captured.ID,
	})
	var verifyResult model.VerifyResult
	if err := json.Unmarshal(verifyRec.Body.Bytes(), &verifyResult); err != nil {
		t.Fatalf("decode verify response: %v", err)
	}
	if !verifyResult.Verified {
		t.Fatalf("expected untampered evidence to verify, got status %q", verifyResult.Status)
	}

	tamperRec := doRequest(t, handler, http.MethodPost, "/api/verify", map[string]any{
		"evidenceId":  captured.ID,
		"scrapedText": captured.ScrapedText + " modified",
	})
	var tamperResult model.VerifyResult
	if err := json.Unmarshal(tamperRec.Body.Bytes(), &tamperResult); err != nil {
		t.Fatalf("decode tamper verify response: %v", err)
	}
	if tamperResult.Verified {
		t.Fatal("expected tampered evidence to fail verification")
	}
	if tamperResult.Status != "tampered" {
		t.Errorf("expected status tampered, got %q", tamperResult.Status)
	}
}

func TestCaptureRequiresURL(t *testing.T) {
	rec := doRequest(t, newTestServer().Routes(), http.MethodPost, "/api/captures", map[string]any{
		"scrapedText": "some text",
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing url, got %d", rec.Code)
	}
}

func TestAuthRejectsMissingOrWrongKey(t *testing.T) {
	handler := newTestServerWithKey("secret").Routes()

	rec := doRequest(t, handler, http.MethodGet, "/api/claims", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with no key, got %d", rec.Code)
	}

	rec = doAuthedRequest(t, handler, http.MethodGet, "/api/claims", nil, "Bearer wrong")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with wrong key, got %d", rec.Code)
	}

	rec = doAuthedRequest(t, handler, http.MethodGet, "/api/claims", nil, "Bearer secret")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with correct key, got %d", rec.Code)
	}
}

func TestAuthSkippedWhenKeyUnset(t *testing.T) {
	rec := doRequest(t, newTestServer().Routes(), http.MethodGet, "/api/claims", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 when no api key configured, got %d", rec.Code)
	}
}

func TestHealthNeverRequiresAuth(t *testing.T) {
	rec := doRequest(t, newTestServerWithKey("secret").Routes(), http.MethodGet, "/health", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for /health regardless of api key, got %d", rec.Code)
	}
}

func TestVerifyUnknownEvidenceReturns404(t *testing.T) {
	rec := doRequest(t, newTestServer().Routes(), http.MethodPost, "/api/verify", map[string]any{
		"evidenceId": "ev_does_not_exist",
	})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}
