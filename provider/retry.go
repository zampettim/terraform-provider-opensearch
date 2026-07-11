package provider

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// retryOptions configures the behavior of doRequestWithRetry.
type retryOptions struct {
	maxRetries      int
	backoffInitial  time.Duration
	retryableStatus map[int]struct{}
	refreshOn409    func() error
	resourceName    string
}

// defaultRetryOptions returns retry options consistent with the previous behavior
// of the provider: three attempts, 100ms linear backoff, retry on 409 and 500.
func defaultRetryOptions(conf *ProviderConf, resourceName string) retryOptions {
	maxRetries := 3
	if conf != nil && conf.maxRetries > 0 {
		maxRetries = conf.maxRetries
	}
	backoff := 100 * time.Millisecond
	if conf != nil && conf.retryBackoffInitialMs > 0 {
		backoff = time.Duration(conf.retryBackoffInitialMs) * time.Millisecond
	}

	return retryOptions{
		maxRetries:     maxRetries,
		backoffInitial: backoff,
		retryableStatus: map[int]struct{}{
			http.StatusConflict:           {},
			http.StatusInternalServerError: {},
		},
		resourceName: resourceName,
	}
}

// securityRetryOptions returns retry options tuned for the security plugin APIs.
// It uses up to five attempts and refreshes internal config state on 409 conflicts.
// Security resources write to a single shared config document, so they need more
// attempts than typical resources to tolerate concurrent writes.
func securityRetryOptions(conf *ProviderConf, resourceName string, refreshOn409 func() error) retryOptions {
	maxRetries := 5
	if conf != nil && conf.maxRetries > maxRetries {
		maxRetries = conf.maxRetries
	}
	backoff := 200 * time.Millisecond
	if conf != nil && conf.retryBackoffInitialMs > 0 {
		backoff = time.Duration(conf.retryBackoffInitialMs) * time.Millisecond
	}

	return retryOptions{
		maxRetries:     maxRetries,
		backoffInitial: backoff,
		retryableStatus: map[int]struct{}{
			http.StatusConflict:           {},
			http.StatusInternalServerError: {},
		},
		refreshOn409: refreshOn409,
		resourceName: resourceName,
	}
}

// ismRetryOptions returns retry options tuned for ISM and SM APIs. It retries
// only on 409 conflicts because these endpoints are idempotent on conflict.
func ismRetryOptions(conf *ProviderConf, resourceName string) retryOptions {
	opts := defaultRetryOptions(conf, resourceName)
	opts.retryableStatus = map[int]struct{}{
		http.StatusConflict: {},
	}
	return opts
}

// auditRetryOptions returns retry options for the audit config API. It retries
// only on 500 internal server errors, matching the original behavior.
func auditRetryOptions(conf *ProviderConf) retryOptions {
	opts := defaultRetryOptions(conf, "audit_config")
	opts.retryableStatus = map[int]struct{}{
		http.StatusInternalServerError: {},
	}
	return opts
}

// doRequestWithRetry executes the provided request builder and performs a
// retry loop around the resulting HTTP request. The request builder must be
// passed because HTTP request bodies cannot be rewound; a fresh request must
// be built for every attempt.
//
// On success it returns the response and no error. The caller is responsible
// for closing resp.Body.
// On failure it returns the last error or a non-retryable HTTP error.
func doRequestWithRetry(opts retryOptions, requestBuilder func() (*http.Request, error), perform func(*http.Request) (*http.Response, error)) (*http.Response, error) {
	var resp *http.Response
	var lastErr error

	for attempt := 0; attempt < opts.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff starting from the configured initial duration.
			backoff := opts.backoffInitial * time.Duration(1<<attempt)
			time.Sleep(backoff)

			if opts.refreshOn409 != nil && resp != nil && resp.StatusCode == http.StatusConflict {
				log.Printf("[INFO] retrying %s after conflict, refreshing state", opts.resourceName)
				if err := opts.refreshOn409(); err != nil {
					log.Printf("[WARN] failed to refresh %s state during conflict retry: %v", opts.resourceName, err)
				}
			}
		}

		req, err := requestBuilder()
		if err != nil {
			return nil, fmt.Errorf("error building request: %w", err)
		}

		resp, lastErr = perform(req)
		if lastErr == nil {
			if _, retryable := opts.retryableStatus[resp.StatusCode]; !retryable {
				return resp, nil
			}
			// Drain and close the body before retrying so the connection can be reused.
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		} else {
			log.Printf("[WARN] request attempt %d/%d for %s failed: %v", attempt+1, opts.maxRetries, opts.resourceName, lastErr)
		}
	}

	if lastErr != nil {
		return resp, fmt.Errorf("error %s: %w", opts.resourceName, lastErr)
	}
	if resp != nil {
		return resp, nil
	}
	return nil, fmt.Errorf("exhausted retries for %s", opts.resourceName)
}
