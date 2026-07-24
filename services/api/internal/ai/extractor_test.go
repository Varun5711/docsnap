package ai

import (
	"testing"

	"github.com/docsnap/docsnap/services/api/internal/model"
)

func TestRuleExtractorClassifiesByKeyword(t *testing.T) {
	req := model.CaptureRequest{
		ScrapedText: "Our enterprise plan costs $99 per month with a free trial available. " +
			"The platform is SOC 2 certified and fully compliant with regulations. " +
			"Encrypted storage keeps your data private and secure at all times.",
	}

	claims, err := NewRuleExtractor().Extract(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	types := map[string]bool{}
	for _, claim := range claims {
		types[claim.Type] = true
	}

	for _, want := range []string{"pricing", "regulatory", "security"} {
		if !types[want] {
			t.Errorf("expected a %q claim among %+v", want, claims)
		}
	}
}

func TestRuleExtractorFallsBackWhenNoClaims(t *testing.T) {
	req := model.CaptureRequest{
		Title:       "A Blank Page",
		ScrapedText: "hi",
	}

	claims, err := NewRuleExtractor().Extract(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(claims) != 1 {
		t.Fatalf("expected exactly one fallback claim, got %d", len(claims))
	}
	if claims[0].Type != "web-evidence" {
		t.Errorf("expected fallback claim type web-evidence, got %q", claims[0].Type)
	}
}

func TestRuleExtractorCapsAtEightClaims(t *testing.T) {
	text := ""
	for i := 0; i < 20; i++ {
		text += "This sentence mentions a price and a discount offer today. "
	}

	claims, err := NewRuleExtractor().Extract(model.CaptureRequest{ScrapedText: text})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(claims) > 8 {
		t.Fatalf("expected at most 8 claims, got %d", len(claims))
	}
}
