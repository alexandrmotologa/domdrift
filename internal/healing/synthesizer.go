package healing

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/alexandrmotologa/domdrift/internal/browser"
)

// Synthesizer generates resilient selectors from DOM nodes.
type Synthesizer struct{}

// NewSynthesizer creates a selector synthesizer.
func NewSynthesizer() *Synthesizer {
	return &Synthesizer{}
}

var (
	hashLikeRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]*[0-9]{2,}[a-zA-Z0-9_-]*$|^_[a-zA-Z0-9]{4,}$|^css-[a-z0-9]{5,}$`)
)

// ComputeShannonEntropy calculates the Shannon entropy of a string token.
func ComputeShannonEntropy(token string) float64 {
	if len(token) == 0 {
		return 0
	}

	charCounts := make(map[rune]int)
	for _, r := range token {
		charCounts[r]++
	}

	total := float64(len(token))
	var entropy float64
	for _, count := range charCounts {
		p := float64(count) / total
		entropy -= p * math.Log2(p)
	}

	return entropy
}

// SynthesizeStableSelector derives the shortest resilient selector for a node.
func (s *Synthesizer) SynthesizeStableSelector(ctx context.Context, sess *browser.Session, node *browser.DOMNode) (string, float64, error) {
	if node == nil {
		return "", 0, fmt.Errorf("node is nil")
	}

	candidates := s.generateCandidates(node)

	// Evaluate candidates inside the browser to verify uniqueness
	bestSelector := ""
	bestScore := -1000.0

	for _, cand := range candidates {
		matchCount, err := s.testSelectorMatches(ctx, sess, cand.Selector)
		if err != nil || matchCount == 0 {
			continue
		}

		// Calculate stability score
		score := cand.BaseScore
		if matchCount == 1 {
			score += 150.0 // Unique single target bonus
		} else {
			score -= float64(matchCount * 10) // Ambiguity penalty
		}

		if score > bestScore {
			bestScore = score
			bestSelector = cand.Selector
		}
	}

	if bestSelector == "" {
		// Fallback to basic tag + index path
		fallback := strings.ToLower(node.NodeName)
		return fallback, 20.0, nil
	}

	confidence := math.Min(0.99, math.Max(0.10, (bestScore+100.0)/250.0))
	return bestSelector, confidence, nil
}

type candidateSelector struct {
	Selector  string
	BaseScore float64
}

func (s *Synthesizer) generateCandidates(node *browser.DOMNode) []candidateSelector {
	var candidates []candidateSelector
	tag := strings.ToLower(node.NodeName)
	if tag == "" {
		tag = strings.ToLower(node.LocalName)
	}
	if tag == "" {
		tag = "*"
	}

	// 1. data-testid / data-cy / data-component attributes
	for _, attr := range []string{"data-testid", "data-test", "data-cy", "data-qa", "data-id"} {
		if val := node.GetAttribute(attr); val != "" {
			candidates = append(candidates, candidateSelector{
				Selector:  fmt.Sprintf("[%s=%q]", attr, val),
				BaseScore: 120.0,
			})
			candidates = append(candidates, candidateSelector{
				Selector:  fmt.Sprintf("%s[%s=%q]", tag, attr, val),
				BaseScore: 125.0,
			})
		}
	}

	// 2. Stable ID attribute (check if ID does not look like random hash)
	if id := node.GetAttribute("id"); id != "" {
		idEntropy := ComputeShannonEntropy(id)
		if idEntropy < 3.0 && !hashLikeRegex.MatchString(id) {
			candidates = append(candidates, candidateSelector{
				Selector:  "#" + id,
				BaseScore: 100.0 - idEntropy*5,
			})
		}
	}

	// 3. Accessible attributes (aria-label, role)
	if ariaLabel := node.GetAttribute("aria-label"); ariaLabel != "" {
		candidates = append(candidates, candidateSelector{
			Selector:  fmt.Sprintf("%s[aria-label=%q]", tag, ariaLabel),
			BaseScore: 90.0,
		})
	}

	if role := node.GetAttribute("role"); role != "" {
		candidates = append(candidates, candidateSelector{
			Selector:  fmt.Sprintf("%s[role=%q]", tag, role),
			BaseScore: 70.0,
		})
	}

	// 4. Clean CSS class tokens (filter out hashes)
	if classAttr := node.GetAttribute("class"); classAttr != "" {
		classes := strings.Fields(classAttr)
		for _, c := range classes {
			c = strings.TrimSpace(c)
			if c == "" {
				continue
			}
			entropy := ComputeShannonEntropy(c)
			isHash := hashLikeRegex.MatchString(c) || entropy > 3.2

			if !isHash && len(c) > 2 {
				score := 80.0 - entropy*10
				candidates = append(candidates, candidateSelector{
					Selector:  fmt.Sprintf("%s.%s", tag, c),
					BaseScore: score,
				})
				candidates = append(candidates, candidateSelector{
					Selector:  "." + c,
					BaseScore: score - 5,
				})
			}
		}
	}

	// 5. Semantic tag itself
	if tag == "h1" || tag == "h2" || tag == "h3" || tag == "main" || tag == "header" || tag == "article" {
		candidates = append(candidates, candidateSelector{
			Selector:  tag,
			BaseScore: 50.0,
		})
	}

	return candidates
}

func (s *Synthesizer) testSelectorMatches(ctx context.Context, sess *browser.Session, selector string) (int, error) {
	script := fmt.Sprintf(`(() => {
		try {
			return document.querySelectorAll(%q).length;
		} catch (e) {
			return 0;
		}
	})()`, selector)

	raw, err := sess.Evaluate(ctx, script)
	if err != nil {
		return 0, err
	}

	var count int
	if err := json.Unmarshal(raw, &count); err != nil {
		return 0, err
	}

	return count, nil
}
