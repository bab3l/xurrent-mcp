// Package client provides a factory for creating configured go-xurrent API clients
// from environment variables, with rate limiting and optional caching.
package client

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	openapiclient "github.com/bab3l/go-xurrent"
	"github.com/xurrent/xurrent-mcp/internal/middleware"
)

// EnvConfig holds configuration parsed from the environment.
type EnvConfig struct {
	Token          string
	Account        string
	APIBase        string
	Language       string
	AllowMutations bool
	WebDomain      string // Xurrent web UI domain: "xurrent.com" (default) or "4me.com" (legacy)
}

// ConfigFromEnv reads the standard xurrent-mcp environment variables.
// Required: XURRENT_TOKEN, XURRENT_ACCOUNT
// Optional: XURRENT_API_BASE, XURRENT_LANGUAGE, XURRENT_ALLOW_MUTATIONS
func ConfigFromEnv() (*EnvConfig, error) {
	token := strings.TrimSpace(os.Getenv("XURRENT_TOKEN"))
	account := strings.TrimSpace(os.Getenv("XURRENT_ACCOUNT"))
	if token == "" || account == "" {
		return nil, fmt.Errorf("client: XURRENT_TOKEN and XURRENT_ACCOUNT must be set")
	}
	return &EnvConfig{
		Token:          token,
		Account:        account,
		APIBase:        strings.TrimSpace(os.Getenv("XURRENT_API_BASE")),
		Language:       strings.TrimSpace(os.Getenv("XURRENT_LANGUAGE")),
		AllowMutations: strings.TrimSpace(os.Getenv("XURRENT_ALLOW_MUTATIONS")) == "1",
		WebDomain:      webDomain(),
	}, nil
}

// NewAPIClient creates a go-xurrent API client from environment configuration.
// The HTTP client is configured with rate limiting based on XURRENT_HTTP_MIN_INTERVAL.
// The Authorization header is set as a default header (not per-request via context)
// so that it is always included.
func NewAPIClient(cfg *EnvConfig) (*openapiclient.APIClient, error) {
	if cfg == nil {
		var err error
		cfg, err = ConfigFromEnv()
		if err != nil {
			return nil, err
		}
	}
	apiCfg := openapiclient.NewConfiguration()
	apiCfg.AddDefaultHeader("Authorization", "Bearer "+cfg.Token)
	apiCfg.AddDefaultHeader("X-4me-Account", cfg.Account)
	if cfg.Language != "" {
		apiCfg.AddDefaultHeader("X-Xurrent-Language", cfg.Language)
	}
	if cfg.APIBase != "" {
		apiCfg.Servers = openapiclient.ServerConfigurations{
			{URL: strings.TrimRight(cfg.APIBase, "/"), Description: "env"},
		}
	}
	// Apply rate limiting.
	httpClient := &http.Client{}
	middleware.WithRateLimit(httpClient)
	apiCfg.HTTPClient = httpClient
	return openapiclient.NewAPIClient(apiCfg), nil
}

// AccountID returns the configured account ID (useful for cache keys and ownership checks).
func (c *EnvConfig) AccountID() string {
	return c.Account
}

// WebURL returns the base URL for the Xurrent web interface.
// Uses XURRENT_WEB_DOMAIN if set, otherwise defaults to xurrent.com.
// Legacy accounts should set XURRENT_WEB_DOMAIN=4me.com.
// Pattern: https://{account}.{domain}
func (c *EnvConfig) WebURL() string {
	domain := c.WebDomain
	if domain == "" {
		domain = "xurrent.com"
	}
	return "https://" + c.Account + "." + domain
}

// Link returns a clickable link to a specific entity in the Xurrent web UI.
func (c *EnvConfig) Link(entityPlural string, id int64) string {
	return fmt.Sprintf("%s/%s/%d", c.WebURL(), entityPlural, id)
}

// webDomain reads XURRENT_WEB_DOMAIN from environment, defaulting to "xurrent.com".
// Legacy accounts on the 4me platform should set XURRENT_WEB_DOMAIN=4me.com.
func webDomain() string {
	d := strings.TrimSpace(os.Getenv("XURRENT_WEB_DOMAIN"))
	if d == "" {
		d = "xurrent.com"
	}
	return d
}
