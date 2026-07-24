package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/docsnap/docsnap/services/api/internal/ai"
	"github.com/docsnap/docsnap/services/api/internal/config"
	"github.com/docsnap/docsnap/services/api/internal/evidence"
	"github.com/docsnap/docsnap/services/api/internal/flare"
	"github.com/docsnap/docsnap/services/api/internal/model"
	"github.com/docsnap/docsnap/services/api/internal/store"
	"github.com/docsnap/docsnap/services/api/internal/tee"
)

type memoryRepo struct {
	items map[string]model.Evidence
}

func newMemoryRepo() *memoryRepo {
	return &memoryRepo{items: map[string]model.Evidence{}}
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
		items = append(items, item)
	}
	return model.SearchResult{Items: items}, nil
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
	return NewServer(
		config.Config{AppOrigin: "http://localhost:3000"},
		newMemoryRepo(),
		ai.NewRuleExtractor(),
		evidence.NewHasher(),
		flare.NewSimulatedClient(),
		newMemoryStore(),
		tee.NewLocalCertifier(),
	)
}

func doRequest(t *testing.T, handler http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
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

func TestVerifyUnknownEvidenceReturns404(t *testing.T) {
	rec := doRequest(t, newTestServer().Routes(), http.MethodPost, "/api/verify", map[string]any{
		"evidenceId": "ev_does_not_exist",
	})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}
