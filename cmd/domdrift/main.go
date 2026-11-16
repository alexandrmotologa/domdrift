package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alexandrmotologa/domdrift/internal/browser"
	"github.com/alexandrmotologa/domdrift/internal/config"
	"github.com/alexandrmotologa/domdrift/internal/export"
	"github.com/alexandrmotologa/domdrift/internal/extractor"
	"github.com/alexandrmotologa/domdrift/internal/healing"
	"github.com/alexandrmotologa/domdrift/internal/server"
	"github.com/alexandrmotologa/domdrift/internal/testserver"
)

const version = "1.0.0"

func printUsage() {
	fmt.Printf(`domdrift v%s - Self-Healing Schema-Driven Web Extraction Engine

Usage:
  domdrift <command> [arguments]

Commands:
  extract       Extract structured data from a single URL using a target schema
  heal          Analyze a target page and re-synthesize broken selectors
  crawl         Batch extract across multiple target URLs
  studio        Launch the embedded React 19 visual extraction studio
  test-server   Launch the local canary mutation test server (v1, v2, v3)
  version       Display current version info

Use "domdrift <command> -h" for more information about a command.
`, version)
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "extract":
		runExtract(args)
	case "heal":
		runHeal(args)
	case "crawl":
		runCrawl(args)
	case "studio":
		runStudio(args)
	case "test-server":
		runTestServer(args)
	case "version", "--version", "-v":
		fmt.Printf("domdrift v%s (Go engine with Chrome DevTools Protocol & React 19 Studio)\n", version)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func runExtract(args []string) {
	fs := flag.NewFlagSet("extract", flag.ExitOnError)
	targetURL := fs.String("url", "", "Target URL to extract")
	schemaPath := fs.String("schema", "examples/schemas/ecommerce_product.json", "Path to JSON schema file")
	outputPath := fs.String("output", "out/extracted.jsonl", "Path to output JSONL file")
	cachePath := fs.String("cache", "selectors.cache.json", "Path to selector cache file")
	configPath := fs.String("config", "", "Optional path to domdrift.yaml config")
	headless := fs.Bool("headless", true, "Run browser in headless mode")

	_ = fs.Parse(args)

	if *targetURL == "" {
		fmt.Fprintln(os.Stderr, "Error: --url is required")
		fs.Usage()
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Load configuration
	var cfg *config.Config
	if *configPath != "" {
		var err error
		cfg, err = config.LoadConfig(*configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed loading config: %v\n", err)
			os.Exit(1)
		}
	} else {
		cfg = &config.Config{
			Pipeline: config.PipelineConfig{
				Tier1Enabled: true,
				Tier2Enabled: true,
				Tier3Enabled: true,
				CacheFile:    *cachePath,
			},
			Vision: config.VisionConfig{
				Provider:            "gemini",
				Model:               "gemini-2.5-flash",
				APIKeyEnv:           "GEMINI_API_KEY",
				FallbackToHeuristic: true,
			},
		}
	}

	// Load schema
	schema, err := config.LoadSchema(*schemaPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed loading schema from %s: %v\n", *schemaPath, err)
		os.Exit(1)
	}

	// Load selector cache
	cache, err := healing.NewSelectorCache(*cachePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed loading cache: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("[domdrift] Launching headless browser (stealth=true, headless=%v)...\n", *headless)
	launchCfg := browser.DefaultLauncherConfig()
	launchCfg.Headless = *headless

	proc, err := browser.Launch(ctx, launchCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed starting browser: %v\n", err)
		os.Exit(1)
	}
	defer proc.Close()

	client, err := browser.NewCDPClient(ctx, proc.WebSocketURL())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed connecting to CDP: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	sess, err := browser.NewSession(ctx, client)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed creating browser session: %v\n", err)
		os.Exit(1)
	}
	defer sess.Close(ctx)

	fmt.Printf("[domdrift] Navigating to %s...\n", *targetURL)
	if err := sess.Navigate(ctx, *targetURL); err != nil {
		fmt.Fprintf(os.Stderr, "Navigation failed: %v\n", err)
		os.Exit(1)
	}

	orch := extractor.NewOrchestrator(cfg, cache, func(evt extractor.DriftEvent) {
		fmt.Printf("  [SELF-HEALING] Field: %s | Tier %d Recovery | New Selector: %s (%dms)\n",
			evt.Field, evt.RecoveryTier, evt.NewSelector, evt.DurationMs)
	})

	start := time.Now()
	records, err := orch.Extract(ctx, sess, *targetURL, schema)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Extraction failed: %v\n", err)
		os.Exit(1)
	}

	duration := time.Since(start)
	fmt.Printf("[domdrift] Extraction succeeded: %d records in %v\n", len(records), duration)

	// Save to JSONL
	writer, err := export.NewJSONLWriter(*outputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed creating output writer: %v\n", err)
		os.Exit(1)
	}
	defer writer.Close()

	for _, rec := range records {
		if err := writer.Write(rec); err != nil {
			fmt.Fprintf(os.Stderr, "Failed exporting record: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Printf("[domdrift] Records written to %s:\n", *outputPath)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	for i, rec := range records {
		if i < 3 {
			_ = enc.Encode(rec.Data)
		}
	}
	if len(records) > 3 {
		fmt.Printf("... and %d more records\n", len(records)-3)
	}
}

func runHeal(args []string) {
	fs := flag.NewFlagSet("heal", flag.ExitOnError)
	targetURL := fs.String("url", "", "Target URL to inspect and self-heal")
	_ = fs.String("schema", "examples/schemas/ecommerce_product.json", "Path to JSON schema file")
	_ = fs.String("cache", "selectors.cache.json", "Path to selector cache file")

	_ = fs.Parse(args)

	if *targetURL == "" {
		fmt.Fprintln(os.Stderr, "Error: --url is required")
		fs.Usage()
		os.Exit(1)
	}

	// Run extraction which automatically triggers self-healing on broken selectors
	runExtract(args)
}

func runCrawl(args []string) {
	fs := flag.NewFlagSet("crawl", flag.ExitOnError)
	urls := fs.String("urls", "", "Comma-separated list of URLs to crawl")
	schemaPath := fs.String("schema", "examples/schemas/ecommerce_product.json", "Path to JSON schema file")
	outputPath := fs.String("output", "out/crawl_results.jsonl", "Path to output JSONL file")

	_ = fs.Parse(args)

	if *urls == "" {
		fmt.Fprintln(os.Stderr, "Error: --urls is required")
		fs.Usage()
		os.Exit(1)
	}

	urlList := []string{*urls}
	for i, u := range urlList {
		fmt.Printf("\n--- [%d/%d] Crawling %s ---\n", i+1, len(urlList), u)
		runExtract([]string{"--url", u, "--schema", *schemaPath, "--output", *outputPath})
	}
}

func runStudio(args []string) {
	fs := flag.NewFlagSet("studio", flag.ExitOnError)
	port := fs.Int("port", 8282, "Port to run React 19 extraction studio")
	cachePath := fs.String("cache", "selectors.cache.json", "Path to selector cache file")

	_ = fs.Parse(args)

	cfg := &config.Config{
		Pipeline: config.PipelineConfig{
			Tier1Enabled: true,
			Tier2Enabled: true,
			Tier3Enabled: true,
			CacheFile:    *cachePath,
		},
		Vision: config.VisionConfig{
			Provider:            "gemini",
			Model:               "gemini-2.5-flash",
			APIKeyEnv:           "GEMINI_API_KEY",
			FallbackToHeuristic: true,
		},
	}

	cache, err := healing.NewSelectorCache(*cachePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed loading cache: %v\n", err)
		os.Exit(1)
	}

	studioSrv := server.NewStudioServer(cfg, cache)
	defer studioSrv.Close()

	mux := http.NewServeMux()
	studioSrv.RegisterRoutes(mux)

	// Serve React 19 embedded assets with SPA fallback
	staticHandler := server.NewStaticFileHandler()
	mux.Handle("/", staticHandler)

	addr := fmt.Sprintf("127.0.0.1:%d", *port)
	httpServer := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	fmt.Printf("\n======================================================\n")
	fmt.Printf("  domdrift v%s - Embedded React 19 Extraction Studio\n", version)
	fmt.Printf("  Dashboard: http://%s\n", addr)
	fmt.Printf("  Event Stream: http://%s/api/events\n", addr)
	fmt.Printf("======================================================\n\n")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println("\n[domdrift] Shutting down studio server...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(ctx)
	}()

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "Studio server failed: %v\n", err)
		os.Exit(1)
	}
}

func runTestServer(args []string) {
	fs := flag.NewFlagSet("test-server", flag.ExitOnError)
	port := fs.Int("port", 9090, "Port to run canary mutation server")

	_ = fs.Parse(args)

	srv, err := testserver.StartCanaryServer(*port)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed launching canary test server: %v\n", err)
		os.Exit(1)
	}
	defer srv.Close()

	fmt.Printf("\n======================================================\n")
	fmt.Printf("  domdrift Canary Mutation Test Server Running on :%d\n", *port)
	fmt.Printf("  - /v1: Clean semantic selectors (Tier 1 fast path)\n")
	fmt.Printf("  - /v2: Obfuscated class hashes (Tier 2 A11y recovery)\n")
	fmt.Printf("  - /v3: Redesigned layout grid (Tier 3 VLM grounding)\n")
	fmt.Printf("======================================================\n\n")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
	fmt.Println("\n[domdrift] Stopping canary server...")
}
