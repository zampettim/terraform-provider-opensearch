package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// TestDiffSuppressIndexTemplate tests the diffSuppressIndexTemplate function
func TestDiffSuppressIndexTemplate(t *testing.T) {
	tests := []struct {
		name     string
		old      string
		new      string
		expected bool
	}{
		{
			name:     "identical JSON",
			old:      `{"index_patterns":["logs-*"],"settings":{"number_of_shards":1}}`,
			new:      `{"index_patterns":["logs-*"],"settings":{"number_of_shards":1}}`,
			expected: true,
		},
		{
			name:     "different key order",
			old:      `{"index_patterns":["logs-*"],"priority":500}`,
			new:      `{"priority":500,"index_patterns":["logs-*"]}`,
			expected: true,
		},
		{
			name:     "API adds version field",
			old:      `{"index_patterns":["logs-*"]}`,
			new:      `{"index_patterns":["logs-*"],"version":1}`,
			expected: true,
		},
		{
			name:     "API adds default order (0)",
			old:      `{"index_patterns":["logs-*"]}`,
			new:      `{"index_patterns":["logs-*"],"order":0}`,
			expected: true,
		},
		{
			name:     "meaningful order change",
			old:      `{"index_patterns":["logs-*"],"order":100}`,
			new:      `{"index_patterns":["logs-*"],"order":200}`,
			expected: false,
		},
		{
			name:     "different index patterns",
			old:      `{"index_patterns":["logs-*"]}`,
			new:      `{"index_patterns":["metrics-*"]}`,
			expected: false,
		},
		{
			name:     "invalid old JSON",
			old:      `{"index_patterns":}`,
			new:      `{"index_patterns":["logs-*"]}`,
			expected: false,
		},
		{
			name:     "invalid new JSON",
			old:      `{"index_patterns":["logs-*"]}`,
			new:      `{"index_patterns":}`,
			expected: false,
		},
		{
			name:     "settings normalization",
			old:      `{"settings":{"number_of_shards":1}}`,
			new:      `{"settings":{"index.number_of_shards":"1"}}`,
			expected: true,
		},
		{
			name:     "removes empty settings",
			old:      `{"index_patterns":["logs-*"],"settings":{}}`,
			new:      `{"index_patterns":["logs-*"]}`,
			expected: true,
		},
		{
			name:     "removes empty mappings",
			old:      `{"index_patterns":["logs-*"],"mappings":{}}`,
			new:      `{"index_patterns":["logs-*"]}`,
			expected: true,
		},
		{
			name:     "empty strings",
			old:      "",
			new:      "",
			expected: false,
		},
		{
			name:     "old empty new has value",
			old:      "",
			new:      `{"index_patterns":["logs-*"]}`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d *schema.ResourceData = nil
			result := diffSuppressIndexTemplate("body", tt.old, tt.new, d)

			if result != tt.expected {
				t.Errorf("diffSuppressIndexTemplate() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestDiffSuppressComposableIndexTemplate tests the diffSuppressComposableIndexTemplate function
func TestDiffSuppressComposableIndexTemplate(t *testing.T) {
	tests := []struct {
		name     string
		old      string
		new      string
		expected bool
	}{
		{
			name:     "identical JSON",
			old:      `{"index_patterns":["logs-*"],"composed_of":["component1"]}`,
			new:      `{"index_patterns":["logs-*"],"composed_of":["component1"]}`,
			expected: true,
		},
		{
			name:     "API adds version field",
			old:      `{"index_patterns":["logs-*"]}`,
			new:      `{"index_patterns":["logs-*"],"version":1}`,
			expected: true,
		},
		{
			name:     "filters data_stream extra fields",
			old:      `{"data_stream":{"hidden":true}}`,
			new:      `{"data_stream":{"hidden":true,"timestamp_field":"@timestamp","extra":"value"}}`,
			expected: true,
		},
		{
			name:     "data_stream hidden change",
			old:      `{"data_stream":{"hidden":false}}`,
			new:      `{"data_stream":{"hidden":true}}`,
			expected: false,
		},
		{
			name:     "settings normalization",
			old:      `{"template":{"settings":{"number_of_shards":1}}}`,
			new:      `{"template":{"settings":{"index.number_of_shards":"1"}}}`,
			expected: true,
		},
		{
			name:     "different composed_of",
			old:      `{"composed_of":["component1"]}`,
			new:      `{"composed_of":["component2"]}`,
			expected: false,
		},
		{
			name:     "different priority",
			old:      `{"priority":100}`,
			new:      `{"priority":200}`,
			expected: false,
		},
		{
			name:     "invalid old JSON",
			old:      `{"index_patterns":}`,
			new:      `{"index_patterns":["logs-*"]}`,
			expected: false,
		},
		{
			name:     "invalid new JSON",
			old:      `{"index_patterns":["logs-*"]}`,
			new:      `{"index_patterns":}`,
			expected: false,
		},
		{
			name:     "empty strings",
			old:      "",
			new:      "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d *schema.ResourceData = nil
			result := diffSuppressComposableIndexTemplate("body", tt.old, tt.new, d)

			if result != tt.expected {
				t.Errorf("diffSuppressComposableIndexTemplate() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestDiffSuppressComponentTemplate tests the diffSuppressComponentTemplate function
func TestDiffSuppressComponentTemplate(t *testing.T) {
	tests := []struct {
		name     string
		old      string
		new      string
		expected bool
	}{
		{
			name:     "identical JSON",
			old:      `{"template":{"settings":{"number_of_shards":1}}}`,
			new:      `{"template":{"settings":{"number_of_shards":1}}}`,
			expected: true,
		},
		{
			name:     "API adds version field",
			old:      `{"template":{"settings":{"number_of_shards":1}}}`,
			new:      `{"version":2,"template":{"settings":{"number_of_shards":1}}}`,
			expected: true,
		},
		{
			name:     "settings normalization",
			old:      `{"template":{"settings":{"number_of_shards":1}}}`,
			new:      `{"template":{"settings":{"index.number_of_shards":"1"}}}`,
			expected: true,
		},
		{
			name:     "different settings value",
			old:      `{"template":{"settings":{"number_of_shards":1}}}`,
			new:      `{"template":{"settings":{"number_of_shards":2}}}`,
			expected: false,
		},
		{
			name:     "different mappings",
			old:      `{"template":{"mappings":{"properties":{"name":{"type":"text"}}}}}`,
			new:      `{"template":{"mappings":{"properties":{"name":{"type":"keyword"}}}}}`,
			expected: false,
		},
		{
			name:     "invalid JSON",
			old:      `{"template":}`,
			new:      `{"template":{}}`,
			expected: false,
		},
		{
			name:     "empty strings",
			old:      "",
			new:      "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d *schema.ResourceData = nil
			result := diffSuppressComponentTemplate("body", tt.old, tt.new, d)

			if result != tt.expected {
				t.Errorf("diffSuppressComponentTemplate() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestDiffSuppressMonitor tests the diffSuppressMonitor function
func TestDiffSuppressMonitor(t *testing.T) {
	tests := []struct {
		name     string
		old      string
		new      string
		expected bool
	}{
		{
			name:     "identical JSON",
			old:      `{"name":"cpu-monitor","type":"monitor"}`,
			new:      `{"name":"cpu-monitor","type":"monitor"}`,
			expected: true,
		},
		{
			name:     "API adds metadata fields",
			old:      `{"name":"cpu-monitor"}`,
			new:      `{"name":"cpu-monitor","id":"monitor-123","last_update_time":1234567890,"enabled_time":1234567800,"schema_version":1}`,
			expected: true,
		},
		{
			name:     "removes trigger IDs",
			old:      `{"triggers":[{"name":"high-cpu","actions":[{"name":"email"}]}]}`,
			new:      `{"triggers":[{"id":"trigger-1","name":"high-cpu","actions":[{"id":"action-1","name":"email"}]}]}`,
			expected: true,
		},
		{
			name:     "different monitor name",
			old:      `{"name":"cpu-monitor"}`,
			new:      `{"name":"memory-monitor"}`,
			expected: false,
		},
		{
			name:     "different monitor type",
			old:      `{"name":"monitor","type":"monitor"}`,
			new:      `{"name":"monitor","type":"bucket_monitor"}`,
			expected: false,
		},
		{
			name:     "different trigger name",
			old:      `{"triggers":[{"name":"trigger1"}]}`,
			new:      `{"triggers":[{"name":"trigger2"}]}`,
			expected: false,
		},
		{
			name:     "different action name",
			old:      `{"triggers":[{"name":"trigger1","actions":[{"name":"action1"}]}]}`,
			new:      `{"triggers":[{"name":"trigger1","actions":[{"name":"action2"}]}]}`,
			expected: false,
		},
		{
			name:     "API adds user field",
			old:      `{"name":"monitor"}`,
			new:      `{"name":"monitor","user":{"name":"admin"}}`,
			expected: true,
		},
		{
			name:     "invalid JSON",
			old:      `{"name":}`,
			new:      `{"name":"monitor"}`,
			expected: false,
		},
		{
			name:     "empty strings",
			old:      "",
			new:      "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d *schema.ResourceData = nil
			result := diffSuppressMonitor("body", tt.old, tt.new, d)

			if result != tt.expected {
				t.Errorf("diffSuppressMonitor() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestDiffSuppressChannelConfiguration tests the diffSuppressChannelConfiguration function
func TestDiffSuppressChannelConfiguration(t *testing.T) {
	tests := []struct {
		name     string
		old      string
		new      string
		expected bool
	}{
		{
			name:     "identical JSON",
			old:      `{"name":"test-channel","config":{"webhook":{"url":"https://example.com"}}}`,
			new:      `{"name":"test-channel","config":{"webhook":{"url":"https://example.com"}}}`,
			expected: true,
		},
		{
			name:     "API adds timestamp fields",
			old:      `{"name":"test-channel"}`,
			new:      `{"name":"test-channel","last_updated_time_ms":1234567890,"created_time_ms":1234567800}`,
			expected: true,
		},
		{
			name:     "different channel name",
			old:      `{"name":"channel1"}`,
			new:      `{"name":"channel2"}`,
			expected: false,
		},
		{
			name:     "different webhook URL",
			old:      `{"config":{"webhook":{"url":"https://old.com"}}}`,
			new:      `{"config":{"webhook":{"url":"https://new.com"}}}`,
			expected: false,
		},
		{
			name:     "different channel type",
			old:      `{"config":{"webhook":{"url":"https://example.com"}}}`,
			new:      `{"config":{"slack":{"url":"https://hooks.slack.com"}}}`,
			expected: false,
		},
		{
			name:     "invalid JSON",
			old:      `{"name":}`,
			new:      `{"name":"test"}`,
			expected: false,
		},
		{
			name:     "empty strings",
			old:      "",
			new:      "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d *schema.ResourceData = nil
			result := diffSuppressChannelConfiguration("body", tt.old, tt.new, d)

			if result != tt.expected {
				t.Errorf("diffSuppressChannelConfiguration() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestDiffSuppressIngestPipeline tests the diffSuppressIngestPipeline function
func TestDiffSuppressIngestPipeline(t *testing.T) {
	tests := []struct {
		name     string
		old      string
		new      string
		expected bool
	}{
		{
			name:     "identical JSON",
			old:      `{"description":"My pipeline","processors":[{"set":{"field":"foo","value":"bar"}}]}`,
			new:      `{"description":"My pipeline","processors":[{"set":{"field":"foo","value":"bar"}}]}`,
			expected: true,
		},
		{
			name:     "different key order",
			old:      `{"description":"My pipeline","processors":[]}`,
			new:      `{"processors":[],"description":"My pipeline"}`,
			expected: true,
		},
		{
			name:     "different processor order",
			old:      `{"processors":[{"set":{"field":"foo"}},{"remove":{"field":"bar"}}]}`,
			new:      `{"processors":[{"remove":{"field":"bar"}},{"set":{"field":"foo"}}]}`,
			expected: false,
		},
		{
			name:     "different description",
			old:      `{"description":"Old pipeline"}`,
			new:      `{"description":"New pipeline"}`,
			expected: false,
		},
		{
			name:     "different processor config",
			old:      `{"processors":[{"set":{"field":"foo","value":"old"}}]}`,
			new:      `{"processors":[{"set":{"field":"foo","value":"new"}}]}`,
			expected: false,
		},
		{
			name:     "invalid old JSON",
			old:      `{"description":}`,
			new:      `{"description":"test"}`,
			expected: false,
		},
		{
			name:     "invalid new JSON",
			old:      `{"description":"test"}`,
			new:      `{"description":}`,
			expected: false,
		},
		{
			name:     "empty strings",
			old:      "",
			new:      "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d *schema.ResourceData = nil
			result := diffSuppressIngestPipeline("body", tt.old, tt.new, d)

			if result != tt.expected {
				t.Errorf("diffSuppressIngestPipeline() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestDiffSuppressPolicy tests the diffSuppressPolicy function
func TestDiffSuppressPolicy(t *testing.T) {
	tests := []struct {
		name     string
		old      string
		new      string
		expected bool
	}{
		{
			name:     "identical JSON",
			old:      `{"description":"My policy","default_state":"hot"}`,
			new:      `{"description":"My policy","default_state":"hot"}`,
			expected: true,
		},
		{
			name:     "API adds metadata fields",
			old:      `{"description":"My policy"}`,
			new:      `{"policy_id":"policy-123","last_updated_time":1234567890,"schema_version":1,"description":"My policy"}`,
			expected: true,
		},
		{
			name:     "removes null error_notification",
			old:      `{"description":"My policy"}`,
			new:      `{"description":"My policy","error_notification":null}`,
			expected: true,
		},
		{
			name:     "removes null ism_template",
			old:      `{"description":"My policy"}`,
			new:      `{"description":"My policy","ism_template":null}`,
			expected: true,
		},
		{
			name:     "removes timestamp from ism_template",
			old:      `{"ism_template":{"index_patterns":["logs-*"]}}`,
			new:      `{"ism_template":{"index_patterns":["logs-*"],"last_updated_time":1234567890}}`,
			expected: true,
		},
		{
			name:     "different description",
			old:      `{"description":"Old policy"}`,
			new:      `{"description":"New policy"}`,
			expected: false,
		},
		{
			name:     "different default_state",
			old:      `{"default_state":"hot"}`,
			new:      `{"default_state":"warm"}`,
			expected: false,
		},
		{
			name:     "different error_notification",
			old:      `{"error_notification":{"destination":{"slack":{"url":"https://old.com"}}}}`,
			new:      `{"error_notification":{"destination":{"slack":{"url":"https://new.com"}}}}`,
			expected: false,
		},
		{
			name:     "keeps non-null error_notification",
			old:      `{"description":"My policy","error_notification":{"destination":{}}}`,
			new:      `{"description":"My policy","error_notification":{"destination":{}}}`,
			expected: true,
		},
		{
			name:     "different ism_template",
			old:      `{"ism_template":{"index_patterns":["logs-*"]}}`,
			new:      `{"ism_template":{"index_patterns":["metrics-*"]}}`,
			expected: false,
		},
		{
			name:     "invalid JSON",
			old:      `{"description":}`,
			new:      `{"description":"test"}`,
			expected: false,
		},
		{
			name:     "empty strings",
			old:      "",
			new:      "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d *schema.ResourceData = nil
			result := diffSuppressPolicy("body", tt.old, tt.new, d)

			if result != tt.expected {
				t.Errorf("diffSuppressPolicy() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestDiffSuppressAnomalyDetection tests the diffSuppressAnomalyDetection function
func TestDiffSuppressAnomalyDetection(t *testing.T) {
	tests := []struct {
		name     string
		old      string
		new      string
		expected bool
	}{
		{
			name:     "identical JSON",
			old:      `{"name":"detector","description":"My detector"}`,
			new:      `{"name":"detector","description":"My detector"}`,
			expected: true,
		},
		{
			name:     "API adds last_update_time",
			old:      `{"name":"detector"}`,
			new:      `{"name":"detector","last_update_time":1234567890}`,
			expected: true,
		},
		{
			name:     "different name",
			old:      `{"name":"detector1"}`,
			new:      `{"name":"detector2"}`,
			expected: false,
		},
		{
			name:     "different description",
			old:      `{"description":"Old detector"}`,
			new:      `{"description":"New detector"}`,
			expected: false,
		},
		{
			name:     "different feature attributes",
			old:      `{"features":[{"feature_name":"cpu","feature_enabled":true}]}`,
			new:      `{"features":[{"feature_name":"cpu","feature_enabled":false}]}`,
			expected: false,
		},
		{
			name:     "invalid JSON",
			old:      `{"name":}`,
			new:      `{"name":"test"}`,
			expected: false,
		},
		{
			name:     "empty strings",
			old:      "",
			new:      "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d *schema.ResourceData = nil
			result := diffSuppressAnomalyDetection("body", tt.old, tt.new, d)

			if result != tt.expected {
				t.Errorf("diffSuppressAnomalyDetection() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestDiffSuppressFuncEdgeCases tests edge cases for all diff suppress functions
func TestDiffSuppressFuncEdgeCases(t *testing.T) {
	functions := map[string]func(string, string, string, *schema.ResourceData) bool{
		"diffSuppressIndexTemplate":           diffSuppressIndexTemplate,
		"diffSuppressComposableIndexTemplate": diffSuppressComposableIndexTemplate,
		"diffSuppressComponentTemplate":       diffSuppressComponentTemplate,
		"diffSuppressMonitor":                 diffSuppressMonitor,
		"diffSuppressChannelConfiguration":    diffSuppressChannelConfiguration,
		"diffSuppressIngestPipeline":          diffSuppressIngestPipeline,
		"diffSuppressPolicy":                  diffSuppressPolicy,
		"diffSuppressAnomalyDetection":        diffSuppressAnomalyDetection,
	}

	for name, fn := range functions {
		t.Run(name+"_both_empty", func(t *testing.T) {
			var d *schema.ResourceData = nil
			result := fn("body", "", "", d)
			if result {
				t.Errorf("%s() with both empty should return false", name)
			}
		})

		t.Run(name+"_invalid_both", func(t *testing.T) {
			var d *schema.ResourceData = nil
			result := fn("body", "{invalid", "{invalid", d)
			if result {
				t.Errorf("%s() with both invalid should return false", name)
			}
		})

		t.Run(name+"_one_invalid", func(t *testing.T) {
			var d *schema.ResourceData = nil
			result := fn("body", `{"valid":true}`, "{invalid", d)
			if result {
				t.Errorf("%s() with one invalid should return false", name)
			}
		})
	}
}
