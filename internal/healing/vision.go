package healing

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/alexandrmotologa/domdrift/internal/browser"
	"github.com/alexandrmotologa/domdrift/internal/config"
)

// GroundingResult holds pixel coordinates identified for a target field.
type GroundingResult struct {
	X          int     `json:"x"`
	Y          int     `json:"y"`
	Confidence float64 `json:"confidence"`
	Source     string  `json:"source"` // "vlm-gemini", "local-heuristic"
}

// VisionGrounder defines the visual coordinate resolver interface.
type VisionGrounder interface {
	Ground(ctx context.Context, sess *browser.Session, screenshot []byte, field config.FieldDefinition) (*GroundingResult, error)
}

// CompositeGrounder orchestrates cloud VLM and local deterministic fallback.
type CompositeGrounder struct {
	cfg        config.VisionConfig
	httpClient *http.Client
}

// NewVisionGrounder initializes the vision grounding engine.
func NewVisionGrounder(cfg config.VisionConfig) *CompositeGrounder {
	return &CompositeGrounder{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Ground resolves pixel coordinates for a field from the screenshot.
func (g *CompositeGrounder) Ground(ctx context.Context, sess *browser.Session, screenshot []byte, field config.FieldDefinition) (*GroundingResult, error) {
	apiKey := ""
	if g.cfg.APIKeyEnv != "" {
		apiKey = os.Getenv(g.cfg.APIKeyEnv)
	}

	// 1. If API key exists and provider is VLM, attempt cloud multimodal grounding
	if apiKey != "" && (g.cfg.Provider == "gemini" || g.cfg.Provider == "") {
		res, err := g.groundWithGemini(ctx, screenshot, field, apiKey)
		if err == nil && res != nil {
			return res, nil
		}
		// If failed, fall back to local heuristic if configured
		if !g.cfg.FallbackToHeuristic {
			return nil, err
		}
	}

	// 2. Local deterministic visual heuristic using browser viewport layout
	return g.groundWithLocalHeuristic(ctx, sess, field)
}

func (g *CompositeGrounder) groundWithGemini(ctx context.Context, screenshot []byte, field config.FieldDefinition, apiKey string) (*GroundingResult, error) {
	model := g.cfg.Model
	if model == "" {
		model = "gemini-2.5-flash"
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, apiKey)

	prompt := fmt.Sprintf(`Analyze this webpage screenshot and locate the visual element representing the field "%s" (Description: "%s", Semantic role: "%s").
Return a JSON object with the exact pixel center coordinates (x, y) where x is horizontal pixels from the top-left and y is vertical pixels from the top-left.
Format your entire output as raw JSON:
{"x": 120, "y": 340, "confidence": 0.95}`, field.Name, field.Description, field.SemanticRole)

	reqBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []interface{}{
					map[string]interface{}{
						"text": prompt,
					},
					map[string]interface{}{
						"inline_data": map[string]interface{}{
							"mime_type": "image/png",
							"data":      base64.StdEncoding.EncodeToString(screenshot),
						},
					},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"response_mime_type": "application/json",
			"temperature":        0.1,
		},
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gemini vision request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gemini api returned status %d: %s", resp.StatusCode, string(body))
	}

	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return nil, err
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, errors.New("empty response from gemini vision")
	}

	rawText := strings.TrimSpace(geminiResp.Candidates[0].Content.Parts[0].Text)
	var coords struct {
		X          int     `json:"x"`
		Y          int     `json:"y"`
		Confidence float64 `json:"confidence"`
	}

	if err := json.Unmarshal([]byte(rawText), &coords); err != nil {
		return nil, fmt.Errorf("failed parsing coordinates from model response %q: %w", rawText, err)
	}

	return &GroundingResult{
		X:          coords.X,
		Y:          coords.Y,
		Confidence: coords.Confidence,
		Source:     "vlm-gemini",
	}, nil
}

// groundWithLocalHeuristic locates elements visually using DOM viewport bounding boxes
func (g *CompositeGrounder) groundWithLocalHeuristic(ctx context.Context, sess *browser.Session, field config.FieldDefinition) (*GroundingResult, error) {
	// Script that inspects visible DOM rectangles and scores candidates
	script := fmt.Sprintf(`(() => {
		const fieldName = %q.toLowerCase();
		const role = %q.toLowerCase();
		const pattern = %q;
		const regex = pattern ? new RegExp(pattern, "i") : null;

		const elements = Array.from(document.querySelectorAll("*")).filter(el => {
			const rect = el.getBoundingClientRect();
			return rect.width > 0 && rect.height > 0 && rect.top >= 0 && rect.top <= window.innerHeight && el.childElementCount <= 2;
		});

		let bestEl = null;
		let bestScore = -1;

		for (const el of elements) {
			const text = (el.innerText || el.textContent || "").trim();
			if (!text || text.length > 300) continue;

			let score = 0;
			const rect = el.getBoundingClientRect();

			if (role === "heading" || fieldName === "title") {
				if (/^H[1-4]$/i.test(el.tagName)) score += 50;
				if (text.length > 5 && text.length < 120) score += 30;
			} else if (role === "price" || fieldName === "price") {
				if (/[\$\€\£\¥\₹]|USD|EUR|MDL|RON/i.test(text)) score += 60;
				if (/[0-9]+[.,][0-9]{2}/.test(text)) score += 40;
			} else {
				if (text.toLowerCase().includes(fieldName)) score += 30;
			}

			if (regex && regex.test(text)) {
				score += 50;
			}

			// Favor elements positioned in main content area
			if (rect.top > 50 && rect.top < 800) score += 10;

			if (score > bestScore) {
				bestScore = score;
				bestEl = el;
			}
		}

		if (!bestEl || bestScore < 20) return null;

		const rect = bestEl.getBoundingClientRect();
		return {
			x: Math.round(rect.left + rect.width / 2),
			y: Math.round(rect.top + rect.height / 2),
			confidence: Math.min(0.99, bestScore / 100),
			tag: bestEl.tagName.toLowerCase(),
			text: (bestEl.innerText || bestEl.textContent || "").trim()
		};
	})()`, field.Name, field.SemanticRole, field.Pattern)

	raw, err := sess.Evaluate(ctx, script)
	if err != nil {
		return nil, fmt.Errorf("local visual heuristic script failed: %w", err)
	}

	var res struct {
		X          int     `json:"x"`
		Y          int     `json:"y"`
		Confidence float64 `json:"confidence"`
	}

	if err := json.Unmarshal(raw, &res); err != nil || (res.X == 0 && res.Y == 0) {
		return nil, fmt.Errorf("local visual heuristic could not localize field %s", field.Name)
	}

	return &GroundingResult{
		X:          res.X,
		Y:          res.Y,
		Confidence: res.Confidence,
		Source:     "local-heuristic",
	}, nil
}
