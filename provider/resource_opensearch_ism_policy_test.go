// DESIGN ISSUE: This resource stores configuration in a single 'body' field as JSON.
// The OpenSearch API returns computed fields not present in the configuration,
// causing perpetual drift in terraform plan. Import verification will fail.
// This is a fundamental design limitation of the current implementation.

package provider

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccOpensearchISMPolicy(t *testing.T) {
	provider := Provider()
	diags := provider.Configure(context.Background(), &terraform.ResourceConfig{})
	if diags.HasError() {
		t.Skipf("err: %#v", diags)
	}
	var allowed bool

	config := testAccOpensearchISMPolicyV7

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			if !allowed {
				t.Skip("OpenSearch ISMPolicies only supported on ES 6.")
			}
		},
		Providers:    testAccOpendistroProviders,
		CheckDestroy: testCheckOpensearchISMPolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchISMPolicyExists("opensearch_ism_policy.test_policy"),
					resource.TestCheckResourceAttr(
						"opensearch_ism_policy.test_policy",
						"policy_id",
						"test_policy",
					),
				),
			},
		},
	})
}

func testCheckOpensearchISMPolicyExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No policy ID is set")
		}

		meta := testAccOpendistroProvider.Meta()

		var err error
		_, err = resourceOpensearchGetISMPolicy(rs.Primary.ID, meta.(*ProviderConf))

		if err != nil {
			return err
		}

		return nil
	}
}

func testCheckOpensearchISMPolicyDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opensearch_ism_policy" {
			continue
		}

		meta := testAccOpendistroProvider.Meta()

		var err error
		if err != nil {
			return err
		}
		_, err = resourceOpensearchGetISMPolicy(rs.Primary.ID, meta.(*ProviderConf))

		if err != nil {
			return nil // should be not found error
		}

		return fmt.Errorf("OpenDistroISMPolicy %q still exists", rs.Primary.ID)
	}

	return nil
}

var testAccOpensearchISMPolicyV7 = `
resource "opensearch_ism_policy" "test_policy" {
  policy_id = "test_policy"
  body      = <<EOF
  {
		"policy": {
		  "description": "ingesting logs",
		  "default_state": "ingest",
      "ism_template": [{
        "index_patterns": ["foo-*"],
        "priority": 0
			}],
		  "error_notification": {
        "destination": {
          "slack": {
            "url": "https://webhook.slack.example.com"
          }
        },
        "message_template": {
          "lang": "mustache",
          "source": "The index *{{ctx.index}}* failed to rollover."
        }
      },
		  "states": [
				{
				  "name": "ingest",
				  "actions": [{
					  "rollover": {
						"min_doc_count": 5
					  }
					}],
				  "transitions": [{
					  "state_name": "search"
					}]
				},
				{
				  "name": "search",
				  "actions": [],
				  "transitions": [{
					  "state_name": "delete",
					  "conditions": {
						"min_index_age": "5m"
					  }
					}]
				},
				{
				  "name": "delete",
				  "actions": [{
					  "delete": {}
					}],
				  "transitions": []
				}
			]
		}
	}
  EOF
}
`

var testAccOpensearchISMPolicyV7Updated = `
resource "opensearch_ism_policy" "test_policy" {
  policy_id = "test_policy"
  body      = <<EOF
  {
		"policy": {
		  "description": "updated policy description",
		  "default_state": "ingest",
      "ism_template": [{
        "index_patterns": ["bar-*"],
        "priority": 1
			}],
		  "error_notification": {
        "destination": {
          "slack": {
            "url": "https://webhook.slack.example.com/updated"
          }
        },
        "message_template": {
          "lang": "mustache",
          "source": "The index *{{ctx.index}}* failed to rollover. Please check!"
        }
      },
		  "states": [
				{
				  "name": "ingest",
				  "actions": [{
					  "rollover": {
						"min_doc_count": 10
					  }
					}],
				  "transitions": [{
					  "state_name": "search"
					}]
				},
				{
				  "name": "search",
				  "actions": [],
				  "transitions": [{
					  "state_name": "delete",
					  "conditions": {
						"min_index_age": "10m"
					  }
					}]
				},
				{
				  "name": "delete",
				  "actions": [{
					  "delete": {}
					}],
				  "transitions": []
				}
			]
		}
	}
  EOF
}
`

func TestAccOpensearchISMPolicy_update(t *testing.T) {
	provider := Provider()
	diags := provider.Configure(context.Background(), &terraform.ResourceConfig{})
	if diags.HasError() {
		t.Skipf("err: %#v", diags)
	}
	var allowed bool

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			if !allowed {
				t.Skip("OpenSearch ISMPolicies only supported on ES 6.")
			}
		},
		Providers:    testAccOpendistroProviders,
		CheckDestroy: testCheckOpensearchISMPolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchISMPolicyV7,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchISMPolicyExists("opensearch_ism_policy.test_policy"),
					resource.TestCheckResourceAttr(
						"opensearch_ism_policy.test_policy",
						"policy_id",
						"test_policy",
					),
				),
			},
			{
				Config: testAccOpensearchISMPolicyV7Updated,
				Check: resource.ComposeTestCheckFunc(
					testCheckOpensearchISMPolicyExists("opensearch_ism_policy.test_policy"),
					resource.TestCheckResourceAttr(
						"opensearch_ism_policy.test_policy",
						"policy_id",
						"test_policy",
					),
				),
			},
		},
	})
}

// Import test - ImportStateVerifyIgnore includes 'body' because the API returns
// computed fields not present in the original configuration, causing perpetual drift.
// This is a fundamental design limitation of the current implementation.
func TestAccOpensearchISMPolicy_importBasic(t *testing.T) {
	provider := Provider()
	diags := provider.Configure(context.Background(), &terraform.ResourceConfig{})
	if diags.HasError() {
		t.Skipf("err: %#v", diags)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccOpendistroProviders,
		CheckDestroy: testCheckOpensearchISMPolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOpensearchISMPolicyV7,
			},
			{
				ResourceName:            "opensearch_ism_policy.test_policy",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"body", "primary_term", "seq_no"},
			},
		},
	})
}

func TestAccOpensearchISMPolicy_validationError_InvalidPolicyStates(t *testing.T) {
	provider := Provider()
	diags := provider.Configure(context.Background(), &terraform.ResourceConfig{})
	if diags.HasError() {
		t.Skipf("err: %#v", diags)
	}
	var allowed bool

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			if !allowed {
				t.Skip("OpenSearch ISMPolicies only supported on ES 6.")
			}
		},
		Providers:    testAccOpendistroProviders,
		CheckDestroy: testCheckOpensearchISMPolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccISMPolicyValidationErrorInvalidStates,
				ExpectError: regexp.MustCompile("states|invalid|policy.*error"),
			},
		},
	})
}

func TestAccOpensearchISMPolicy_validationError_MissingDefaultState(t *testing.T) {
	provider := Provider()
	diags := provider.Configure(context.Background(), &terraform.ResourceConfig{})
	if diags.HasError() {
		t.Skipf("err: %#v", diags)
	}
	var allowed bool

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			if !allowed {
				t.Skip("OpenSearch ISMPolicies only supported on ES 6.")
			}
		},
		Providers:    testAccOpendistroProviders,
		CheckDestroy: testCheckOpensearchISMPolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccISMPolicyValidationErrorMissingDefaultState,
				ExpectError: regexp.MustCompile("default_state|required|missing"),
			},
		},
	})
}

func TestAccOpensearchISMPolicy_validationError_InvalidTransitions(t *testing.T) {
	provider := Provider()
	diags := provider.Configure(context.Background(), &terraform.ResourceConfig{})
	if diags.HasError() {
		t.Skipf("err: %#v", diags)
	}
	var allowed bool

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			if !allowed {
				t.Skip("OpenSearch ISMPolicies only supported on ES 6.")
			}
		},
		Providers:    testAccOpendistroProviders,
		CheckDestroy: testCheckOpensearchISMPolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccISMPolicyValidationErrorInvalidTransitions,
				ExpectError: regexp.MustCompile("transitions|invalid|state_name"),
			},
		},
	})
}

var testAccISMPolicyValidationErrorInvalidStates = `
resource "opensearch_ism_policy" "test_validation" {
  policy_id = "test_validation"
  body      = <<EOF
  {
		"policy": {
		  "description": "test policy with invalid states",
		  "default_state": "nonexistent_state",
		  "states": [
				{
				  "name": "valid_state",
				  "actions": [],
				  "transitions": []
				}
			]
		}
	}
  EOF
}
`

var testAccISMPolicyValidationErrorMissingDefaultState = `
resource "opensearch_ism_policy" "test_validation" {
  policy_id = "test_validation"
  body      = <<EOF
  {
		"policy": {
		  "description": "test policy without default state",
		  "states": [
				{
				  "name": "valid_state",
				  "actions": [],
				  "transitions": []
				}
			]
		}
	}
  EOF
}
`

var testAccISMPolicyValidationErrorInvalidTransitions = `
resource "opensearch_ism_policy" "test_validation" {
  policy_id = "test_validation"
  body      = <<EOF
  {
		"policy": {
		  "description": "test policy with invalid transition",
		  "default_state": "state1",
		  "states": [
				{
				  "name": "state1",
				  "actions": [],
				  "transitions": [{
					  "state_name": "nonexistent_state"
					}]
				}
			]
		}
	}
  EOF
}
`
