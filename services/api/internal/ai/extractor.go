package ai

import (
	"regexp"
	"strings"

	"github.com/docsnap/docsnap/services/api/internal/model"
)

type Extractor interface {
	Extract(req model.CaptureRequest) ([]model.Claim, error)
}

type RuleExtractor struct {
	patterns []typedPattern
}

type typedPattern struct {
	claimType string
	re        *regexp.Regexp
}

func NewRuleExtractor() RuleExtractor {
	return RuleExtractor{
		patterns: []typedPattern{
			{"pricing", regexp.MustCompile(`(?i)(\$|usd|price|pricing|cost|free trial|discount|refund)`)},
			{"regulatory", regexp.MustCompile(`(?i)(approved|certified|compliant|regulated|license|licence|audit|policy|terms)`)},
			{"performance", regexp.MustCompile(`(?i)(increase|decrease|growth|revenue|uptime|guarantee|faster|saves|reduces)`)},
			{"security", regexp.MustCompile(`(?i)(encrypted|secure|privacy|private|confidential|verified|identity|risk)`)},
		},
	}
}

func (r RuleExtractor) Extract(req model.CaptureRequest) ([]model.Claim, error) {
	text := strings.TrimSpace(req.ScrapedText)
	if text == "" {
		text = strings.TrimSpace(req.Title)
	}

	sentences := splitSentences(text)
	claims := make([]model.Claim, 0, 8)

	for _, sentence := range sentences {
		trimmed := strings.TrimSpace(sentence)
		if len(trimmed) < 24 {
			continue
		}
		claimType := r.classify(trimmed)
		if claimType == "" {
			continue
		}
		claims = append(claims, model.Claim{
			Text:          trimmed,
			Type:          claimType,
			Confidence:    confidenceFor(trimmed, claimType),
			SourceExcerpt: excerpt(trimmed),
			Status:        "certified",
		})
		if len(claims) == 8 {
			break
		}
	}

	if len(claims) == 0 {
		claims = append(claims, model.Claim{
			Text:          fallbackClaim(req),
			Type:          "web-evidence",
			Confidence:    0.62,
			SourceExcerpt: excerpt(text),
			Status:        "certified",
		})
	}

	return claims, nil
}

func (r RuleExtractor) classify(sentence string) string {
	for _, pattern := range r.patterns {
		if pattern.re.MatchString(sentence) {
			return pattern.claimType
		}
	}
	return ""
}

func splitSentences(text string) []string {
	replacer := strings.NewReplacer("\n", ". ", "\t", " ")
	normalized := replacer.Replace(text)
	return regexp.MustCompile(`[.!?]+`).Split(normalized, -1)
}

func confidenceFor(sentence string, claimType string) float64 {
	score := 0.72
	if len(sentence) > 80 {
		score += 0.08
	}
	if claimType == "regulatory" || claimType == "pricing" {
		score += 0.06
	}
	if score > 0.94 {
		return 0.94
	}
	return score
}

func excerpt(text string) string {
	text = strings.TrimSpace(text)
	if len(text) <= 180 {
		return text
	}
	return text[:180]
}

func fallbackClaim(req model.CaptureRequest) string {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = "Captured webpage"
	}
	return title + " was present at the captured source URL"
}
