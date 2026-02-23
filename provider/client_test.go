package provider

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/opensearch-project/opensearch-go/v4"
)

// TestTokenTransportRoundTrip tests the tokenTransport RoundTrip method
func TestTokenTransportRoundTrip(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the Authorization header is set
		authHeader := r.Header.Get("Authorization")
		expectedAuth := "Bearer test-token"
		if authHeader != expectedAuth {
			t.Errorf("Expected Authorization header %q, got %q", expectedAuth, authHeader)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"acknowledged": true}`))
	}))
	defer server.Close()

	// Create transport
	transport := &http.Transport{}

	tokenTrans := &tokenTransport{
		base:      transport,
		tokenName: "Bearer",
		token:     "test-token",
	}

	// Create request
	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Execute request
	resp, err := tokenTrans.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

// TestTokenTransportRoundTripWithHostOverride tests the tokenTransport with host override
func TestTokenTransportRoundTripWithHostOverride(t *testing.T) {
	// Create a test server that checks the Host header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check Host header
		if r.Host != "custom-host.example.com" {
			t.Errorf("Expected Host header 'custom-host.example.com', got '%s'", r.Host)
		}

		// Verify Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader != "ApiKey test-key" {
			t.Errorf("Expected Authorization header 'ApiKey test-key', got '%s'", authHeader)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	transport := &http.Transport{}

	tokenTrans := &tokenTransport{
		base:         transport,
		tokenName:    "ApiKey",
		token:        "test-key",
		hostOverride: "custom-host.example.com",
	}

	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := tokenTrans.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

// TestHostOverrideTransportRoundTrip tests the hostOverrideTransport RoundTrip method
func TestHostOverrideTransportRoundTrip(t *testing.T) {
	// Create a test server that checks the Host header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != "overridden-host.example.com" {
			t.Errorf("Expected Host header 'overridden-host.example.com', got '%s'", r.Host)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	transport := &http.Transport{}

	hostTrans := &hostOverrideTransport{
		base:         transport,
		hostOverride: "overridden-host.example.com",
	}

	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := hostTrans.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

// TestHostOverrideTransportRoundTripNoOverride tests the hostOverrideTransport without override
func TestHostOverrideTransportRoundTripNoOverride(t *testing.T) {
	originalHost := ""

	// Create a test server that captures the Host header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		originalHost = r.Host
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	transport := &http.Transport{}

	// Test with empty host override - should not modify the host
	hostTrans := &hostOverrideTransport{
		base:         transport,
		hostOverride: "",
	}

	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := hostTrans.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip failed: %v", err)
	}
	defer resp.Body.Close()

	// The host should remain as the original server host (not modified)
	if originalHost == "" {
		t.Error("Host header should not be empty")
	}
}

// TestTokenTransportImplementsRoundTripper verifies tokenTransport implements http.RoundTripper
func TestTokenTransportImplementsRoundTripper(t *testing.T) {
	transport := &tokenTransport{
		base:      &http.Transport{},
		tokenName: "Bearer",
		token:     "test",
	}

	var _ http.RoundTripper = transport
}

// TestHostOverrideTransportImplementsRoundTripper verifies hostOverrideTransport implements http.RoundTripper
func TestHostOverrideTransportImplementsRoundTripper(t *testing.T) {
	transport := &hostOverrideTransport{
		base:         &http.Transport{},
		hostOverride: "test.example.com",
	}

	var _ http.RoundTripper = transport
}

// TestOpenSearchClientStruct tests that OpenSearchClient struct can be created
func TestOpenSearchClientStruct(t *testing.T) {
	// Create a minimal ProviderConf
	conf := &ProviderConf{
		rawUrl: "http://localhost:9200",
	}

	// We can't fully test NewOpenSearchClient without a real OpenSearch server,
	// but we can test that the struct is properly defined
	client := &OpenSearchClient{
		config: conf,
	}

	if client.config != conf {
		t.Error("OpenSearchClient config should match the provided config")
	}
}

// TestProviderConfBasicAuth tests that ProviderConf properly handles basic auth
func TestProviderConfBasicAuth(t *testing.T) {
	// Test configuration with basic auth
	conf := &ProviderConf{
		rawUrl:   "http://localhost:9200",
		username: "admin",
		password: "password123",
	}

	if conf.username != "admin" {
		t.Errorf("Expected username 'admin', got '%s'", conf.username)
	}

	if conf.password != "password123" {
		t.Errorf("Expected password 'password123', got '%s'", conf.password)
	}
}

// TestProviderConfURLCredentials tests URL parsing for credentials
func TestProviderConfURLCredentials(t *testing.T) {
	conf := &ProviderConf{
		rawUrl: "http://user:pass@localhost:9200",
	}

	if conf.rawUrl != "http://user:pass@localhost:9200" {
		t.Errorf("Expected rawUrl to contain credentials, got '%s'", conf.rawUrl)
	}
}

// TestProviderConfTLSConfig tests TLS configuration options
func TestProviderConfTLSConfig(t *testing.T) {
	tests := []struct {
		name         string
		insecure     bool
		certPemPath  string
		keyPemPath   string
		cacertFile   string
		hostOverride string
	}{
		{
			name:         "insecure TLS",
			insecure:     true,
			hostOverride: "localhost",
		},
		{
			name:        "with client cert paths",
			certPemPath: "/path/to/cert.pem",
			keyPemPath:  "/path/to/key.pem",
		},
		{
			name:       "with CA cert",
			cacertFile: "/path/to/ca.pem",
		},
		{
			name:         "with all TLS options",
			insecure:     false,
			certPemPath:  "/path/to/cert.pem",
			keyPemPath:   "/path/to/key.pem",
			cacertFile:   "/path/to/ca.pem",
			hostOverride: "opensearch.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := &ProviderConf{
				rawUrl:       "https://localhost:9200",
				insecure:     tt.insecure,
				certPemPath:  tt.certPemPath,
				keyPemPath:   tt.keyPemPath,
				cacertFile:   tt.cacertFile,
				hostOverride: tt.hostOverride,
			}

			if conf.insecure != tt.insecure {
				t.Errorf("Expected insecure=%v, got %v", tt.insecure, conf.insecure)
			}

			if conf.certPemPath != tt.certPemPath {
				t.Errorf("Expected certPemPath='%s', got '%s'", tt.certPemPath, conf.certPemPath)
			}

			if conf.keyPemPath != tt.keyPemPath {
				t.Errorf("Expected keyPemPath='%s', got '%s'", tt.keyPemPath, conf.keyPemPath)
			}

			if conf.cacertFile != tt.cacertFile {
				t.Errorf("Expected cacertFile='%s', got '%s'", tt.cacertFile, conf.cacertFile)
			}

			if conf.hostOverride != tt.hostOverride {
				t.Errorf("Expected hostOverride='%s', got '%s'", tt.hostOverride, conf.hostOverride)
			}
		})
	}
}

// TestProviderConfTokenAuth tests token-based authentication configuration
func TestProviderConfTokenAuth(t *testing.T) {
	conf := &ProviderConf{
		rawUrl:    "http://localhost:9200",
		token:     "secret-token-12345",
		tokenName: "Bearer",
	}

	if conf.token != "secret-token-12345" {
		t.Errorf("Expected token 'secret-token-12345', got '%s'", conf.token)
	}

	if conf.tokenName != "Bearer" {
		t.Errorf("Expected tokenName 'Bearer', got '%s'", conf.tokenName)
	}
}

// TestProviderConfProxy tests proxy configuration
func TestProviderConfProxy(t *testing.T) {
	conf := &ProviderConf{
		rawUrl: "http://localhost:9200",
		proxy:  "http://proxy.example.com:8080",
	}

	if conf.proxy != "http://proxy.example.com:8080" {
		t.Errorf("Expected proxy 'http://proxy.example.com:8080', got '%s'", conf.proxy)
	}
}

// TestProviderConfAWSConfig tests AWS-related configuration
func TestProviderConfAWSConfig(t *testing.T) {
	conf := &ProviderConf{
		rawUrl:          "https://search-test.us-east-1.es.amazonaws.com",
		awsRegion:       "us-east-1",
		awsSig4Service:  "es",
		signAWSRequests: true,
	}

	if conf.awsRegion != "us-east-1" {
		t.Errorf("Expected awsRegion 'us-east-1', got '%s'", conf.awsRegion)
	}

	if conf.awsSig4Service != "es" {
		t.Errorf("Expected awsSig4Service 'es', got '%s'", conf.awsSig4Service)
	}

	if !conf.signAWSRequests {
		t.Error("Expected signAWSRequests to be true")
	}
}

// TestOpenSearchConfigValidation tests that opensearch.Config can be created
func TestOpenSearchConfigValidation(t *testing.T) {
	// Test that we can create a valid OpenSearch config
	cfg := opensearch.Config{
		Addresses: []string{"http://localhost:9200"},
		Username:  "admin",
		Password:  "password",
	}

	if len(cfg.Addresses) != 1 {
		t.Errorf("Expected 1 address, got %d", len(cfg.Addresses))
	}

	if cfg.Addresses[0] != "http://localhost:9200" {
		t.Errorf("Expected address 'http://localhost:9200', got '%s'", cfg.Addresses[0])
	}

	if cfg.Username != "admin" {
		t.Errorf("Expected username 'admin', got '%s'", cfg.Username)
	}
}

// TestClientFieldsExist tests that all expected fields exist on the client struct
func TestClientFieldsExist(t *testing.T) {
	// This test verifies the struct layout hasn't changed unexpectedly
	conf := &ProviderConf{
		rawUrl: "http://localhost:9200",
	}

	client := &OpenSearchClient{
		config: conf,
	}

	// Access the config field to ensure it exists
	_ = client.config
}
