package healing

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CachedSelector stores a learned stable selector with health metrics.
type CachedSelector struct {
	Selector    string    `json:"selector"`
	Tier        int       `json:"tier"` // 1: Cached/CSS, 2: A11y, 3: Visual
	Confidence  float64   `json:"confidence"`
	Stability   float64   `json:"stability"`
	LastSuccess time.Time `json:"lastSuccess"`
	UpdatedAt   time.Time `json:"updatedAt"`
	Failures    int       `json:"failures"`
	HealCount   int       `json:"healCount"`
}

// SelectorCache persists learned selectors across runs.
type SelectorCache struct {
	mu       sync.RWMutex
	filePath string
	Data     map[string]map[string]CachedSelector `json:"domains"` // domain -> fieldName -> CachedSelector
}

// NewSelectorCache initializes or loads a persistent selector cache.
func NewSelectorCache(filePath string) (*SelectorCache, error) {
	if filePath == "" {
		filePath = "selectors.cache.json"
	}

	c := &SelectorCache{
		filePath: filePath,
		Data:     make(map[string]map[string]CachedSelector),
	}

	if err := c.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed loading cache file %s: %w", filePath, err)
	}

	return c, nil
}

func (c *SelectorCache) load() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := os.ReadFile(c.filePath)
	if err != nil {
		return err
	}

	var root struct {
		Domains map[string]map[string]CachedSelector `json:"domains"`
	}

	if err := json.Unmarshal(data, &root); err != nil {
		return err
	}

	if root.Domains != nil {
		c.Data = root.Domains
	}
	return nil
}

// Save commits cache state atomically to disk.
func (c *SelectorCache) Save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	root := struct {
		Version   string                               `json:"version"`
		UpdatedAt time.Time                            `json:"updatedAt"`
		Domains   map[string]map[string]CachedSelector `json:"domains"`
	}{
		Version:   "1.0",
		UpdatedAt: time.Now().UTC(),
		Domains:   c.Data,
	}

	bytes, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(c.filePath)
	if dir != "" && dir != "." {
		_ = os.MkdirAll(dir, 0755)
	}

	tmpFile := c.filePath + ".tmp"
	if err := os.WriteFile(tmpFile, bytes, 0644); err != nil {
		return err
	}

	return os.Rename(tmpFile, c.filePath)
}

// ExtractDomain normalizes a URL string into a domain key.
func ExtractDomain(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return "default"
	}
	return u.Host
}

// Get retrieves a cached selector for a given domain and field.
func (c *SelectorCache) Get(domain, field string) (CachedSelector, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if fields, ok := c.Data[domain]; ok {
		if sel, found := fields[field]; found {
			return sel, true
		}
	}
	return CachedSelector{}, false
}

// Set updates or stores a selector for a given domain and field.
func (c *SelectorCache) Set(domain, field string, sel CachedSelector) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.Data[domain]; !ok {
		c.Data[domain] = make(map[string]CachedSelector)
	}

	sel.UpdatedAt = time.Now().UTC()
	c.Data[domain][field] = sel
}

// RecordSuccess updates metrics when an extraction succeeds on a selector.
func (c *SelectorCache) RecordSuccess(domain, field string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if fields, ok := c.Data[domain]; ok {
		if sel, found := fields[field]; found {
			sel.LastSuccess = time.Now().UTC()
			sel.Failures = 0
			c.Data[domain][field] = sel
		}
	}
}

// RecordFailure increments failure count for a cached selector.
func (c *SelectorCache) RecordFailure(domain, field string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if fields, ok := c.Data[domain]; ok {
		if sel, found := fields[field]; found {
			sel.Failures++
			c.Data[domain][field] = sel
		}
	}
}
