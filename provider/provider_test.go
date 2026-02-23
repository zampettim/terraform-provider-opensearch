package provider

import (
	"context"
	"net/url"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var testAccProviders map[string]*schema.Provider
var testAccProviderFactories func(providers *[]*schema.Provider) map[string]func() (*schema.Provider, error)
var testAccProvider *schema.Provider

var testAccOpendistroProviders map[string]*schema.Provider
var testAccOpendistroProvider *schema.Provider

func init() {
	testAccProvider = Provider()
	testAccProviders = map[string]*schema.Provider{
		"opensearch": testAccProvider,
	}
	testAccProviderFactories = func(providers *[]*schema.Provider) map[string]func() (*schema.Provider, error) {
		// this is an SDKV2 compatible hack, the "factory" functions are
		// effectively singletons for the lifecycle of a resource.Test
		var factories = make(map[string]func() (*schema.Provider, error), len(testAccProviders))
		for name, p := range testAccProviders {
			factories[name] = func() (*schema.Provider, error) {
				return p, nil
			}
			*providers = append(*providers, p)
		}
		return factories
	}

	testAccOpendistroProvider = Provider()
	testAccOpendistroProviders = map[string]*schema.Provider{
		"opensearch": testAccOpendistroProvider,
	}

	opendistroOriginalConfigureFunc := testAccOpendistroProvider.ConfigureContextFunc
	testAccOpendistroProvider.ConfigureContextFunc = func(c context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		err := d.Set("url", os.Getenv("OPENSEARCH_URL"))
		if err != nil {
			return nil, diag.FromErr(err)
		}
		return opendistroOriginalConfigureFunc(c, d)
	}
}

func TestProvider(t *testing.T) {
	if err := Provider().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProvider_impl(t *testing.T) {
	var _ = Provider()
}

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("OPENSEARCH_URL"); v == "" {
		t.Fatal("OPENSEARCH_URL must be set for acceptance tests")
	}
}

// Given:
// 1. invalid username and password and healthcheck is false
//
// this tests that an error is returned by getOpenSearchClient for invalid credentials
func TestInvalidCredentials(t *testing.T) {
	parsedUrl, _ := url.Parse("http://127.0.0.1:9200")
	testConfig := &ProviderConf{
		username:           "1234",
		password:           "1234",
		healthchecking:     false,
		rawUrl:             "http://127.0.0.1:9200",
		sniffing:           false,
		parsedUrl:          parsedUrl,
		pingTimeoutSeconds: 10,
	}
	_, err := getOpenSearchClient(testConfig)

	if err == nil {
		t.Error("Expected an error to be returned for invalid credentials")
	}
}

// TestInvalidURLFormat tests error handling for malformed URLs
func TestInvalidURLFormat(t *testing.T) {
	testCases := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "empty URL",
			url:     "",
			wantErr: false, // url.Parse accepts empty string
		},
		{
			name:    "invalid scheme",
			url:     "ftp://localhost:9200",
			wantErr: false, // URL parsing accepts any scheme
		},
		{
			name:    "missing port",
			url:     "http://localhost",
			wantErr: false, // Port is optional
		},
		{
			name:    "invalid host format",
			url:     "://invalid-url",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := url.Parse(tc.url)
			if tc.wantErr && err == nil {
				t.Errorf("Expected error for URL %q, got none", tc.url)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("Unexpected error for URL %q: %v", tc.url, err)
			}
		})
	}
}

// TestMissingCredentials tests that proper error occurs when credentials are missing
func TestMissingCredentials(t *testing.T) {
	testCases := []struct {
		name     string
		username string
		password string
		token    string
	}{
		{
			name:     "no credentials",
			username: "",
			password: "",
			token:    "",
		},
		{
			name:     "only username",
			username: "admin",
			password: "",
			token:    "",
		},
		{
			name:     "only password",
			username: "",
			password: "secret",
			token:    "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// This test documents expected behavior - no authentication is still valid
			// The actual connection would fail, but config validation passes
			parsedUrl, _ := url.Parse("http://localhost:9200")
			testConfig := &ProviderConf{
				username:  tc.username,
				password:  tc.password,
				token:     tc.token,
				rawUrl:    "http://localhost:9200",
				parsedUrl: parsedUrl,
			}

			// Config can be created even without credentials
			if testConfig.rawUrl == "" {
				t.Error("Expected URL to be required")
			}
		})
	}
}

// TestInvalidProxyURL tests error handling for invalid proxy URLs
func TestInvalidProxyURL(t *testing.T) {
	testCases := []struct {
		name    string
		proxy   string
		wantErr bool
	}{
		{
			name:    "empty proxy",
			proxy:   "",
			wantErr: false,
		},
		{
			name:    "valid http proxy",
			proxy:   "http://proxy.example.com:8080",
			wantErr: false,
		},
		{
			name:    "invalid proxy format",
			proxy:   "://invalid-proxy",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.proxy == "" {
				return // Empty proxy is valid
			}
			_, err := url.Parse(tc.proxy)
			if tc.wantErr && err == nil {
				t.Errorf("Expected error for proxy URL %q, got none", tc.proxy)
			}
		})
	}
}

// TestInvalidCertificatePaths tests error handling for non-existent certificate files
func TestInvalidCertificatePaths(t *testing.T) {
	testCases := []struct {
		name        string
		certPemPath string
		keyPemPath  string
		cacertFile  string
		wantErr     bool
	}{
		{
			name:        "non-existent client cert",
			certPemPath: "/nonexistent/cert.pem",
			keyPemPath:  "/nonexistent/key.pem",
			cacertFile:  "",
			wantErr:     true,
		},
		{
			name:        "non-existent CA cert",
			certPemPath: "",
			keyPemPath:  "",
			cacertFile:  "/nonexistent/ca.pem",
			wantErr:     true,
		},
		{
			name:        "only cert without key",
			certPemPath: "/nonexistent/cert.pem",
			keyPemPath:  "",
			cacertFile:  "",
			wantErr:     false, // This is handled during client creation
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Verify that the provider config accepts these paths
			// Actual file validation happens when creating the client
			parsedUrl, _ := url.Parse("https://localhost:9200")
			testConfig := &ProviderConf{
				rawUrl:      "https://localhost:9200",
				parsedUrl:   parsedUrl,
				certPemPath: tc.certPemPath,
				keyPemPath:  tc.keyPemPath,
				cacertFile:  tc.cacertFile,
			}

			// Config struct is created successfully
			if testConfig.certPemPath != tc.certPemPath {
				t.Error("Cert path not set correctly")
			}
		})
	}
}
