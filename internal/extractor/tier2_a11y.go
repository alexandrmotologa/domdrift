package extractor

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/alexandrmotologa/domdrift/internal/browser"
	"github.com/alexandrmotologa/domdrift/internal/config"
)

// Tier2Result captures semantic extraction from Chrome Accessibility Tree.
type Tier2Result struct {
	FieldName        string
	Value            string
	RoleMatched      string
	BackendDOMNodeID int64
	Duration         time.Duration
	Success          bool
	Error            error
}

var (
	priceRegex    = regexp.MustCompile(`[$€£¥₹]?\s*[0-9]+([.,][0-9]{2})?(\s*(USD|EUR|MDL|RON|GBP))?`)
	currencyRegex = regexp.MustCompile(`[$€£¥₹]|USD|EUR|MDL|RON|GBP`)
)

// ExecuteTier2 navigates the Chrome Accessibility Tree to locate fields semantically.
func ExecuteTier2(ctx context.Context, sess *browser.Session, field config.FieldDefinition) Tier2Result {
	start := time.Now()
	res := Tier2Result{
		FieldName: field.Name,
	}

	nodes, err := sess.GetAccessibilityTree(ctx)
	if err != nil {
		res.Duration = time.Since(start)
		res.Error = fmt.Errorf("failed fetching accessibility tree: %w", err)
		return res
	}

	// 1. Target matching by SemanticRole
	role := strings.ToLower(field.SemanticRole)
	nameLower := strings.ToLower(field.Name)

	switch {
	case role == "heading" || nameLower == "title" || nameLower == "name":
		val, backendID := matchHeadingOrTitle(nodes)
		if val != "" {
			res.Value = val
			res.RoleMatched = "heading"
			res.BackendDOMNodeID = backendID
			res.Success = true
			res.Duration = time.Since(start)
			return res
		}

	case role == "price" || nameLower == "price" || nameLower == "cost":
		val, backendID := matchPrice(nodes)
		if val != "" {
			res.Value = val
			res.RoleMatched = "price"
			res.BackendDOMNodeID = backendID
			res.Success = true
			res.Duration = time.Since(start)
			return res
		}

	default:
		// Generic semantic label traversal (locate label node, then find following sibling value)
		val, backendID, matchedRole := matchLabelProximity(nodes, field.Name, field.Description)
		if val != "" {
			res.Value = val
			res.RoleMatched = matchedRole
			res.BackendDOMNodeID = backendID
			res.Success = true
			res.Duration = time.Since(start)
			return res
		}
	}

	res.Duration = time.Since(start)
	res.Error = fmt.Errorf("no matching node found in accessibility tree for field %s (role: %s)", field.Name, field.SemanticRole)
	return res
}

func matchHeadingOrTitle(nodes []browser.AXNode) (string, int64) {
	for _, n := range nodes {
		if n.Ignored {
			continue
		}
		role := strings.ToLower(n.RoleName())
		name := strings.TrimSpace(n.NameString())
		if (role == "heading" || role == "banner") && name != "" && len(name) > 2 {
			return name, n.BackendDOMNodeID
		}
	}
	return "", 0
}

func matchPrice(nodes []browser.AXNode) (string, int64) {
	for _, n := range nodes {
		if n.Ignored {
			continue
		}
		text := strings.TrimSpace(n.NameString())
		if text == "" && n.Value != nil {
			text = fmt.Sprintf("%v", n.Value.Value)
		}

		if text != "" && currencyRegex.MatchString(text) {
			match := priceRegex.FindString(text)
			if match != "" {
				return strings.TrimSpace(match), n.BackendDOMNodeID
			}
		}
	}
	return "", 0
}

func matchLabelProximity(nodes []browser.AXNode, fieldName, fieldDesc string) (string, int64, string) {
	targets := []string{
		strings.ToLower(fieldName),
		strings.ToLower(fieldName) + ":",
	}

	for i, n := range nodes {
		if n.Ignored {
			continue
		}
		name := strings.ToLower(strings.TrimSpace(n.NameString()))
		isMatch := false
		for _, target := range targets {
			if strings.EqualFold(name, target) || strings.HasPrefix(name, target) {
				isMatch = true
				break
			}
		}

		if isMatch {
			// Look at immediate subsequent siblings in the AXTree
			for j := i + 1; j < len(nodes) && j <= i+3; j++ {
				next := nodes[j]
				if next.Ignored {
					continue
				}
				val := strings.TrimSpace(next.NameString())
				if val == "" && next.Value != nil {
					val = fmt.Sprintf("%v", next.Value.Value)
				}
				if val != "" && !strings.EqualFold(val, name) {
					return val, next.BackendDOMNodeID, next.RoleName()
				}
			}
		}
	}

	return "", 0, ""
}
