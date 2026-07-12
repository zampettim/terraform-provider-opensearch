package provider

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDefaultRetryOptions(t *testing.T) {
	cases := []struct {
		name     string
		conf     *ProviderConf
		expected retryOptions
	}{
		{
			name: "nil config uses defaults",
			conf: nil,
			expected: retryOptions{
				maxRetries:     3,
				backoffInitial: 100 * time.Millisecond,
				retryableStatus: map[int]struct{}{
					http.StatusConflict:            {},
					http.StatusInternalServerError: {},
				},
				resourceName: "role",
			},
		},
		{
			name: "custom max retries and backoff",
			conf: &ProviderConf{maxRetries: 7, retryBackoffInitialMs: 250},
			expected: retryOptions{
				maxRetries:     7,
				backoffInitial: 250 * time.Millisecond,
				retryableStatus: map[int]struct{}{
					http.StatusConflict:            {},
					http.StatusInternalServerError: {},
				},
				resourceName: "role",
			},
		},
		{
			name: "zero values fall back to defaults",
			conf: &ProviderConf{maxRetries: 0, retryBackoffInitialMs: 0},
			expected: retryOptions{
				maxRetries:     3,
				backoffInitial: 100 * time.Millisecond,
				retryableStatus: map[int]struct{}{
					http.StatusConflict:            {},
					http.StatusInternalServerError: {},
				},
				resourceName: "role",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			opts := defaultRetryOptions(tc.conf, "role")
			if opts.maxRetries != tc.expected.maxRetries {
				t.Errorf("maxRetries: got %d, want %d", opts.maxRetries, tc.expected.maxRetries)
			}
			if opts.backoffInitial != tc.expected.backoffInitial {
				t.Errorf("backoffInitial: got %v, want %v", opts.backoffInitial, tc.expected.backoffInitial)
			}
			if len(opts.retryableStatus) != len(tc.expected.retryableStatus) {
				t.Errorf("retryableStatus length: got %d, want %d", len(opts.retryableStatus), len(tc.expected.retryableStatus))
			}
			if opts.resourceName != tc.expected.resourceName {
				t.Errorf("resourceName: got %q, want %q", opts.resourceName, tc.expected.resourceName)
			}
		})
	}
}

func TestSecurityRetryOptions(t *testing.T) {
	refreshCalls := 0
	refresh := func() error {
		refreshCalls++
		return nil
	}

	cases := []struct {
		name            string
		conf            *ProviderConf
		expectedRetries int
	}{
		{name: "nil config uses 5", conf: nil, expectedRetries: 5},
		{name: "default provider value does not reduce security retries", conf: &ProviderConf{maxRetries: 3, retryBackoffInitialMs: 100}, expectedRetries: 5},
		{name: "higher provider value is respected", conf: &ProviderConf{maxRetries: 10, retryBackoffInitialMs: 100}, expectedRetries: 10},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			opts := securityRetryOptions(tc.conf, "user", refresh)
			if opts.maxRetries != tc.expectedRetries {
				t.Errorf("maxRetries: got %d, want %d", opts.maxRetries, tc.expectedRetries)
			}
			if opts.refreshOn409 == nil {
				t.Error("expected refreshOn409 to be set")
			}
		})
	}
}

func TestISMRetryOptions(t *testing.T) {
	opts := ismRetryOptions(&ProviderConf{maxRetries: 4, retryBackoffInitialMs: 150}, "ism")
	if opts.maxRetries != 4 {
		t.Errorf("maxRetries: got %d, want 4", opts.maxRetries)
	}
	if opts.backoffInitial != 150*time.Millisecond {
		t.Errorf("backoffInitial: got %v, want 150ms", opts.backoffInitial)
	}
	if _, ok := opts.retryableStatus[http.StatusConflict]; !ok {
		t.Error("expected 409 to be retryable")
	}
	if _, ok := opts.retryableStatus[http.StatusInternalServerError]; ok {
		t.Error("expected 500 not to be retryable for ISM")
	}
}

func TestAuditRetryOptions(t *testing.T) {
	opts := auditRetryOptions(&ProviderConf{maxRetries: 2, retryBackoffInitialMs: 50})
	if opts.maxRetries != 2 {
		t.Errorf("maxRetries: got %d, want 2", opts.maxRetries)
	}
	if opts.backoffInitial != 50*time.Millisecond {
		t.Errorf("backoffInitial: got %v, want 50ms", opts.backoffInitial)
	}
	if _, ok := opts.retryableStatus[http.StatusInternalServerError]; !ok {
		t.Error("expected 500 to be retryable")
	}
	if _, ok := opts.retryableStatus[http.StatusConflict]; ok {
		t.Error("expected 409 not to be retryable for audit")
	}
}

func TestDoRequestWithRetry_Success(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/", nil)
	opts := retryOptions{maxRetries: 3, backoffInitial: 1 * time.Millisecond, resourceName: "test"}

	resp, err := doRequestWithRetry(opts, func() (*http.Request, error) {
		return req, nil
	}, func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("ok"))}, nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestDoRequestWithRetry_RetryableStatusThenSuccess(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/", nil)
	opts := retryOptions{
		maxRetries:      3,
		backoffInitial:  1 * time.Millisecond,
		retryableStatus: map[int]struct{}{http.StatusConflict: {}},
		resourceName:    "test",
	}

	calls := 0
	resp, err := doRequestWithRetry(opts, func() (*http.Request, error) {
		return req, nil
	}, func(r *http.Request) (*http.Response, error) {
		calls++
		if calls < 3 {
			return &http.Response{StatusCode: http.StatusConflict, Body: io.NopCloser(strings.NewReader("conflict"))}, nil
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("ok"))}, nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 3 {
		t.Errorf("calls: got %d, want 3", calls)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestDoRequestWithRetry_ExhaustedRetries(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/", nil)
	opts := retryOptions{
		maxRetries:      2,
		backoffInitial:  1 * time.Millisecond,
		retryableStatus: map[int]struct{}{http.StatusConflict: {}},
		resourceName:    "test",
	}

	calls := 0
	resp, err := doRequestWithRetry(opts, func() (*http.Request, error) {
		return req, nil
	}, func(r *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: http.StatusConflict, Body: io.NopCloser(strings.NewReader("conflict"))}, nil
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if calls != 2 {
		t.Errorf("calls: got %d, want 2", calls)
	}
	if resp == nil || resp.StatusCode != http.StatusConflict {
		t.Errorf("expected last response to be returned, got %v", resp)
	}
}

func TestDoRequestWithRetry_NonRetryableStatus(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/", nil)
	opts := retryOptions{
		maxRetries:      3,
		backoffInitial:  1 * time.Millisecond,
		retryableStatus: map[int]struct{}{http.StatusConflict: {}},
		resourceName:    "test",
	}

	calls := 0
	resp, err := doRequestWithRetry(opts, func() (*http.Request, error) {
		return req, nil
	}, func(r *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: http.StatusBadRequest, Body: io.NopCloser(strings.NewReader("bad request"))}, nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Errorf("calls: got %d, want 1", calls)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status: got %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestDoRequestWithRetry_RequestBuilderError(t *testing.T) {
	opts := retryOptions{maxRetries: 3, backoffInitial: 1 * time.Millisecond, resourceName: "test"}

	_, err := doRequestWithRetry(opts, func() (*http.Request, error) {
		return nil, errors.New("boom")
	}, func(r *http.Request) (*http.Response, error) {
		t.Fatal("perform should not be called")
		return nil, nil
	})

	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected builder error, got %v", err)
	}
}

func TestDoRequestWithRetry_PerformErrorThenSuccess(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/", nil)
	opts := retryOptions{maxRetries: 3, backoffInitial: 1 * time.Millisecond, resourceName: "test"}

	calls := 0
	resp, err := doRequestWithRetry(opts, func() (*http.Request, error) {
		return req, nil
	}, func(r *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return nil, errors.New("network error")
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("ok"))}, nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 2 {
		t.Errorf("calls: got %d, want 2", calls)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestDoRequestWithRetry_RefreshOn409Called(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/", nil)
	refreshCalls := 0
	opts := retryOptions{
		maxRetries:      3,
		backoffInitial:  1 * time.Millisecond,
		retryableStatus: map[int]struct{}{http.StatusConflict: {}},
		refreshOn409: func() error {
			refreshCalls++
			return nil
		},
		resourceName: "test",
	}

	calls := 0
	_, err := doRequestWithRetry(opts, func() (*http.Request, error) {
		return req, nil
	}, func(r *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: http.StatusConflict, Body: io.NopCloser(strings.NewReader("conflict"))}, nil
	})

	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if refreshCalls != calls-1 {
		t.Errorf("refreshOn409 calls: got %d, want %d", refreshCalls, calls-1)
	}
}
