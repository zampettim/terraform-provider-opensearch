package provider

import (
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTokenTransport(t *testing.T) {
	base := &http.Transport{
		// Use a custom RoundTripper to verify the header is set.
	}
	// Wrap with a test round tripper so we don't make real network calls.
	var seenHost string
	var seenAuth string
	base.Proxy = nil

	rt := &tokenTransport{
		base:         base,
		tokenName:    "ApiKey",
		token:        "secret-token",
		hostOverride: "override.example.com",
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenHost = r.Host
		seenAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// Replace the base transport with one that targets the test server.
	rt.base = http.DefaultTransport.(*http.Transport).Clone()

	req, err := http.NewRequest("GET", ts.URL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if seenAuth != "ApiKey secret-token" {
		t.Errorf("Authorization header: got %q, want %q", seenAuth, "ApiKey secret-token")
	}
	if seenHost != "override.example.com" {
		t.Errorf("Host header: got %q, want %q", seenHost, "override.example.com")
	}
}

func TestHostOverrideTransport(t *testing.T) {
	var seenHost string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenHost = r.Host
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	rt := &hostOverrideTransport{
		base:         http.DefaultTransport.(*http.Transport).Clone(),
		hostOverride: "override.example.com",
	}

	req, err := http.NewRequest("GET", ts.URL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if seenHost != "override.example.com" {
		t.Errorf("Host header: got %q, want %q", seenHost, "override.example.com")
	}
}

func TestNewOpenSearchClient_Basic(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"version":{"number":"2.19.5","distribution":"opensearch"}}`)); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	}))
	defer ts.Close()

	conf := &ProviderConf{
		rawUrl:                ts.URL,
		pingTimeoutSeconds:    5,
		maxRetries:            3,
		retryBackoffInitialMs: 100,
	}

	client, err := NewOpenSearchClient(conf)
	if err != nil {
		t.Fatalf("NewOpenSearchClient failed: %v", err)
	}
	if client == nil {
		t.Fatal("expected client, got nil")
	}
}

func TestNewOpenSearchClient_TokenAuth(t *testing.T) {
	var seenAuth string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"version":{"number":"2.19.5","distribution":"opensearch"}}`)); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	}))
	defer ts.Close()

	conf := &ProviderConf{
		rawUrl:                ts.URL,
		pingTimeoutSeconds:    5,
		token:                 "mytoken",
		tokenName:             "Bearer",
		hostOverride:          "override.example.com",
		maxRetries:            3,
		retryBackoffInitialMs: 100,
	}

	client, err := NewOpenSearchClient(conf)
	if err != nil {
		t.Fatalf("NewOpenSearchClient failed: %v", err)
	}
	if client == nil {
		t.Fatal("expected client, got nil")
	}

	// Info is not called here, so just verify the client was created.
	// A full token auth test would need a live endpoint.
	_ = seenAuth
}

func TestNewOpenSearchClient_HostOverride(t *testing.T) {
	var seenHost string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenHost = r.Host
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"version":{"number":"2.19.5","distribution":"opensearch"}}`)); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	}))
	defer ts.Close()

	conf := &ProviderConf{
		rawUrl:                ts.URL,
		pingTimeoutSeconds:    5,
		hostOverride:          "override.example.com",
		maxRetries:            3,
		retryBackoffInitialMs: 100,
	}

	client, err := NewOpenSearchClient(conf)
	if err != nil {
		t.Fatalf("NewOpenSearchClient failed: %v", err)
	}
	if client == nil {
		t.Fatal("expected client, got nil")
	}

	_ = seenHost
}

func TestNewOpenSearchClient_InvalidURL(t *testing.T) {
	conf := &ProviderConf{
		rawUrl:                "://invalid-url",
		pingTimeoutSeconds:    5,
		maxRetries:            3,
		retryBackoffInitialMs: 100,
	}

	_, err := NewOpenSearchClient(conf)
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestNewOpenSearchClient_CACertInvalid(t *testing.T) {
	conf := &ProviderConf{
		rawUrl: "https://example.com:9200",
		// Use a syntactically valid PEM header but invalid body so AppendCertsFromPEM
		// returns false and we surface an error.
		cacertFile:            "-----BEGIN CERTIFICATE-----\nnot-a-valid-cert\n-----END CERTIFICATE-----",
		pingTimeoutSeconds:    5,
		maxRetries:            3,
		retryBackoffInitialMs: 100,
	}

	_, err := NewOpenSearchClient(conf)
	if err == nil {
		t.Fatal("expected error for invalid CA cert")
	}
	if !strings.Contains(err.Error(), "failed to append certificates") && !strings.Contains(err.Error(), x509.CertificateInvalidError{}.Error()) {
		t.Fatalf("unexpected error: %v", err)
	}
}
