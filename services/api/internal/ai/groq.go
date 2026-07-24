package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/docsnap/docsnap/services/api/internal/model"
)

type GroqExtractor struct {
	apiKey string
	url    string
	model  string
	client *http.Client
}

type groqRequest struct {
	Model               string        `json:"model"`
	Messages            []groqMessage `json:"messages"`
	Temperature         float64       `json:"temperature"`
	MaxCompletionTokens int           `json:"max_completion_tokens"`
	TopP                float64       `json:"top_p"`
	Stream              bool          `json:"stream"`
	ResponseFormat      groqFormat    `json:"response_format"`
}

type groqFormat struct {
	Type string `json:"type"`
}

type groqMessage struct {
	Role    string        `json:"role"`
	Content []groqContent `json:"content"`
}

type groqContent struct {
	Type     string        `json:"type"`
	Text     string        `json:"text,omitempty"`
	ImageURL *groqImageURL `json:"image_url,omitempty"`
}

type groqImageURL struct {
	URL string `json:"url"`
}

type groqResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

type groqClaims struct {
	Claims []groqClaim `json:"claims"`
}

type groqClaim struct {
	Text          string  `json:"text"`
	Type          string  `json:"type"`
	Confidence    float64 `json:"confidence"`
	SourceExcerpt string  `json:"sourceExcerpt"`
}

func NewGroqExtractor(apiKey string, baseURL string, modelName string) GroqExtractor {
	baseURL = strings.TrimRight(baseURL, "/")
	if modelName == "" {
		modelName = "qwen/qwen3.6-27b"
	}
	return GroqExtractor{
		apiKey: apiKey,
		url:    baseURL + "/chat/completions",
		model:  modelName,
		client: &http.Client{Timeout: 45 * time.Second},
	}
}

func (g GroqExtractor) Extract(req model.CaptureRequest) ([]model.Claim, error) {
	prompt := `Extract factual, auditable claims from this captured webpage evidence.
Return JSON only with this shape:
{"claims":[{"text":"specific claim text","type":"pricing|regulatory|security|performance|legal|availability|identity|other","confidence":0.0,"sourceExcerpt":"short supporting excerpt"}]}
Rules:
- Use both screenshot OCR/visual context and scraped page text.
- Do not invent claims.
- Prefer concrete claims a compliance reviewer would search for later.
- Return at most 8 claims.
- If no meaningful claims exist, return one web-evidence claim describing what was present.`

	content := []groqContent{
		{Type: "text", Text: prompt + "\n\nURL: " + req.URL + "\nTitle: " + req.Title + "\nScraped text:\n" + truncate(req.ScrapedText, 24000)},
	}
	if strings.TrimSpace(req.ScreenshotDataURL) != "" {
		content = append(content, groqContent{
			Type:     "image_url",
			ImageURL: &groqImageURL{URL: req.ScreenshotDataURL},
		})
	}

	body := groqRequest{
		Model:               g.model,
		Messages:            []groqMessage{{Role: "user", Content: content}},
		Temperature:         0,
		MaxCompletionTokens: 1200,
		TopP:                1,
		Stream:              false,
		ResponseFormat:      groqFormat{Type: "json_object"},
	}

	var buffer bytes.Buffer
	if err := json.NewEncoder(&buffer).Encode(body); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, g.url, &buffer)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+g.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var decoded groqResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if decoded.Error != nil {
			return nil, fmt.Errorf("groq error: %s", decoded.Error.Message)
		}
		return nil, fmt.Errorf("groq error: %s", resp.Status)
	}
	if len(decoded.Choices) == 0 {
		return nil, errors.New("groq returned no choices")
	}

	var parsed groqClaims
	if err := json.Unmarshal([]byte(decoded.Choices[0].Message.Content), &parsed); err != nil {
		return nil, err
	}

	claims := make([]model.Claim, 0, len(parsed.Claims))
	for _, claim := range parsed.Claims {
		text := strings.TrimSpace(claim.Text)
		if text == "" {
			continue
		}
		claimType := strings.TrimSpace(claim.Type)
		if claimType == "" {
			claimType = "other"
		}
		confidence := claim.Confidence
		if confidence <= 0 || confidence > 1 {
			confidence = 0.7
		}
		sourceExcerpt := strings.TrimSpace(claim.SourceExcerpt)
		if sourceExcerpt == "" {
			sourceExcerpt = excerpt(req.ScrapedText)
		}
		claims = append(claims, model.Claim{
			Text:          text,
			Type:          claimType,
			Confidence:    confidence,
			SourceExcerpt: sourceExcerpt,
			Status:        "certified",
		})
	}

	if len(claims) == 0 {
		return NewRuleExtractor().Extract(req)
	}

	return claims, nil
}

func truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}
