package evidence

import (
	"testing"
	"time"

	"github.com/docsnap/docsnap/services/api/internal/model"
)

func TestEvidenceCommitmentDeterministic(t *testing.T) {
	input := HashInput{
		URL:         "https://example.com/pricing",
		Company:     "ExampleCo",
		CaseID:      "CASE-1",
		UserID:      "user@example.com",
		ScrapedText: "Plan costs $99 per month.",
		CapturedAt:  time.Unix(1700000000, 0),
		Claims: []model.Claim{
			{Text: "Plan costs $99 per month.", Type: "pricing", SourceExcerpt: "excerpt"},
		},
	}

	_, _, _, _, commitmentA, _ := NewHasher().Evidence(input)
	_, _, _, _, commitmentB, _ := NewHasher().Evidence(input)

	if commitmentA != commitmentB {
		t.Fatalf("expected identical input to produce identical commitment, got %q vs %q", commitmentA, commitmentB)
	}
}

func TestEvidenceCommitmentChangesOnTamper(t *testing.T) {
	base := HashInput{
		URL:         "https://example.com/pricing",
		Company:     "ExampleCo",
		ScrapedText: "Plan costs $99 per month.",
		CapturedAt:  time.Unix(1700000000, 0),
		Claims: []model.Claim{
			{Text: "Plan costs $99 per month.", Type: "pricing"},
		},
	}
	_, _, _, _, original, _ := NewHasher().Evidence(base)

	tampered := base
	tampered.ScrapedText = "Plan costs $99 per month. modified"
	_, _, _, _, changed, _ := NewHasher().Evidence(tampered)

	if original == changed {
		t.Fatal("expected commitment to change when scraped text is tampered")
	}
}

func TestEvidenceCommitmentStableUnderClaimReordering(t *testing.T) {
	claimsA := []model.Claim{
		{Text: "first", Type: "pricing"},
		{Text: "second", Type: "security"},
	}
	claimsB := []model.Claim{
		{Text: "second", Type: "security"},
		{Text: "first", Type: "pricing"},
	}

	base := HashInput{URL: "https://example.com", CapturedAt: time.Unix(1700000000, 0)}
	inputA := base
	inputA.Claims = claimsA
	inputB := base
	inputB.Claims = claimsB

	_, _, _, rootA, _, _ := NewHasher().Evidence(inputA)
	_, _, _, rootB, _, _ := NewHasher().Evidence(inputB)

	if rootA != rootB {
		t.Fatal("expected claims root to be order-independent")
	}
}

func TestDomainStripsWWW(t *testing.T) {
	cases := map[string]string{
		"https://www.example.com/path": "example.com",
		"https://example.com":          "example.com",
		"not a url %%%":                "",
	}
	for input, want := range cases {
		if got := Domain(input); got != want {
			t.Errorf("Domain(%q) = %q, want %q", input, got, want)
		}
	}
}
