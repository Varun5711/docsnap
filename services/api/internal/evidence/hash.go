package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/docsnap/docsnap/services/api/internal/model"
)

type Hasher struct{}

type HashInput struct {
	URL               string
	Company           string
	CaseID            string
	UserID            string
	ScreenshotDataURL string
	ScrapedText       string
	CapturedAt        time.Time
	Claims            []model.Claim
}

func NewHasher() Hasher {
	return Hasher{}
}

func (Hasher) Evidence(input HashInput) (string, string, string, string, string, []model.Claim) {
	screenshotHash := digest(normalizeDataURL(input.ScreenshotDataURL))
	textHash := digest(normalizeText(input.ScrapedText))
	metadataCommitment := digest(input.URL + "|" + input.Company + "|" + input.CaseID + "|" + input.UserID + "|" + input.CapturedAt.UTC().Format(time.RFC3339Nano))

	claims := make([]model.Claim, len(input.Claims))
	copy(claims, input.Claims)
	for i := range claims {
		claims[i].Hash = digest(claims[i].Text + "|" + claims[i].Type + "|" + claims[i].SourceExcerpt + "|" + screenshotHash + "|" + textHash)
	}

	claimHashes := make([]string, len(claims))
	for i, claim := range claims {
		claimHashes[i] = claim.Hash
	}
	sort.Strings(claimHashes)
	claimsRoot := digest(strings.Join(claimHashes, "|"))
	evidenceCommitment := digest(screenshotHash + "|" + textHash + "|" + metadataCommitment + "|" + claimsRoot)

	return screenshotHash, textHash, metadataCommitment, claimsRoot, evidenceCommitment, claims
}

func Domain(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	host := parsed.Hostname()
	if strings.HasPrefix(host, "www.") {
		return strings.TrimPrefix(host, "www.")
	}
	return host
}

func CanonicalJSON(value any) string {
	body, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(body)
}

func digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func normalizeDataURL(value string) string {
	return strings.TrimSpace(value)
}

func normalizeText(value string) string {
	fields := strings.Fields(value)
	return strings.Join(fields, " ")
}
