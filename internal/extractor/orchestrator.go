package extractor

import (
	"context"
	"fmt"
	"time"

	"github.com/alexandrmotologa/domdrift/internal/browser"
	"github.com/alexandrmotologa/domdrift/internal/config"
	"github.com/alexandrmotologa/domdrift/internal/healing"
)

// DriftEvent records an occurrence of selector degradation and auto-repair.
type DriftEvent struct {
	Timestamp    time.Time `json:"timestamp"`
	URL          string    `json:"url"`
	Field        string    `json:"field"`
	OldSelector  string    `json:"oldSelector"`
	NewSelector  string    `json:"newSelector"`
	RecoveryTier int       `json:"recoveryTier"` // 2 or 3
	Confidence   float64   `json:"confidence"`
	DurationMs   int64     `json:"durationMs"`
	Reason       string    `json:"reason"`
}

// ExtractedRecord represents the structured data extracted for an entity.
type ExtractedRecord struct {
	URL         string                 `json:"url"`
	Timestamp   time.Time              `json:"timestamp"`
	Data        map[string]interface{} `json:"data"`
	TiersUsed   map[string]int         `json:"tiersUsed"`
	DriftEvents []DriftEvent           `json:"driftEvents,omitempty"`
}

// Orchestrator coordinates the 3-tier extraction and self-healing lifecycle.
type Orchestrator struct {
	cfg         *config.Config
	cache       *healing.SelectorCache
	grounder    *healing.CompositeGrounder
	resolver    *healing.NodeResolver
	synthesizer *healing.Synthesizer
	onDrift     func(DriftEvent)
}

// NewOrchestrator creates a new extraction orchestrator.
func NewOrchestrator(cfg *config.Config, cache *healing.SelectorCache, onDrift func(DriftEvent)) *Orchestrator {
	return &Orchestrator{
		cfg:         cfg,
		cache:       cache,
		grounder:    healing.NewVisionGrounder(cfg.Vision),
		resolver:    healing.NewNodeResolver(),
		synthesizer: healing.NewSynthesizer(),
		onDrift:     onDrift,
	}
}

// ExtractEntity runs the 3-tier extraction sequence across all fields in the schema.
func (o *Orchestrator) ExtractEntity(ctx context.Context, sess *browser.Session, targetURL string, schema *config.ExtractionSchema) (*ExtractedRecord, error) {
	record := &ExtractedRecord{
		URL:       targetURL,
		Timestamp: time.Now().UTC(),
		Data:      make(map[string]interface{}),
		TiersUsed: make(map[string]int),
	}

	domain := healing.ExtractDomain(targetURL)

	for _, field := range schema.Fields {
		val, tier, err := o.extractFieldWithFallback(ctx, sess, domain, targetURL, field, schema, record)
		if err != nil {
			if field.Required {
				return record, fmt.Errorf("extraction failed for required field %s: %w", field.Name, err)
			}
			continue
		}
		record.Data[field.Name] = val
		record.TiersUsed[field.Name] = tier
	}

	_ = o.cache.Save()
	return record, nil
}

func (o *Orchestrator) extractFieldWithFallback(
	ctx context.Context,
	sess *browser.Session,
	domain, targetURL string,
	field config.FieldDefinition,
	schema *config.ExtractionSchema,
	record *ExtractedRecord,
) (string, int, error) {
	start := time.Now()

	// 1. Determine candidate selector from Cache or Schema
	cached, hasCached := o.cache.Get(domain, field.Name)
	candidateSelector := field.InitialSelector
	if hasCached && cached.Selector != "" {
		candidateSelector = cached.Selector
	}

	// ==================== TIER 1: Fast Cached Selector ====================
	if o.cfg.Pipeline.Tier1Enabled && candidateSelector != "" {
		t1Res := ExecuteTier1(ctx, sess, field, candidateSelector)
		if t1Res.Success {
			// Validate extracted value with schema
			if err := schema.ValidateField(field.Name, t1Res.Value); err == nil {
				o.cache.RecordSuccess(domain, field.Name)
				return t1Res.Value, 1, nil
			}
		}
	}

	// Tier 1 broke or produced invalid data: trigger self-healing
	o.cache.RecordFailure(domain, field.Name)

	// ==================== TIER 2: Accessibility Tree (A11y) ====================
	if o.cfg.Pipeline.Tier2Enabled {
		t2Res := ExecuteTier2(ctx, sess, field)
		if t2Res.Success {
			if err := schema.ValidateField(field.Name, t2Res.Value); err == nil {
				// Re-synthesize stable selector for this node
				var newSelector string
				var confidence float64
				if t2Res.BackendDOMNodeID > 0 {
					domNode, err := sess.DescribeBackendNode(ctx, t2Res.BackendDOMNodeID, 3)
					if err == nil && domNode != nil {
						newSelector, confidence, _ = o.synthesizer.SynthesizeStableSelector(ctx, sess, domNode)
					}
				}

				if newSelector == "" {
					newSelector = candidateSelector
					confidence = 0.85
				}

				// Update cache
				o.cache.Set(domain, field.Name, healing.CachedSelector{
					Selector:    newSelector,
					Tier:        2,
					Confidence:  confidence,
					Stability:   0.88,
					LastSuccess: time.Now().UTC(),
					HealCount:   cached.HealCount + 1,
				})

				event := DriftEvent{
					Timestamp:    time.Now().UTC(),
					URL:          targetURL,
					Field:        field.Name,
					OldSelector:  candidateSelector,
					NewSelector:  newSelector,
					RecoveryTier: 2,
					Confidence:   confidence,
					DurationMs:   time.Since(start).Milliseconds(),
					Reason:       fmt.Sprintf("Tier 1 selector broken; recovered via A11y role %q", t2Res.RoleMatched),
				}
				record.DriftEvents = append(record.DriftEvents, event)
				if o.onDrift != nil {
					o.onDrift(event)
				}

				return t2Res.Value, 2, nil
			}
		}
	}

	// ==================== TIER 3: Multimodal Visual Grounding ====================
	if o.cfg.Pipeline.Tier3Enabled {
		screenshot, err := sess.CaptureScreenshot(ctx, "png")
		if err == nil {
			grounding, err := o.grounder.Ground(ctx, sess, screenshot, field)
			if err == nil && grounding != nil {
				resolved, err := o.resolver.Resolve(ctx, sess, grounding.X, grounding.Y)
				if err == nil && resolved != nil && resolved.Node != nil {
					newSelector, confidence, err := o.synthesizer.SynthesizeStableSelector(ctx, sess, resolved.Node)
					if err == nil && newSelector != "" {
						// Extract using synthesized selector
						t1Check := ExecuteTier1(ctx, sess, field, newSelector)
						if t1Check.Success {
							if err := schema.ValidateField(field.Name, t1Check.Value); err == nil {
								o.cache.Set(domain, field.Name, healing.CachedSelector{
									Selector:    newSelector,
									Tier:        3,
									Confidence:  confidence,
									Stability:   0.92,
									LastSuccess: time.Now().UTC(),
									HealCount:   cached.HealCount + 1,
								})

								event := DriftEvent{
									Timestamp:    time.Now().UTC(),
									URL:          targetURL,
									Field:        field.Name,
									OldSelector:  candidateSelector,
									NewSelector:  newSelector,
									RecoveryTier: 3,
									Confidence:   confidence,
									DurationMs:   time.Since(start).Milliseconds(),
									Reason:       fmt.Sprintf("Structural redesign detected; recovered via Tier 3 Visual Grounding (%s at [%d, %d])", grounding.Source, grounding.X, grounding.Y),
								}
								record.DriftEvents = append(record.DriftEvents, event)
								if o.onDrift != nil {
									o.onDrift(event)
								}

								return t1Check.Value, 3, nil
							}
						}
					}
				}
			}
		}
	}

	return "", 0, fmt.Errorf("all extraction tiers exhausted for field %s", field.Name)
}
