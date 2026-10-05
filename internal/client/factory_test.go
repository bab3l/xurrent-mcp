package client

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigFromEnv_Full(t *testing.T) {
	t.Setenv("XURRENT_TOKEN", "tok-abc123")
	t.Setenv("XURRENT_ACCOUNT", "my-account")
	t.Setenv("XURRENT_API_BASE", "https://api.example.com")
	t.Setenv("XURRENT_LANGUAGE", "en-US")
	t.Setenv("XURRENT_ALLOW_MUTATIONS", "1")

	cfg, err := ConfigFromEnv()
	require.NoError(t, err)
	require.Equal(t, "tok-abc123", cfg.Token)
	require.Equal(t, "my-account", cfg.Account)
	require.Equal(t, "https://api.example.com", cfg.APIBase)
	require.Equal(t, "en-US", cfg.Language)
	require.True(t, cfg.AllowMutations)
}

func TestConfigFromEnv_Minimal(t *testing.T) {
	t.Setenv("XURRENT_TOKEN", "tok")
	t.Setenv("XURRENT_ACCOUNT", "acct")

	cfg, err := ConfigFromEnv()
	require.NoError(t, err)
	require.Equal(t, "tok", cfg.Token)
	require.Equal(t, "acct", cfg.Account)
	require.Equal(t, "", cfg.APIBase)
	require.False(t, cfg.AllowMutations)
}

func TestConfigFromEnv_MissingToken(t *testing.T) {
	t.Setenv("XURRENT_TOKEN", "")
	t.Setenv("XURRENT_ACCOUNT", "acct")
	_, err := ConfigFromEnv()
	require.Error(t, err)
	require.Contains(t, err.Error(), "XURRENT_TOKEN")
}

func TestConfigFromEnv_MissingAccount(t *testing.T) {
	t.Setenv("XURRENT_TOKEN", "tok")
	t.Setenv("XURRENT_ACCOUNT", "")
	_, err := ConfigFromEnv()
	require.Error(t, err)
	require.Contains(t, err.Error(), "XURRENT_ACCOUNT")
}

func TestConfigFromEnv_AllowMutationsFalse(t *testing.T) {
	t.Setenv("XURRENT_TOKEN", "tok")
	t.Setenv("XURRENT_ACCOUNT", "acct")
	t.Setenv("XURRENT_ALLOW_MUTATIONS", "0")
	cfg, err := ConfigFromEnv()
	require.NoError(t, err)
	require.False(t, cfg.AllowMutations)
}

func TestConfigFromEnv_AllowMutationsNotSet(t *testing.T) {
	t.Setenv("XURRENT_TOKEN", "tok")
	t.Setenv("XURRENT_ACCOUNT", "acct")
	os.Unsetenv("XURRENT_ALLOW_MUTATIONS")
	cfg, err := ConfigFromEnv()
	require.NoError(t, err)
	require.False(t, cfg.AllowMutations)
}

func TestConfigFromEnv_TrimsWhitespace(t *testing.T) {
	t.Setenv("XURRENT_TOKEN", "  tok  ")
	t.Setenv("XURRENT_ACCOUNT", "  acct  ")
	cfg, err := ConfigFromEnv()
	require.NoError(t, err)
	require.Equal(t, "tok", cfg.Token)
	require.Equal(t, "acct", cfg.Account)
}

func TestConfig_AccountID(t *testing.T) {
	cfg := &EnvConfig{Account: "my-account"}
	require.Equal(t, "my-account", cfg.AccountID())
}

func TestConfigFromEnv_APIBasedTrimmed(t *testing.T) {
	t.Setenv("XURRENT_TOKEN", "tok")
	t.Setenv("XURRENT_ACCOUNT", "acct")
	t.Setenv("XURRENT_API_BASE", "  https://api.example.com/  ")
	cfg, err := ConfigFromEnv()
	require.NoError(t, err)
	require.Equal(t, "https://api.example.com/", cfg.APIBase)
}

func TestNewAPIClient_NoEnv(t *testing.T) {
	// Clear env
	for _, k := range []string{"XURRENT_TOKEN", "XURRENT_ACCOUNT"} {
		t.Setenv(k, "")
	}
	_, err := NewAPIClient(nil)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "XURRENT_TOKEN") || strings.Contains(err.Error(), "must be set"))
}

func TestNewAPIClient_FromCfg(t *testing.T) {
	cfg := &EnvConfig{
		Token:   "test-tok",
		Account: "test-acct",
	}
	client, err := NewAPIClient(cfg)
	require.NoError(t, err)
	require.NotNil(t, client)
}

func TestNewAPIClient_WithLanguage(t *testing.T) {
	cfg := &EnvConfig{
		Token:    "test-tok",
		Account:  "test-acct",
		Language: "nl",
	}
	client, err := NewAPIClient(cfg)
	require.NoError(t, err)
	require.NotNil(t, client)
}

func TestNewAPIClient_WithCustomBase(t *testing.T) {
	cfg := &EnvConfig{
		Token:   "test-tok",
		Account: "test-acct",
		APIBase: "https://api.custom.com",
	}
	client, err := NewAPIClient(cfg)
	require.NoError(t, err)
	require.NotNil(t, client)
}
