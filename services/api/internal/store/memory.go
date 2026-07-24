package store

import (
	"errors"
	"sort"
	"strings"
	"sync"

	"github.com/docsnap/docsnap/services/api/internal/model"
)

var ErrNotFound = errors.New("not found")

type Memory struct {
	mu       sync.RWMutex
	items    map[string]model.Evidence
	claimIDs map[string]string
}

func NewMemory() *Memory {
	return &Memory{
		items:    map[string]model.Evidence{},
		claimIDs: map[string]string{},
	}
}

func (m *Memory) Save(evidence model.Evidence) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[evidence.ID] = evidence
	for _, claim := range evidence.Claims {
		m.claimIDs[claim.ID] = evidence.ID
	}
}

func (m *Memory) GetEvidence(id string) (model.Evidence, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	item, ok := m.items[id]
	if !ok {
		return model.Evidence{}, ErrNotFound
	}
	return item, nil
}

func (m *Memory) Search(query string, company string, domain string, status string) model.SearchResult {
	m.mu.RLock()
	defer m.mu.RUnlock()

	query = strings.ToLower(strings.TrimSpace(query))
	company = strings.ToLower(strings.TrimSpace(company))
	domain = strings.ToLower(strings.TrimSpace(domain))
	status = strings.ToLower(strings.TrimSpace(status))

	items := make([]model.Evidence, 0, len(m.items))
	claims := make([]model.Claim, 0)

	for _, item := range m.items {
		if company != "" && !strings.Contains(strings.ToLower(item.Company), company) {
			continue
		}
		if domain != "" && !strings.Contains(strings.ToLower(item.Domain), domain) {
			continue
		}
		if status != "" && strings.ToLower(item.VerificationStatus) != status {
			continue
		}

		matchedItem := query == "" || strings.Contains(strings.ToLower(item.URL+" "+item.Title+" "+item.Company+" "+item.CaseID+" "+item.UserID), query)
		matchedClaims := make([]model.Claim, 0)
		for _, claim := range item.Claims {
			claimText := strings.ToLower(claim.Text + " " + claim.Type + " " + claim.SourceExcerpt)
			if query == "" || strings.Contains(claimText, query) {
				matchedClaims = append(matchedClaims, claim)
			}
		}

		if matchedItem || len(matchedClaims) > 0 {
			items = append(items, item)
			claims = append(claims, matchedClaims...)
		}
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	sort.Slice(claims, func(i, j int) bool {
		return claims[i].Confidence > claims[j].Confidence
	})

	return model.SearchResult{Claims: claims, Items: items}
}

