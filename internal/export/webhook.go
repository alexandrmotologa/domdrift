package export

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// WebhookClient dispatches extraction records with HMAC-SHA256 signatures.
type WebhookClient struct {
	url        string
	secret     string
	httpClient *http.Client
	maxRetries int
}

// NewWebhookClient initializes a webhook dispatcher.
func NewWebhookClient(url, secret string) *WebhookClient {
	return &WebhookClient{
		url:    url,
		secret: secret,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		maxRetries: 3,
	}
}

// Dispatch sends a payload with exponential retry and HMAC signature.
func (c *WebhookClient) Dispatch(ctx context.Context, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed marshaling webhook payload: %w", err)
	}

	var signature string
	if c.secret != "" {
		mac := hmac.New(sha256.New, []byte(c.secret))
		mac.Write(data)
		signature = hex.EncodeToString(mac.Sum(nil))
	}

	var lastErr error
	backoff := 200 * time.Millisecond

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
				backoff *= 2
			}
		}

		req, err := http.NewRequestWithContext(ctx, "POST", c.url, bytes.NewReader(data))
		if err != nil {
			return err
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "domdrift/1.0")
		if signature != "" {
			req.Header.Set("X-Domdrift-Signature", "sha256="+signature)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		_ = resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}

		lastErr = fmt.Errorf("webhook endpoint returned status %d", resp.StatusCode)
	}

	return fmt.Errorf("webhook dispatch failed after %d attempts: %w", c.maxRetries, lastErr)
}
