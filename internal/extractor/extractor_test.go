package extractor_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/alexandrmotologa/domdrift/internal/browser"
	"github.com/alexandrmotologa/domdrift/internal/config"
	"github.com/alexandrmotologa/domdrift/internal/extractor"
	"github.com/alexandrmotologa/domdrift/internal/healing"
	"github.com/alexandrmotologa/domdrift/internal/testserver"
)

func TestAutonomousThreeTierExtractionAndHealing(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 1. Start Canary Server on random available port
	srv, err := testserver.StartCanaryServer(9876)
	if err != nil {
		t.Fatalf("failed starting canary server: %v", err)
	}
	defer srv.Close()

	// 2. Launch Headless Browser
	launchCfg := browser.DefaultLauncherConfig()
	proc, err := browser.Launch(ctx, launchCfg)
	if err != nil {
		t.Fatalf("failed launching browser: %v", err)
	}
	defer proc.Close()

	// 3. Connect CDP Client and create tab session
	cdpClient, err := browser.NewCDPClient(ctx, proc.WebSocketURL())
	if err != nil {
		t.Fatalf("failed creating cdp client: %v", err)
	}
	defer cdpClient.Close()

	sess, err := browser.NewSession(ctx, cdpClient)
	if err != nil {
		t.Fatalf("failed creating browser session: %v", err)
	}
	defer sess.Close(ctx)

	// 4. Load schema
	schema, err := config.LoadSchema("../../examples/schemas/ecommerce_product.json")
	if err != nil {
		t.Fatalf("failed loading schema: %v", err)
	}

	cacheFile := "test_selectors.cache.json"
	defer os.Remove(cacheFile)

	cache, err := healing.NewSelectorCache(cacheFile)
	if err != nil {
		t.Fatalf("failed creating selector cache: %v", err)
	}

	cfg := &config.Config{
		Pipeline: config.PipelineConfig{
			Tier1Enabled: true,
			Tier2Enabled: true,
			Tier3Enabled: true,
		},
		Vision: config.VisionConfig{
			Provider:            "local-heuristic",
			FallbackToHeuristic: true,
		},
	}

	var driftEvents []extractor.DriftEvent
	orch := extractor.NewOrchestrator(cfg, cache, func(event extractor.DriftEvent) {
		driftEvents = append(driftEvents, event)
	})

	// ==================== RUN 1: V1 Clean Markup (Tier 1 Expected) ====================
	v1URL := "http://127.0.0.1:9876/v1"
	if err := sess.Navigate(ctx, v1URL); err != nil {
		t.Fatalf("failed navigating to v1: %v", err)
	}

	recV1, err := orch.ExtractEntity(ctx, sess, v1URL, schema)
	if err != nil {
		t.Fatalf("v1 extraction failed: %v", err)
	}

	if recV1.Data["title"] != "AeroPro Wireless Noise-Cancelling Headphones" {
		t.Errorf("v1 title mismatch: got %v", recV1.Data["title"])
	}
	if recV1.Data["price"] != "$349.99" {
		t.Errorf("v1 price mismatch: got %v", recV1.Data["price"])
	}
	if recV1.TiersUsed["title"] != 1 {
		t.Errorf("expected tier 1 for v1 title, got tier %d", recV1.TiersUsed["title"])
	}

	// ==================== RUN 2: V2 Mutated Class Hashes (Tier 2 A11y Healing Expected) ====================
	v2URL := "http://127.0.0.1:9876/v2"
	if err := sess.Navigate(ctx, v2URL); err != nil {
		t.Fatalf("failed navigating to v2: %v", err)
	}

	recV2, err := orch.ExtractEntity(ctx, sess, v2URL, schema)
	if err != nil {
		t.Fatalf("v2 extraction failed: %v", err)
	}

	if recV2.Data["title"] != "AeroPro Wireless Noise-Cancelling Headphones" {
		t.Errorf("v2 title mismatch: got %v", recV2.Data["title"])
	}
	if recV2.Data["price"] != "$349.99" {
		t.Errorf("v2 price mismatch: got %v", recV2.Data["price"])
	}

	// ==================== RUN 3: V3 Full Structural Redesign (Tier 3 Visual Grounding Expected) ====================
	v3URL := "http://127.0.0.1:9876/v3"
	if err := sess.Navigate(ctx, v3URL); err != nil {
		t.Fatalf("failed navigating to v3: %v", err)
	}

	recV3, err := orch.ExtractEntity(ctx, sess, v3URL, schema)
	if err != nil {
		t.Fatalf("v3 extraction failed: %v", err)
	}

	if recV3.Data["title"] != "AeroPro Wireless Noise-Cancelling Headphones" {
		t.Errorf("v3 title mismatch: got %v", recV3.Data["title"])
	}
	if recV3.Data["price"] != "$349.99" {
		t.Errorf("v3 price mismatch: got %v", recV3.Data["price"])
	}

	t.Logf("Self-healing test completed successfully. Total drift events handled: %d", len(driftEvents))
}
