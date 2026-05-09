// Package provider provides Terraform provider for OpenSearch.
// This file contains migration utilities for transitioning from olivere/elastic to opensearch-go/v4.
package provider

import (
	"strings"
)

// isNotFound checks if an error represents a "not found" response
// This replaces elastic7.IsNotFound(err)
func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "404") ||
		strings.Contains(errStr, "not found") ||
		strings.Contains(errStr, "Not Found") ||
		// ISM policy mapping returns 400 with "no documents to get" when index doesn't exist
		(strings.Contains(errStr, "400") && strings.Contains(errStr, "no documents to get"))
}
