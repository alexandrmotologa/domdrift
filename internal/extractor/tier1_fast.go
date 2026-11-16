package extractor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/alexandrmotologa/domdrift/internal/browser"
	"github.com/alexandrmotologa/domdrift/internal/config"
)

// Tier1Result captures the extraction output from cached selectors.
type Tier1Result struct {
	FieldName string
	Value     string
	Selector  string
	Duration  time.Duration
	Success   bool
	Error     error
}

// ExecuteTier1 attempts fast extraction using cached or initial CSS/XPath selectors.
func ExecuteTier1(ctx context.Context, sess *browser.Session, field config.FieldDefinition, candidateSelector string) Tier1Result {
	start := time.Now()
	res := Tier1Result{
		FieldName: field.Name,
		Selector:  candidateSelector,
	}

	if candidateSelector == "" {
		res.Duration = time.Since(start)
		res.Error = fmt.Errorf("no candidate selector available for field %s", field.Name)
		return res
	}

	// Support both CSS and XPath queries via browser JS execution
	var jsQuery string
	if strings.HasPrefix(candidateSelector, "//") || strings.HasPrefix(candidateSelector, "(//") {
		// XPath query
		jsQuery = fmt.Sprintf(`(() => {
			const res = document.evaluate(%s, document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null);
			const node = res.singleNodeValue;
			if (!node) return null;
			return (node.innerText || node.textContent || node.value || "").trim();
		})()`, quoteJS(candidateSelector))
	} else {
		// CSS query (may contain comma-separated fallback selectors)
		jsQuery = fmt.Sprintf(`(() => {
			const sel = %s;
			const el = document.querySelector(sel);
			if (!el) return null;
			return (el.innerText || el.textContent || el.value || "").trim();
		})()`, quoteJS(candidateSelector))
	}

	rawVal, err := sess.Evaluate(ctx, jsQuery)
	res.Duration = time.Since(start)

	if err != nil {
		res.Error = fmt.Errorf("evaluation failed: %w", err)
		return res
	}

	var extractedStr string
	if err := json.Unmarshal(rawVal, &extractedStr); err != nil {
		// Might be null or empty
		res.Error = fmt.Errorf("null or invalid value returned from selector %s", candidateSelector)
		return res
	}

	extractedStr = strings.TrimSpace(extractedStr)
	if extractedStr == "" {
		res.Error = fmt.Errorf("empty text retrieved for selector %s", candidateSelector)
		return res
	}

	res.Value = extractedStr
	res.Success = true
	return res
}

func quoteJS(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
