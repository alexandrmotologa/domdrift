package server

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/alexandrmotologa/domdrift/internal/browser"
	"github.com/alexandrmotologa/domdrift/internal/config"
	"github.com/alexandrmotologa/domdrift/internal/extractor"
	"github.com/alexandrmotologa/domdrift/internal/healing"
)

// StudioServer coordinates HTTP API, SSE streaming, and studio assets.
type StudioServer struct {
	cfg          *config.Config
	cache        *healing.SelectorCache
	hub          *SSEHub
	mu           sync.RWMutex
	driftHistory []extractor.DriftEvent
	lastScreen   []byte

	browserProc *browser.BrowserProcess
	cdpClient   *browser.CDPClient
	session     *browser.Session
}

// NewStudioServer instantiates the API server.
func NewStudioServer(cfg *config.Config, cache *healing.SelectorCache) *StudioServer {
	return &StudioServer{
		cfg:   cfg,
		cache: cache,
		hub:   NewSSEHub(),
	}
}

// SetupBrowser initializes or recovers the shared browser session for interactive studio work.
func (s *StudioServer) SetupBrowser(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.browserProc != nil && s.browserProc.IsAlive() && s.session != nil {
		return nil
	}

	launchCfg := browser.DefaultLauncherConfig()
	proc, err := browser.Launch(ctx, launchCfg)
	if err != nil {
		return err
	}

	client, err := browser.NewCDPClient(ctx, proc.WebSocketURL())
	if err != nil {
		_ = proc.Close()
		return err
	}

	sess, err := browser.NewSession(ctx, client)
	if err != nil {
		_ = client.Close()
		_ = proc.Close()
		return err
	}

	s.browserProc = proc
	s.cdpClient = client
	s.session = sess
	return nil
}

// RegisterRoutes attaches REST API and SSE endpoints to a mux.
func (s *StudioServer) RegisterRoutes(mux *http.ServeMux) {
	mux.Handle("/api/events", s.hub)
	mux.HandleFunc("/api/extract", s.handleExtract)
	mux.HandleFunc("/api/cache", s.handleCache)
	mux.HandleFunc("/api/cache/reset", s.handleCacheReset)
	mux.HandleFunc("/api/drift-history", s.handleDriftHistory)
	mux.HandleFunc("/api/screenshot", s.handleScreenshot)
}

func (s *StudioServer) handleExtract(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		URL    string                   `json:"url"`
		Schema *config.ExtractionSchema `json:"schema"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		http.Error(w, "Target URL is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()

	if err := s.SetupBrowser(ctx); err != nil {
		http.Error(w, "Failed launching browser: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := s.session.Navigate(ctx, req.URL); err != nil {
		http.Error(w, "Navigation failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Capture screenshot for visual studio preview
	if scr, err := s.session.CaptureScreenshot(ctx, "png"); err == nil {
		s.mu.Lock()
		s.lastScreen = scr
		s.mu.Unlock()
		s.hub.Broadcast("screenshot_updated", map[string]interface{}{"url": req.URL})
	}

	orch := extractor.NewOrchestrator(s.cfg, s.cache, func(event extractor.DriftEvent) {
		s.mu.Lock()
		s.driftHistory = append(s.driftHistory, event)
		s.mu.Unlock()
		s.hub.Broadcast("drift_event", event)
	})

	record, err := orch.ExtractEntity(ctx, s.session, req.URL, req.Schema)
	if err != nil {
		http.Error(w, "Extraction failed: "+err.Error(), http.StatusUnprocessableEntity)
		return
	}

	s.hub.Broadcast("extraction_record", record)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(record)
}

func (s *StudioServer) handleCache(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	s.cache.Save()
	_ = json.NewEncoder(w).Encode(s.cache.Data)
}

func (s *StudioServer) handleCacheReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.cache.Data = make(map[string]map[string]healing.CachedSelector)
	_ = s.cache.Save()
	s.hub.Broadcast("cache_reset", map[string]bool{"success": true})
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"cache cleared"}`))
}

func (s *StudioServer) handleDriftHistory(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.driftHistory)
}

func (s *StudioServer) handleScreenshot(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	scr := s.lastScreen
	s.mu.RUnlock()

	if len(scr) == 0 {
		http.Error(w, "No screenshot available yet", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	_, _ = w.Write(scr)
}

// Close cleans up browser and listeners.
func (s *StudioServer) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.session != nil {
		_ = s.session.Close(context.Background())
	}
	if s.cdpClient != nil {
		_ = s.cdpClient.Close()
	}
	if s.browserProc != nil {
		_ = s.browserProc.Close()
	}
}
