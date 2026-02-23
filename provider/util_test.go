package provider

import (
	"hash/crc32"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// TestNormalizeChannelConfiguration tests the normalizeChannelConfiguration function
func TestNormalizeChannelConfiguration(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "removes timestamps",
			input: map[string]interface{}{
				"name":                 "test-channel",
				"last_updated_time_ms": 1234567890,
				"created_time_ms":      1234567800,
				"config": map[string]interface{}{
					"webhook": map[string]interface{}{
						"url": "https://example.com",
					},
				},
			},
			expected: map[string]interface{}{
				"name": "test-channel",
				"config": map[string]interface{}{
					"webhook": map[string]interface{}{
						"url": "https://example.com",
					},
				},
			},
		},
		{
			name: "no timestamps to remove",
			input: map[string]interface{}{
				"name": "test-channel",
				"config": map[string]interface{}{
					"slack": map[string]interface{}{
						"url": "https://hooks.slack.com",
					},
				},
			},
			expected: map[string]interface{}{
				"name": "test-channel",
				"config": map[string]interface{}{
					"slack": map[string]interface{}{
						"url": "https://hooks.slack.com",
					},
				},
			},
		},
		{
			name:     "empty map",
			input:    map[string]interface{}{},
			expected: map[string]interface{}{},
		},
		{
			name: "only timestamps",
			input: map[string]interface{}{
				"last_updated_time_ms": 1234567890,
				"created_time_ms":      1234567800,
			},
			expected: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy to avoid modifying the test case
			input := make(map[string]interface{})
			for k, v := range tt.input {
				input[k] = v
			}

			normalizeChannelConfiguration(input)

			if !reflect.DeepEqual(input, tt.expected) {
				t.Errorf("normalizeChannelConfiguration() = %v, want %v", input, tt.expected)
			}
		})
	}
}

// TestNormalizeMonitor tests the normalizeMonitor function
func TestNormalizeMonitor(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "removes metadata fields",
			input: map[string]interface{}{
				"name":             "test-monitor",
				"id":               "monitor-123",
				"last_update_time": 1234567890,
				"enabled_time":     1234567800,
				"schema_version":   1,
				"user": map[string]interface{}{
					"name": "admin",
				},
				"type": "monitor",
			},
			expected: map[string]interface{}{
				"name": "test-monitor",
				"type": "monitor",
			},
		},
		{
			name: "with triggers",
			input: map[string]interface{}{
				"name": "test-monitor",
				"id":   "monitor-123",
				"triggers": []interface{}{
					map[string]interface{}{
						"id":   "trigger-1",
						"name": "high-cpu",
						"actions": []interface{}{
							map[string]interface{}{
								"id":   "action-1",
								"name": "send-email",
							},
						},
					},
				},
			},
			expected: map[string]interface{}{
				"name": "test-monitor",
				"triggers": []interface{}{
					map[string]interface{}{
						"name": "high-cpu",
						"actions": []interface{}{
							map[string]interface{}{
								"name": "send-email",
							},
						},
					},
				},
			},
		},
		{
			name: "no triggers",
			input: map[string]interface{}{
				"name":           "test-monitor",
				"id":             "monitor-123",
				"schema_version": 1,
			},
			expected: map[string]interface{}{
				"name": "test-monitor",
			},
		},
		{
			name:     "empty map",
			input:    map[string]interface{}{},
			expected: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Deep copy for complex types
			input := deepCopyMap(tt.input)

			normalizeMonitor(input)

			if !reflect.DeepEqual(input, tt.expected) {
				t.Errorf("normalizeMonitor() = %v, want %v", input, tt.expected)
			}
		})
	}
}

// TestNormalizeMonitorTriggers tests the normalizeMonitorTriggers function
func TestNormalizeMonitorTriggers(t *testing.T) {
	tests := []struct {
		name     string
		input    []interface{}
		expected []interface{}
	}{
		{
			name: "removes trigger ids",
			input: []interface{}{
				map[string]interface{}{
					"id":   "trigger-1",
					"name": "high-cpu",
				},
				map[string]interface{}{
					"id":   "trigger-2",
					"name": "high-memory",
				},
			},
			expected: []interface{}{
				map[string]interface{}{
					"name": "high-cpu",
				},
				map[string]interface{}{
					"name": "high-memory",
				},
			},
		},
		{
			name: "with actions",
			input: []interface{}{
				map[string]interface{}{
					"id":   "trigger-1",
					"name": "high-cpu",
					"actions": []interface{}{
						map[string]interface{}{
							"id":   "action-1",
							"name": "send-email",
						},
					},
				},
			},
			expected: []interface{}{
				map[string]interface{}{
					"name": "high-cpu",
					"actions": []interface{}{
						map[string]interface{}{
							"name": "send-email",
						},
					},
				},
			},
		},
		{
			name:     "empty triggers",
			input:    []interface{}{},
			expected: []interface{}{},
		},
		{
			name: "non-map elements",
			input: []interface{}{
				"invalid-trigger",
				map[string]interface{}{
					"id":   "trigger-1",
					"name": "high-cpu",
				},
			},
			expected: []interface{}{
				"invalid-trigger",
				map[string]interface{}{
					"name": "high-cpu",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := deepCopySlice(tt.input)

			normalizeMonitorTriggers(input)

			if !reflect.DeepEqual(input, tt.expected) {
				t.Errorf("normalizeMonitorTriggers() = %v, want %v", input, tt.expected)
			}
		})
	}
}

// TestNormalizeMonitorTriggerActions tests the normalizeMonitorTriggerActions function
func TestNormalizeMonitorTriggerActions(t *testing.T) {
	tests := []struct {
		name     string
		input    []interface{}
		expected []interface{}
	}{
		{
			name: "removes action ids",
			input: []interface{}{
				map[string]interface{}{
					"id":   "action-1",
					"name": "send-email",
				},
				map[string]interface{}{
					"id":   "action-2",
					"name": "send-slack",
				},
			},
			expected: []interface{}{
				map[string]interface{}{
					"name": "send-email",
				},
				map[string]interface{}{
					"name": "send-slack",
				},
			},
		},
		{
			name:     "empty actions",
			input:    []interface{}{},
			expected: []interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := deepCopySlice(tt.input)

			normalizeMonitorTriggerActions(input)

			if !reflect.DeepEqual(input, tt.expected) {
				t.Errorf("normalizeMonitorTriggerActions() = %v, want %v", input, tt.expected)
			}
		})
	}
}

// TestNormalizePolicy tests the normalizePolicy function
func TestNormalizePolicy(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "removes metadata fields",
			input: map[string]interface{}{
				"policy_id":         "policy-123",
				"last_updated_time": 1234567890,
				"schema_version":    1,
				"description":       "Test policy",
				"default_state":     "hot",
			},
			expected: map[string]interface{}{
				"description":   "Test policy",
				"default_state": "hot",
			},
		},
		{
			name: "removes null error_notification",
			input: map[string]interface{}{
				"description":        "Test policy",
				"error_notification": nil,
				"default_state":      "hot",
			},
			expected: map[string]interface{}{
				"description":   "Test policy",
				"default_state": "hot",
			},
		},
		{
			name: "keeps non-null error_notification",
			input: map[string]interface{}{
				"description":   "Test policy",
				"default_state": "hot",
				"error_notification": map[string]interface{}{
					"destination": map[string]interface{}{
						"slack": map[string]interface{}{
							"url": "https://hooks.slack.com",
						},
					},
				},
			},
			expected: map[string]interface{}{
				"description":   "Test policy",
				"default_state": "hot",
				"error_notification": map[string]interface{}{
					"destination": map[string]interface{}{
						"slack": map[string]interface{}{
							"url": "https://hooks.slack.com",
						},
					},
				},
			},
		},
		{
			name: "removes null ism_template",
			input: map[string]interface{}{
				"description":   "Test policy",
				"default_state": "hot",
				"ism_template":  nil,
			},
			expected: map[string]interface{}{
				"description":   "Test policy",
				"default_state": "hot",
			},
		},
		{
			name: "handles ism_template as map",
			input: map[string]interface{}{
				"description":   "Test policy",
				"default_state": "hot",
				"ism_template": map[string]interface{}{
					"index_patterns":    []string{"log-*"},
					"priority":          100,
					"last_updated_time": 1234567890,
				},
			},
			expected: map[string]interface{}{
				"description":   "Test policy",
				"default_state": "hot",
				"ism_template": map[string]interface{}{
					"index_patterns": []string{"log-*"},
					"priority":       100,
				},
			},
		},
		{
			name: "handles ism_template as slice",
			input: map[string]interface{}{
				"description":   "Test policy",
				"default_state": "hot",
				"ism_template": []interface{}{
					map[string]interface{}{
						"index_patterns":    []string{"log-*"},
						"priority":          100,
						"last_updated_time": 1234567890,
					},
				},
			},
			expected: map[string]interface{}{
				"description":   "Test policy",
				"default_state": "hot",
				"ism_template": []interface{}{
					map[string]interface{}{
						"index_patterns": []string{"log-*"},
						"priority":       100,
					},
				},
			},
		},
		{
			name:     "empty map",
			input:    map[string]interface{}{},
			expected: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := deepCopyMap(tt.input)

			normalizePolicy(input)

			if !reflect.DeepEqual(input, tt.expected) {
				t.Errorf("normalizePolicy() = %v, want %v", input, tt.expected)
			}
		})
	}
}

// TestNormalizeIndexTemplate tests the normalizeIndexTemplate function
func TestNormalizeIndexTemplate(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "removes version",
			input: map[string]interface{}{
				"name":           "test-template",
				"version":        1,
				"index_patterns": []string{"logs-*"},
			},
			expected: map[string]interface{}{
				"name":           "test-template",
				"index_patterns": []string{"logs-*"},
			},
		},
		{
			name: "removes default order (0)",
			input: map[string]interface{}{
				"name":    "test-template",
				"order":   0.0,
				"version": 1,
			},
			expected: map[string]interface{}{
				"name": "test-template",
			},
		},
		{
			name: "keeps non-zero order",
			input: map[string]interface{}{
				"name":    "test-template",
				"order":   100.0,
				"version": 1,
			},
			expected: map[string]interface{}{
				"name":  "test-template",
				"order": 100.0,
			},
		},
		{
			name: "removes empty settings",
			input: map[string]interface{}{
				"name":     "test-template",
				"version":  1,
				"settings": map[string]interface{}{},
			},
			expected: map[string]interface{}{
				"name": "test-template",
			},
		},
		{
			name: "normalizes settings",
			input: map[string]interface{}{
				"name":    "test-template",
				"version": 1,
				"settings": map[string]interface{}{
					"number_of_shards": 1,
				},
			},
			expected: map[string]interface{}{
				"name": "test-template",
				"settings": map[string]interface{}{
					"index.number_of_shards": "1",
				},
			},
		},
		{
			name: "removes empty mappings",
			input: map[string]interface{}{
				"name":     "test-template",
				"version":  1,
				"mappings": map[string]interface{}{},
			},
			expected: map[string]interface{}{
				"name": "test-template",
			},
		},
		{
			name: "keeps non-empty mappings",
			input: map[string]interface{}{
				"name":    "test-template",
				"version": 1,
				"mappings": map[string]interface{}{
					"properties": map[string]interface{}{
						"message": map[string]interface{}{
							"type": "text",
						},
					},
				},
			},
			expected: map[string]interface{}{
				"name": "test-template",
				"mappings": map[string]interface{}{
					"properties": map[string]interface{}{
						"message": map[string]interface{}{
							"type": "text",
						},
					},
				},
			},
		},
		{
			name: "removes empty aliases",
			input: map[string]interface{}{
				"name":    "test-template",
				"version": 1,
				"aliases": map[string]interface{}{},
			},
			expected: map[string]interface{}{
				"name": "test-template",
			},
		},
		{
			name: "handles template.settings (ES 7.8+ format)",
			input: map[string]interface{}{
				"name":    "test-template",
				"version": 1,
				"template": map[string]interface{}{
					"settings": map[string]interface{}{
						"number_of_shards": 2,
					},
				},
			},
			expected: map[string]interface{}{
				"name": "test-template",
				"template": map[string]interface{}{
					"settings": map[string]interface{}{
						"index.number_of_shards": "2",
					},
				},
			},
		},
		{
			name:     "empty map",
			input:    map[string]interface{}{},
			expected: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := deepCopyMap(tt.input)

			normalizeIndexTemplate(input)

			if !reflect.DeepEqual(input, tt.expected) {
				t.Errorf("normalizeIndexTemplate() = %v, want %v", input, tt.expected)
			}
		})
	}
}

// TestNormalizeComposableIndexTemplate tests the normalizeComposableIndexTemplate function
func TestNormalizeComposableIndexTemplate(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "removes version",
			input: map[string]interface{}{
				"name":           "test-template",
				"version":        1,
				"index_patterns": []string{"logs-*"},
				"composed_of":    []string{"component1"},
				"priority":       500,
			},
			expected: map[string]interface{}{
				"name":           "test-template",
				"index_patterns": []string{"logs-*"},
				"composed_of":    []string{"component1"},
				"priority":       500,
			},
		},
		{
			name: "filters data_stream to only keep hidden",
			input: map[string]interface{}{
				"name":    "test-template",
				"version": 1,
				"data_stream": map[string]interface{}{
					"hidden":          true,
					"timestamp_field": "@timestamp",
					"extra_field":     "value",
				},
			},
			expected: map[string]interface{}{
				"name": "test-template",
				"data_stream": map[string]interface{}{
					"hidden": true,
				},
			},
		},
		{
			name: "normalizes template settings",
			input: map[string]interface{}{
				"name":    "test-template",
				"version": 1,
				"template": map[string]interface{}{
					"settings": map[string]interface{}{
						"number_of_shards": 3,
					},
				},
			},
			expected: map[string]interface{}{
				"name": "test-template",
				"template": map[string]interface{}{
					"settings": map[string]interface{}{
						"index.number_of_shards": "3",
					},
				},
			},
		},
		{
			name:     "empty map",
			input:    map[string]interface{}{},
			expected: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := deepCopyMap(tt.input)

			normalizeComposableIndexTemplate(input)

			if !reflect.DeepEqual(input, tt.expected) {
				t.Errorf("normalizeComposableIndexTemplate() = %v, want %v", input, tt.expected)
			}
		})
	}
}

// TestNormalizeComponentTemplate tests the normalizeComponentTemplate function
func TestNormalizeComponentTemplate(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "removes version",
			input: map[string]interface{}{
				"name":    "test-component",
				"version": 1,
				"template": map[string]interface{}{
					"settings": map[string]interface{}{
						"number_of_shards": 1,
					},
				},
			},
			expected: map[string]interface{}{
				"name": "test-component",
				"template": map[string]interface{}{
					"settings": map[string]interface{}{
						"index.number_of_shards": "1",
					},
				},
			},
		},
		{
			name: "normalizes settings in template",
			input: map[string]interface{}{
				"name":    "test-component",
				"version": 2,
				"template": map[string]interface{}{
					"settings": map[string]interface{}{
						"index": map[string]interface{}{
							"number_of_replicas": 0,
						},
					},
				},
			},
			expected: map[string]interface{}{
				"name": "test-component",
				"template": map[string]interface{}{
					"settings": map[string]interface{}{
						"index.number_of_replicas": "0",
					},
				},
			},
		},
		{
			name:     "empty map",
			input:    map[string]interface{}{},
			expected: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := deepCopyMap(tt.input)

			normalizeComponentTemplate(input)

			if !reflect.DeepEqual(input, tt.expected) {
				t.Errorf("normalizeComponentTemplate() = %v, want %v", input, tt.expected)
			}
		})
	}
}

// TestNormalizeAnomalyDetection tests the normalizeAnomalyDetection function
func TestNormalizeAnomalyDetection(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "removes last_update_time",
			input: map[string]interface{}{
				"name":             "test-detector",
				"last_update_time": 1234567890,
				"description":      "Test detector",
			},
			expected: map[string]interface{}{
				"name":        "test-detector",
				"description": "Test detector",
			},
		},
		{
			name: "no last_update_time",
			input: map[string]interface{}{
				"name":        "test-detector",
				"description": "Test detector",
			},
			expected: map[string]interface{}{
				"name":        "test-detector",
				"description": "Test detector",
			},
		},
		{
			name:     "empty map",
			input:    map[string]interface{}{},
			expected: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := deepCopyMap(tt.input)

			normalizeAnomalyDetection(input)

			if !reflect.DeepEqual(input, tt.expected) {
				t.Errorf("normalizeAnomalyDetection() = %v, want %v", input, tt.expected)
			}
		})
	}
}

// TestNormalizedIndexSettings tests the normalizedIndexSettings function
func TestNormalizedIndexSettings(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "flattens and prefixes settings",
			input: map[string]interface{}{
				"number_of_shards":   1,
				"number_of_replicas": 0,
			},
			expected: map[string]interface{}{
				"index.number_of_shards":   "1",
				"index.number_of_replicas": "0",
			},
		},
		{
			name: "handles already prefixed settings",
			input: map[string]interface{}{
				"index.number_of_shards": 2,
			},
			expected: map[string]interface{}{
				"index.number_of_shards": "2",
			},
		},
		{
			name: "handles nested settings",
			input: map[string]interface{}{
				"index": map[string]interface{}{
					"number_of_shards": 3,
				},
			},
			expected: map[string]interface{}{
				"index.number_of_shards": "3",
			},
		},
		{
			name:     "empty settings",
			input:    map[string]interface{}{},
			expected: map[string]interface{}{},
		},
		{
			name: "handles mixed types",
			input: map[string]interface{}{
				"number_of_shards":  1,
				"refresh_interval":  "1s",
				"max_result_window": 10000,
				"blocks_read_only":  false,
			},
			expected: map[string]interface{}{
				"index.number_of_shards":  "1",
				"index.refresh_interval":  "1s",
				"index.max_result_window": "10000",
				"index.blocks_read_only":  "false",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizedIndexSettings(tt.input)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("normalizedIndexSettings() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestFlattenMap tests the flattenMap function
func TestFlattenMap(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "flattens nested map",
			input: map[string]interface{}{
				"index": map[string]interface{}{
					"number_of_shards":   1,
					"number_of_replicas": 0,
				},
			},
			expected: map[string]interface{}{
				"index.number_of_shards":   1,
				"index.number_of_replicas": 0,
			},
		},
		{
			name: "handles deeply nested map",
			input: map[string]interface{}{
				"a": map[string]interface{}{
					"b": map[string]interface{}{
						"c": "value",
					},
				},
			},
			expected: map[string]interface{}{
				"a.b.c": "value",
			},
		},
		{
			name: "handles flat map",
			input: map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
			expected: map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name:     "empty map",
			input:    map[string]interface{}{},
			expected: map[string]interface{}{},
		},
		{
			name: "handles mixed nested and flat",
			input: map[string]interface{}{
				"flat": "value",
				"nested": map[string]interface{}{
					"key": "nested-value",
				},
			},
			expected: map[string]interface{}{
				"flat":       "value",
				"nested.key": "nested-value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := flattenMap(tt.input)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("flattenMap() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestConcatStringSlice tests the concatStringSlice function
func TestConcatStringSlice(t *testing.T) {
	tests := []struct {
		name     string
		args     [][]string
		expected []string
	}{
		{
			name:     "concatenates multiple slices",
			args:     [][]string{{"a", "b"}, {"c", "d"}, {"e"}},
			expected: []string{"a", "b", "c", "d", "e"},
		},
		{
			name:     "single slice",
			args:     [][]string{{"x", "y", "z"}},
			expected: []string{"x", "y", "z"},
		},
		{
			name:     "empty slices",
			args:     [][]string{{}, {"a"}, {}},
			expected: []string{"a"},
		},
		{
			name:     "no slices",
			args:     [][]string{},
			expected: []string{},
		},
		{
			name:     "all empty slices",
			args:     [][]string{{}, {}, {}},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := concatStringSlice(tt.args...)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("concatStringSlice() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestContainsString tests the containsString function
func TestContainsString(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		needle   string
		expected bool
	}{
		{
			name:     "contains string",
			slice:    []string{"a", "b", "c"},
			needle:   "b",
			expected: true,
		},
		{
			name:     "does not contain string",
			slice:    []string{"a", "b", "c"},
			needle:   "d",
			expected: false,
		},
		{
			name:     "empty slice",
			slice:    []string{},
			needle:   "a",
			expected: false,
		},
		{
			name:     "empty string in slice",
			slice:    []string{"", "a", "b"},
			needle:   "",
			expected: true,
		},
		{
			name:     "case sensitive",
			slice:    []string{"A", "B", "C"},
			needle:   "b",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsString(tt.slice, tt.needle)

			if result != tt.expected {
				t.Errorf("containsString() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestFunctionallyEquivalentJSON tests the functionallyEquivalentJSON function
func TestFunctionallyEquivalentJSON(t *testing.T) {
	tests := []struct {
		name     string
		json1    string
		json2    string
		expected bool
	}{
		{
			name:     "identical JSON",
			json1:    `{"key": "value"}`,
			json2:    `{"key": "value"}`,
			expected: true,
		},
		{
			name:     "different key order",
			json1:    `{"a": 1, "b": 2}`,
			json2:    `{"b": 2, "a": 1}`,
			expected: true,
		},
		{
			name:     "different values",
			json1:    `{"key": "value1"}`,
			json2:    `{"key": "value2"}`,
			expected: false,
		},
		{
			name:     "different keys",
			json1:    `{"key1": "value"}`,
			json2:    `{"key2": "value"}`,
			expected: false,
		},
		{
			name:     "invalid JSON first",
			json1:    `{"key":}`,
			json2:    `{"key": "value"}`,
			expected: false,
		},
		{
			name:     "invalid JSON second",
			json1:    `{"key": "value"}`,
			json2:    `{"key":}`,
			expected: false,
		},
		{
			name:     "empty objects",
			json1:    `{}`,
			json2:    `{}`,
			expected: true,
		},
		{
			name:     "nested objects",
			json1:    `{"outer": {"inner": "value"}}`,
			json2:    `{"outer": {"inner": "value"}}`,
			expected: true,
		},
		{
			name:     "arrays",
			json1:    `{"arr": [1, 2, 3]}`,
			json2:    `{"arr": [1, 2, 3]}`,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := functionallyEquivalentJSON(tt.json1, tt.json2)

			if result != tt.expected {
				t.Errorf("functionallyEquivalentJSON() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestExpandStringList tests the expandStringList function
func TestExpandStringList(t *testing.T) {
	tests := []struct {
		name     string
		input    []interface{}
		expected []string
	}{
		{
			name:     "expands strings",
			input:    []interface{}{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "filters empty strings",
			input:    []interface{}{"a", "", "b", ""},
			expected: []string{"a", "b"},
		},
		{
			name:     "empty input",
			input:    []interface{}{},
			expected: []string{},
		},
		{
			name:     "filters non-strings",
			input:    []interface{}{"a", 123, "b", nil, true},
			expected: []string{"a", "b"},
		},
		{
			name:     "all empty strings",
			input:    []interface{}{"", "", ""},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandStringList(tt.input)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("expandStringList() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestFlattenStringList tests the flattenStringList function
func TestFlattenStringList(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []interface{}
	}{
		{
			name:     "flattens strings",
			input:    []string{"a", "b", "c"},
			expected: []interface{}{"a", "b", "c"},
		},
		{
			name:     "empty strings included",
			input:    []string{"a", "", "b"},
			expected: []interface{}{"a", "", "b"},
		},
		{
			name:     "empty input",
			input:    []string{},
			expected: []interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := flattenStringList(tt.input)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("flattenStringList() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestHashcode tests the hashcode function
func TestHashcode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "hash of string",
			input:    "test-string",
			expected: int(crc32.ChecksumIEEE([]byte("test-string"))),
		},
		{
			name:     "empty string",
			input:    "",
			expected: int(crc32.ChecksumIEEE([]byte(""))),
		},
		{
			name:     "unicode string",
			input:    "日本語",
			expected: int(crc32.ChecksumIEEE([]byte("日本語"))),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hashcode(tt.input)

			if result != tt.expected {
				t.Errorf("hashcode() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestHashSum tests the hashSum function
func TestHashSum(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "hash of string",
			input:    "test-content",
			expected: "0a3666a0710c08aa6d0de92ce72beeb5b93124cce1bf3701c9d6cdeb543cb73e",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hashSum(tt.input)

			if result != tt.expected {
				t.Errorf("hashSum() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestTenantPermissionsHash tests the tenantPermissionsHash function
func TestTenantPermissionsHash(t *testing.T) {
	tests := []struct {
		name string
		v    interface{}
	}{
		{
			name: "basic tenant permissions",
			v: map[string]interface{}{
				"tenant_patterns": schema.NewSet(schema.HashString, []interface{}{"global_tenant", "admin_tenant"}),
				"allowed_actions": schema.NewSet(schema.HashString, []interface{}{"kibana_all_write"}),
			},
		},
		{
			name: "empty tenant permissions",
			v:    map[string]interface{}{},
		},
		{
			name: "only tenant patterns",
			v: map[string]interface{}{
				"tenant_patterns": schema.NewSet(schema.HashString, []interface{}{"tenant1"}),
			},
		},
		{
			name: "only allowed actions",
			v: map[string]interface{}{
				"allowed_actions": schema.NewSet(schema.HashString, []interface{}{"read", "write"}),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tenantPermissionsHash(tt.v)

			// Just verify it doesn't panic and returns a reasonable value
			if result < 0 {
				t.Errorf("tenantPermissionsHash() returned negative value: %v", result)
			}
		})
	}
}

// TestIndexPermissionsHash tests the indexPermissionsHash function
func TestIndexPermissionsHash(t *testing.T) {
	tests := []struct {
		name string
		v    interface{}
	}{
		{
			name: "full index permissions",
			v: map[string]interface{}{
				"index_patterns":          schema.NewSet(schema.HashString, []interface{}{"logs-*", "metrics-*"}),
				"document_level_security": "{}",
				"field_level_security":    schema.NewSet(schema.HashString, []interface{}{"field1", "field2"}),
				"masked_fields":           schema.NewSet(schema.HashString, []interface{}{"password"}),
				"allowed_actions":         schema.NewSet(schema.HashString, []interface{}{"read", "write"}),
			},
		},
		{
			name: "minimal index permissions",
			v: map[string]interface{}{
				"index_patterns":  schema.NewSet(schema.HashString, []interface{}{"*"}),
				"allowed_actions": schema.NewSet(schema.HashString, []interface{}{"read"}),
			},
		},
		{
			name: "empty index permissions",
			v:    map[string]interface{}{},
		},
		{
			name: "with fls field (deprecated)",
			v: map[string]interface{}{
				"index_patterns": schema.NewSet(schema.HashString, []interface{}{"*"}),
				"fls":            schema.NewSet(schema.HashString, []interface{}{"field1"}),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := indexPermissionsHash(tt.v)

			// Just verify it doesn't panic and returns a reasonable value
			if result < 0 {
				t.Errorf("indexPermissionsHash() returned negative value: %v", result)
			}
		})
	}
}

// TestIndexPermissionsHashConsistency tests that the hash is consistent
func TestIndexPermissionsHashConsistency(t *testing.T) {
	perm1 := map[string]interface{}{
		"index_patterns":          schema.NewSet(schema.HashString, []interface{}{"logs-*", "metrics-*"}),
		"document_level_security": "",
		"field_level_security":    schema.NewSet(schema.HashString, []interface{}{}),
		"masked_fields":           schema.NewSet(schema.HashString, []interface{}{}),
		"allowed_actions":         schema.NewSet(schema.HashString, []interface{}{"read", "write"}),
	}

	perm2 := map[string]interface{}{
		"index_patterns":          schema.NewSet(schema.HashString, []interface{}{"metrics-*", "logs-*"}), // Different order
		"document_level_security": "",
		"field_level_security":    schema.NewSet(schema.HashString, []interface{}{}),
		"masked_fields":           schema.NewSet(schema.HashString, []interface{}{}),
		"allowed_actions":         schema.NewSet(schema.HashString, []interface{}{"write", "read"}), // Different order
	}

	hash1 := indexPermissionsHash(perm1)
	hash2 := indexPermissionsHash(perm2)

	if hash1 != hash2 {
		t.Errorf("indexPermissionsHash() not consistent for different order: %v != %v", hash1, hash2)
	}
}

// TestTenantPermissionsHashConsistency tests that the hash is consistent
func TestTenantPermissionsHashConsistency(t *testing.T) {
	perm1 := map[string]interface{}{
		"tenant_patterns": schema.NewSet(schema.HashString, []interface{}{"global_tenant", "admin_tenant"}),
		"allowed_actions": schema.NewSet(schema.HashString, []interface{}{"kibana_all_read", "kibana_all_write"}),
	}

	perm2 := map[string]interface{}{
		"tenant_patterns": schema.NewSet(schema.HashString, []interface{}{"admin_tenant", "global_tenant"}),       // Different order
		"allowed_actions": schema.NewSet(schema.HashString, []interface{}{"kibana_all_write", "kibana_all_read"}), // Different order
	}

	hash1 := tenantPermissionsHash(perm1)
	hash2 := tenantPermissionsHash(perm2)

	if hash1 != hash2 {
		t.Errorf("tenantPermissionsHash() not consistent for different order: %v != %v", hash1, hash2)
	}
}

// Helper function to deep copy a map
func deepCopyMap(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		switch val := v.(type) {
		case map[string]interface{}:
			result[k] = deepCopyMap(val)
		case []interface{}:
			result[k] = deepCopySlice(val)
		default:
			result[k] = v
		}
	}
	return result
}

// Helper function to deep copy a slice
func deepCopySlice(s []interface{}) []interface{} {
	result := make([]interface{}, len(s))
	for i, v := range s {
		switch val := v.(type) {
		case map[string]interface{}:
			result[i] = deepCopyMap(val)
		case []interface{}:
			result[i] = deepCopySlice(val)
		default:
			result[i] = v
		}
	}
	return result
}
