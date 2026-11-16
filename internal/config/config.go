package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the complete runtime configuration.
type Config struct {
	Version  string         `yaml:"version"`
	Target   TargetConfig   `yaml:"target"`
	Schema   string         `yaml:"schema"`
	Pipeline PipelineConfig `yaml:"pipeline"`
	Vision   VisionConfig   `yaml:"vision"`
	Export   ExportConfig   `yaml:"export"`
	Server   ServerConfig   `yaml:"server"`
}

// TargetConfig specifies the destination page settings.
type TargetConfig struct {
	URL            string `yaml:"url"`
	WaitUntil      string `yaml:"wait_until"` // load, networkidle, domcontentloaded
	TimeoutSeconds int    `yaml:"timeout_seconds"`
	Stealth        bool   `yaml:"stealth"`
}

// PipelineConfig controls recovery tiers.
type PipelineConfig struct {
	Tier1Enabled bool   `yaml:"tier1_enabled"`
	Tier2Enabled bool   `yaml:"tier2_enabled"`
	Tier3Enabled bool   `yaml:"tier3_enabled"`
	MaxRetries   int    `yaml:"max_retries"`
	CacheFile    string `yaml:"cache_file"`
}

// VisionConfig configures VLM multimodal grounding and local heuristics.
type VisionConfig struct {
	Provider            string `yaml:"provider"` // gemini, openai, local-heuristic
	Model               string `yaml:"model"`
	APIKeyEnv           string `yaml:"api_key_env"`
	FallbackToHeuristic bool   `yaml:"fallback_to_heuristic"`
}

// ExportConfig defines data delivery destinations.
type ExportConfig struct {
	JSONL   JSONLConfig   `yaml:"jsonl"`
	Parquet ParquetConfig `yaml:"parquet"`
	Webhook WebhookConfig `yaml:"webhook"`
}

// JSONLConfig specifies streaming JSONL output.
type JSONLConfig struct {
	Enabled bool   `yaml:"enabled"`
	Path    string `yaml:"path"`
}

// ParquetConfig specifies columnar Parquet output.
type ParquetConfig struct {
	Enabled bool   `yaml:"enabled"`
	Path    string `yaml:"path"`
}

// WebhookConfig specifies HTTP push notifications.
type WebhookConfig struct {
	Enabled   bool   `yaml:"enabled"`
	URL       string `yaml:"url"`
	SecretEnv string `yaml:"secret_env"`
}

// ServerConfig configures the embedded React 19 studio.
type ServerConfig struct {
	Port         int  `yaml:"port"`
	EnableStudio bool `yaml:"enable_studio"`
}

// LoadConfig reads and parses a YAML configuration file.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed reading config file %s: %w", path, err)
	}

	cfg := &Config{
		Target: TargetConfig{
			WaitUntil:      "load",
			TimeoutSeconds: 30,
			Stealth:        true,
		},
		Pipeline: PipelineConfig{
			Tier1Enabled: true,
			Tier2Enabled: true,
			Tier3Enabled: true,
			MaxRetries:   3,
			CacheFile:    "selectors.cache.json",
		},
		Vision: VisionConfig{
			Provider:            "gemini",
			Model:               "gemini-2.5-flash",
			APIKeyEnv:           "GEMINI_API_KEY",
			FallbackToHeuristic: true,
		},
		Server: ServerConfig{
			Port:         8282,
			EnableStudio: true,
		},
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed parsing config yaml: %w", err)
	}

	return cfg, nil
}
