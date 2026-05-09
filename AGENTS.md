# Terraform OpenSearch Provider - Test Coverage Enhancement Plan

## Executive Summary

This document outlines the current state of test coverage for the terraform-opensearch-provider and provides a comprehensive plan to address gaps, standardize patterns, and fix drift detection issues.

**Current State:**
- 21 resources defined in the provider
- All resources have test files (100% coverage at file level)
- Test quality varies significantly (1-16 test functions per resource)
- Several resources have perpetual drift issues
- Inconsistent implementation patterns across resources

---

## Resource Inventory and Test Coverage

### Comprehensive Coverage (16 resources)
These resources have CRUD tests, import tests, update tests, and handle edge cases:

| Resource | Test Functions | Key Features Tested |
|----------|---------------|---------------------|
| `opensearch_index` | 16 | CRUD, import, analysis, date math, rollover alias, KNN config, similarity, aliases, whitespace handling |
| `opensearch_cluster_settings` | 3 | Basic, slow logs, type list handling |
| `opensearch_dashboard_object` | 4 | Basic, import, search, multiple objects |
| `opensearch_user` | 2 | Basic, password hash, minimal config |
| `opensearch_role` | 2 | Basic, update, field-level security, import |
| `opensearch_roles_mapping` | 1 | CRUD operations |
| `opensearch_dashboard_tenant` | 1 | CRUD operations |
| `opensearch_audit_config` | 1 | Full CRUD with nested structures |
| `opensearch_monitor` | 2 | Basic, update, import |
| `opensearch_ism_policy` | 2 | Basic, update, import |
| `opensearch_script` | 2 | Basic, update, import |
| `opensearch_component_template` | 2 | Basic, update, import |
| `opensearch_composable_index_template` | 2 | Basic, update, import |
| `opensearch_index_template` | 2 | Basic, update, import |
| `opensearch_ingest_pipeline` | 2 | Basic, update, import |
| `opensearch_snapshot_repository` | 2 | Basic, update, import |

### Missing Import Tests (4 resources)
These resources have basic and update tests but lack import tests:

| Resource | Test Functions | Missing Tests |
|----------|---------------|---------------|
| `opensearch_anomaly_detection` | 1 | Import test (intentionally skipped due to drift issues) |
| `opensearch_channel_configuration` | 1 | Update test, import test |
| `opensearch_ism_policy_mapping` | 1 | Import test |
| `opensearch_sm_policy` | 1 | Import test |

### ForceNew Resources (1 resource)
This resource has `ForceNew: true` on all fields, so update tests are not applicable:

| Resource | Test Functions | Notes |
|----------|---------------|-------|
| `opensearch_data_stream` | 2 | Has basic and import tests; update not applicable due to `ForceNew: true` |

---

## DiffSuppressFunc Analysis

### What is DiffSuppressFunc?

`DiffSuppressFunc` is a Terraform schema function that prevents false-positive diffs when the OpenSearch API modifies resources after creation. The API commonly adds:
- Timestamps (`last_update_time`, `created_time`, `enabled_time`)
- Schema versions (`schema_version`)
- Auto-generated IDs
- Default values (`boost`, `adjust_pure_negative`)

### Current Implementation Status

#### Resources WITH DiffSuppressFunc (9 resources)

| Resource | Function | Fields Removed |
|----------|----------|---------------|
| `opensearch_monitor` | `diffSuppressMonitor` | `id`, `last_update_time`, `enabled_time`, `schema_version`, `user`, trigger IDs, action IDs |
| `opensearch_ism_policy` | `diffSuppressPolicy` | `last_updated_time`, `policy_id`, `schema_version`, `ism_template.last_updated_time` |
| `opensearch_channel_configuration` | `diffSuppressChannelConfiguration` | `last_updated_time_ms`, `created_time_ms` |
| `opensearch_anomaly_detection` | `diffSuppressAnomalyDetection` | `last_update_time` |
| `opensearch_component_template` | `diffSuppressComponentTemplate` | `version`, settings normalization |
| `opensearch_composable_index_template` | `diffSuppressComposableIndexTemplate` | `version`, `data_stream` extra attrs, settings |
| `opensearch_index_template` | `diffSuppressIndexTemplate` | `version`, settings normalization |
| `opensearch_ingest_pipeline` | `diffSuppressIngestPipeline` | Basic JSON compare only |
| `opensearch_sm_policy` | `smDiffSuppressPolicy` | `name`, `enabled_time`, `schedule` |

#### Resources WITHOUT DiffSuppressFunc (12 resources)

These resources may experience drift issues:

| Resource | Body Field Type | Risk Level |
|----------|----------------|------------|
| `opensearch_script` | Plain text source | Low |
| `opensearch_snapshot_repository` | JSON body | Medium |
| `opensearch_dashboard_object` | JSON body | High |
| `opensearch_data_stream` | No JSON body | Low |
| `opensearch_cluster_settings` | Key-value pairs | Low |
| `opensearch_audit_config` | Structured schema | Low |
| `opensearch_user` | Structured schema | Low |
| `opensearch_role` | Structured schema | Low |
| `opensearch_roles_mapping` | Structured schema | Low |
| `opensearch_dashboard_tenant` | Simple fields | Low |
| `opensearch_index` | Multiple fields | Medium |
| `opensearch_ism_policy_mapping` | JSON body | Medium |

### Critical Issue: Channel Configuration and Anomaly Detection Resources

**Problem:** The `opensearch_channel_configuration` and `opensearch_anomaly_detection` tests have `ExpectNonEmptyPlan: true`, indicating persistent drift.

**Root Cause - Channel Configuration:**
1. Channel configuration resource stores its definition in JSON `body` field
2. OpenSearch adds timestamps after creation:
   - `last_updated_time_ms`
   - `created_time_ms`
3. The `diffSuppressChannelConfiguration` function may not fully normalize all config types

**Root Cause - Anomaly Detection:**
1. Anomaly detection resource stores its definition in JSON `body` field
2. OpenSearch transforms query structures after creation (e.g., `gt` queries become `from`/`to` objects)
3. The `diffSuppressAnomalyDetection` function doesn't fully handle filter_query transformations

**Impact:** Users experience perpetual diffs on every plan/apply cycle.

**Note:** The `opensearch_monitor` resource previously had drift issues but has been resolved through enhanced diff suppression that normalizes query defaults (`adjust_pure_negative`, `boost`) and trigger action IDs.

---

## Drift Detection Mechanisms

### How Drift is Currently Detected

1. **404 on Read**: `elastic7.IsNotFound(err)` triggers removal from state
2. **Diff on Plan**: API-added fields cause perpetual diffs (where DiffSuppressFunc is insufficient)
3. **JSON Comparison**: Some resources compare functionally equivalent JSON

### Common Drift Sources

| Source | Affected Resources | Solution |
|--------|-------------------|----------|
| API adds default fields | `opensearch_monitor`, `opensearch_anomaly_detection` | Enhance DiffSuppressFunc |
| Timestamps in responses | All ISM/security resources | Remove timestamp fields |
| Schema versions | `opensearch_ism_policy`, `opensearch_monitor` | Remove `schema_version` |
| Settings normalization | Index templates | Flatten and normalize settings |
| Auto-generated IDs | Triggers, actions in monitors | Remove ID fields |

---

## Proposed Enhancement Plan

### Phase 1: Fix Diff Suppression (Priority: HIGH)

**Goal:** Eliminate perpetual drift issues

#### 1.1 Fix Channel Configuration and Anomaly Detection Diff Suppression

**Current Issues:**
- `diffSuppressChannelConfiguration` doesn't fully normalize all config types (webhook, slack, chime, sns, smtp, ses)
- `diffSuppressAnomalyDetection` doesn't handle filter_query transformations (e.g., `gt` to `from`/`to`)

**Required Changes for Channel Configuration:**
```go
// Enhance normalizeChannelConfiguration to handle:
// - Additional timestamp fields added by API
// - Config-specific default values

func normalizeChannelConfiguration(tpl map[string]interface{}) {
    // Current implementation
    delete(tpl, "last_updated_time_ms")
    delete(tpl, "created_time_ms")
    delete(tpl, "config_id")
    
    // Add normalization for all config types (webhook, slack, chime, sns, smtp, ses, email)
    if config, ok := tpl["config"].(map[string]interface{}); ok {
        // Normalize each config type to remove API-added defaults
        for _, configType := range []string{"webhook", "slack", "chime", "sns", "smtp_account", "ses_account", "email"} {
            if typeConfig, ok := config[configType].(map[string]interface{}); ok {
                normalizeChannelConfigType(typeConfig)
            }
        }
    }
}
```

**Required Changes for Anomaly Detection:**
```go
// Enhance normalizeAnomalyDetection to handle:
// - filter_query transformations
// - Range query normalization

func normalizeAnomalyDetection(tpl map[string]interface{}) {
    // Current implementation
    delete(tpl, "last_update_time")
    delete(tpl, "schema_version")
    delete(tpl, "user")
    
    // Enhanced filter_query normalization
    if filterQuery, ok := tpl["filter_query"].(map[string]interface{}); ok {
        NormalizeQueryDefaults(filterQuery)
        // Handle range query transformations (gt -> from/to)
        normalizeRangeQueries(filterQuery)
    }
}
```

**Testing:**
- Verify no perpetual diffs in tests
- Remove `ExpectNonEmptyPlan: true` from channel_configuration and anomaly_detection tests

#### 1.2 Add Diff Suppression to Missing Resources

**Priority 1:**
- `opensearch_script`: Normalize whitespace in source field
- `opensearch_snapshot_repository`: Add DiffSuppressFunc for JSON body

**Priority 2:**
- `opensearch_dashboard_object`: Consider diff suppression for complex body

#### 1.3 Standardize Diff Suppression Implementation

**Create Shared Utilities:**

File: `provider/diff_suppress_utils.go`

```go
package provider

// RemoveCommonAPIMetadata removes fields commonly added by OpenSearch API
func RemoveCommonAPIMetadata(data map[string]interface{}) {
    delete(data, "last_update_time")
    delete(data, "last_updated_time")
    delete(data, "created_time")
    delete(data, "enabled_time")
    delete(data, "schema_version")
}

// RemoveIDFields removes auto-generated ID fields from nested structures
func RemoveIDFields(data []interface{}) {
    for _, item := range data {
        if m, ok := item.(map[string]interface{}); ok {
            delete(m, "id")
        }
    }
}

// NormalizeJSONBody wraps common JSON normalization logic
func NormalizeJSONBody(old, new string, normalizer func(map[string]interface{})) bool {
    var oo, no interface{}
    if err := json.Unmarshal([]byte(old), &oo); err != nil {
        return false
    }
    if err := json.Unmarshal([]byte(new), &no); err != nil {
        return false
    }
    
    if om, ok := oo.(map[string]interface{}); ok {
        normalizer(om)
    }
    if nm, ok := no.(map[string]interface{}); ok {
        normalizer(nm)
    }
    
    return reflect.DeepEqual(oo, no)
}
```

**Refactor existing functions** to use shared utilities

---

### Phase 2: Add Missing Tests (Priority: HIGH)

#### 2.1 Update Tests (Missing for 10 resources)

**Template:**
```go
func TestAccOpensearch<Resource>Update(t *testing.T) {
    resource.Test(t, resource.TestCase{
        PreCheck:     func() { testAccPreCheck(t) },
        Providers:    testAccProviders,
        CheckDestroy: testCheckOpensearch<Resource>Destroy,
        Steps: []resource.TestStep{
            {
                Config: testAccOpensearch<Resource>Basic,
                Check: resource.ComposeTestCheckFunc(
                    testCheckOpensearch<Resource>Exists("opensearch_<resource>.test"),
                    resource.TestCheckResourceAttr("opensearch_<resource>.test", "<field>", "<initial_value>"),
                ),
            },
            {
                Config: testAccOpensearch<Resource>Update,
                Check: resource.ComposeTestCheckFunc(
                    testCheckOpensearch<Resource>Exists("opensearch_<resource>.test"),
                    resource.TestCheckResourceAttr("opensearch_<resource>.test", "<field>", "<updated_value>"),
                ),
            },
        },
    })
}
```

**Resources needing update tests:**
1. `opensearch_channel_configuration` - Only basic test exists, no update config
2. `opensearch_anomaly_detection` - Only basic test exists, no update config

**Resources needing import tests:**
1. `opensearch_channel_configuration`
2. `opensearch_anomaly_detection`
3. `opensearch_ism_policy_mapping`
4. `opensearch_sm_policy`

---

### Phase 3: Standardize Test Patterns (Priority: MEDIUM)

#### 3.1 Create Test Helper Library

**File:** `provider/test_helpers.go`

```go
package provider

import (
    "fmt"
    "testing"
    
    "github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
    "github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// StandardTestCase returns a common test case structure
type StandardTestCase struct {
    ResourceName    string
    BasicConfig     string
    UpdateConfig    string
    CheckBasic      resource.TestCheckFunc
    CheckUpdate     resource.TestCheckFunc
    ImportVerify    bool
}

// RunStandardTest runs CRUD and import tests for a resource
func RunStandardTest(t *testing.T, tc StandardTestCase) {
    steps := []resource.TestStep{
        {
            Config: tc.BasicConfig,
            Check:  tc.CheckBasic,
        },
        {
            Config: tc.UpdateConfig,
            Check:  tc.CheckUpdate,
        },
    }
    
    if tc.ImportVerify {
        steps = append(steps, resource.TestStep{
            ResourceName:      tc.ResourceName,
            ImportState:       true,
            ImportStateVerify: true,
        })
    }
    
    resource.Test(t, resource.TestCase{
        PreCheck:     func() { testAccPreCheck(t) },
        Providers:    testAccProviders,
        CheckDestroy: testCheckResourceDestroy(tc.ResourceName),
        Steps:        steps,
    })
}

// testCheckResourceDestroy creates a generic destroy checker
func testCheckResourceDestroy(resourceName string) resource.TestCheckFunc {
    return func(s *terraform.State) error {
        // Implementation
    }
}
```

#### 3.2 Standardize Resource Check Functions

**Current Issue:** Each resource implements its own `testCheckOpensearch<Resource>Exists` and `testCheckOpensearch<Resource>Destroy`

**Solution:** Create generic functions parameterized by resource type

#### 3.3 Version-Aware Testing

**Current Issue:** Many tests have unused `allowed` boolean flags that don't actually check OpenSearch version

**Solution:** Implement proper version checking utility:

```go
func skipIfVersionLessThan(t *testing.T, minVersion string) {
    // Implementation using conf.osVersion
}

func skipIfVersionGreaterThan(t *testing.T, maxVersion string) {
    // Implementation
}
```

---

### Phase 4: Drift Detection Improvements (Priority: MEDIUM)

#### 4.1 Add Computed Fields for API-Managed Values

**Problem:** Some fields should be marked as `Computed: true` since they're set by the API

**Resources to review:**
- `opensearch_monitor`: trigger IDs, action IDs
- `opensearch_ism_policy`: `policy_id`, `schema_version`
- `opensearch_anomaly_detection`: `last_update_time`

#### 4.2 Document Drift-Prone Fields

**Add to resource documentation:**
```markdown
## Drift Detection

This resource stores its configuration in a JSON `body` field. The OpenSearch API may add default values that can cause diffs:

- `boost`: Default value of 1 is added to queries
- `adjust_pure_negative`: Default value of true is added
- Trigger and action IDs are auto-generated

These differences are handled by the provider's diff suppression logic and should not cause perpetual diffs.
```

#### 4.3 Add Warning Logs for Drift

**Enhance Read functions:**
```go
func resourceOpensearch<Resource>Read(d *schema.ResourceData, m interface{}) error {
    // ... existing code ...
    
    // Log when API returns unexpected fields
    if apiVersion, ok := response["_version"]; ok && apiVersion != d.Get("version") {
        log.Printf("[WARN] API version %v differs from state version %v", 
            apiVersion, d.Get("version"))
    }
    
    return nil
}
```

---

### Phase 5: Deprecate Legacy Resources (Priority: LOW)

#### 5.1 Mark ism_policy_mapping as Deprecated

**Reason:** This resource is deprecated in OpenSearch 1.x and replaced by `ism_template` in policy definitions

**Action:**
```go
func resourceOpenSearchISMPolicyMapping() *schema.Resource {
    return &schema.Resource{
        Description: "**DEPRECATED**: Use `ism_template` in opensearch_ism_policy instead. ...",
        Deprecated:  "Use opensearch_ism_policy with ism_template instead",
        // ... rest of definition
    }
}
```

---

## Implementation Timeline

### Sprint 1 (Week 1-2): Fix Critical Issues
- [ ] Fix `opensearch_channel_configuration` diff suppression
- [ ] Fix `opensearch_anomaly_detection` diff suppression  
- [ ] Remove `ExpectNonEmptyPlan: true` from channel_configuration and anomaly_detection tests
- [ ] Add shared diff suppression utilities

### Sprint 2 (Week 3-4): Complete Test Coverage
- [ ] Add update test for `opensearch_channel_configuration`
- [ ] Add import tests for missing resources (4 resources)
- [ ] Verify all existing tests pass without `ExpectNonEmptyPlan`

### Sprint 3 (Week 5-6): Standardization
- [ ] Create test helper library
- [ ] Refactor existing tests to use helpers
- [ ] Standardize version checking
- [ ] Add documentation for drift-prone resources

### Sprint 4 (Week 7-8): Cleanup and Documentation
- [ ] Deprecate `opensearch_ism_policy_mapping`
- [ ] Add drift detection warnings
- [ ] Update resource documentation
- [ ] Final review and testing

---

## Resource Priority Matrix

| Resource | Diff Issue | Missing Update | Missing Import | Priority |
|----------|-----------|---------------|----------------|----------|
| opensearch_channel_configuration | HIGH | YES | YES | **P0** |
| opensearch_anomaly_detection | HIGH | YES | YES | **P0** |
| opensearch_index | MEDIUM* | NO | NO | **P1** |
| opensearch_ism_policy_mapping | LOW | NO | YES | **P2** |
| opensearch_sm_policy | LOW | NO | YES | **P2** |
| opensearch_data_stream | LOW | N/A** | NO | **P3** |
| opensearch_monitor | LOW | NO | NO | **Completed** |
| opensearch_ism_policy | LOW | NO | NO | **Completed** |
| opensearch_component_template | LOW | NO | NO | **Completed** |
| opensearch_composable_index_template | LOW | NO | NO | **Completed** |
| opensearch_index_template | LOW | NO | NO | **Completed** |
| opensearch_ingest_pipeline | LOW | NO | NO | **Completed** |
| opensearch_script | LOW | NO | NO | **Completed** |
| opensearch_snapshot_repository | LOW | NO | NO | **Completed** |

*Note: `opensearch_index` has `ExpectNonEmptyPlan: true` on some test steps (rollover alias tests), but basic CRUD is stable.
**Note: `opensearch_data_stream` has `ForceNew: true` on all fields, so update tests are not applicable

---

## Success Criteria

1. **Zero Perpetual Diffs**: All resources should pass tests without `ExpectNonEmptyPlan: true`
2. **100% Import Coverage**: All resources should have import tests
3. **100% Update Coverage**: All mutable resources should have update tests (excluding ForceNew resources)
4. **Standardized Patterns**: All tests should use helper functions where applicable
5. **Documented Drift**: All resources with API-added fields should document drift behavior

---

## Appendix A: Diff Suppression Implementation Guide

### When to Use DiffSuppressFunc

Use `DiffSuppressFunc` when:
1. The API adds default values not present in user configuration
2. The API returns fields in different order than sent
3. The API adds metadata fields (timestamps, versions)
4. Whitespace or formatting differs between config and API response

### Implementation Pattern

```go
func diffSuppress<Resource>(k, old, new string, d *schema.ResourceData) bool {
    return NormalizeJSONBody(old, new, normalize<Resource>)
}

func normalize<Resource>(data map[string]interface{}) {
    // Remove API-generated fields
    RemoveCommonAPIMetadata(data)
    
    // Resource-specific normalization
    delete(data, "<resource_specific_field>")
    
    // Normalize nested structures
    if nested, ok := data["nested"].([]interface{}); ok {
        RemoveIDFields(nested)
    }
}
```

---

## Appendix B: Test Template

### Standard Resource Test

```go
package provider

import (
    "fmt"
    "testing"
    
    "github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
    "github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccOpensearch<Resource>Basic(t *testing.T) {
    resource.Test(t, resource.TestCase{
        PreCheck:     func() { testAccPreCheck(t) },
        Providers:    testAccProviders,
        CheckDestroy: testCheckOpensearch<Resource>Destroy,
        Steps: []resource.TestStep{
            {
                Config: testAccOpensearch<Resource>Basic,
                Check: resource.ComposeTestCheckFunc(
                    testCheckOpensearch<Resource>Exists("opensearch_<resource>.test"),
                ),
            },
        },
    })
}

func TestAccOpensearch<Resource>Update(t *testing.T) {
    resource.Test(t, resource.TestCase{
        PreCheck:     func() { testAccPreCheck(t) },
        Providers:    testAccProviders,
        CheckDestroy: testCheckOpensearch<Resource>Destroy,
        Steps: []resource.TestStep{
            {
                Config: testAccOpensearch<Resource>Basic,
                Check: resource.ComposeTestCheckFunc(
                    testCheckOpensearch<Resource>Exists("opensearch_<resource>.test"),
                ),
            },
            {
                Config: testAccOpensearch<Resource>Update,
                Check: resource.ComposeTestCheckFunc(
                    testCheckOpensearch<Resource>Exists("opensearch_<resource>.test"),
                ),
            },
        },
    })
}

func TestAccOpensearch<Resource>Import(t *testing.T) {
    resource.Test(t, resource.TestCase{
        PreCheck:     func() { testAccPreCheck(t) },
        Providers:    testAccProviders,
        CheckDestroy: testCheckOpensearch<Resource>Destroy,
        Steps: []resource.TestStep{
            {
                Config: testAccOpensearch<Resource>Basic,
            },
            {
                ResourceName:      "opensearch_<resource>.test",
                ImportState:       true,
                ImportStateVerify: true,
            },
        },
    })
}

func testCheckOpensearch<Resource>Exists(name string) resource.TestCheckFunc {
    return func(s *terraform.State) error {
        rs, ok := s.RootModule().Resources[name]
        if !ok {
            return fmt.Errorf("Not found: %s", name)
        }
        if rs.Primary.ID == "" {
            return fmt.Errorf("No resource ID is set")
        }
        
        meta := testAccProvider.Meta()
        _, err := resourceOpensearchGet<Resource>(rs.Primary.ID, meta.(*ProviderConf))
        if err != nil {
            return err
        }
        
        return nil
    }
}

func testCheckOpensearch<Resource>Destroy(s *terraform.State) error {
    for _, rs := range s.RootModule().Resources {
        if rs.Type != "opensearch_<resource>" {
            continue
        }
        
        meta := testAccProvider.Meta()
        _, err := resourceOpensearchGet<Resource>(rs.Primary.ID, meta.(*ProviderConf))
        
        if err != nil {
            return nil // Should be not found
        }
        
        return fmt.Errorf("Resource %q still exists", rs.Primary.ID)
    }
    
    return nil
}

var testAccOpensearch<Resource>Basic = `
resource "opensearch_<resource>" "test" {
  // Basic configuration
}
`

var testAccOpensearch<Resource>Update = `
resource "opensearch_<resource>" "test" {
  // Updated configuration
}
`
```

---

## Notes

- This plan assumes OpenSearch 2.x compatibility
- Some resources may require version-specific test variants
- AWS OpenSearch Service testing may require additional configuration
- Consider adding acceptance test environment setup documentation

---

## Remaining Work

This section tracks bugs and technical debt identified during the latest code review and test suite run. Items are organized by priority.

### 🔴 High Priority

| # | Issue | Impact | Proposed Fix | Status |
|---|-------|--------|------------|--------|
| 1 | **Diff suppression for `opensearch_channel_configuration`** — `diffSuppressChannelConfiguration` doesn't normalize all config types (webhook, slack, chime, sns, smtp, ses, email). OpenSearch adds API-specific defaults that cause perpetual diffs. | Perpetual diffs in production; all 7 test steps use `ExpectNonEmptyPlan: true` to mask the issue | Add normalization for each config type inside the `config` block | Pending |
| 2 | **Diff suppression for `opensearch_anomaly_detection`** — `diffSuppressAnomalyDetection` doesn't handle `filter_query` transformations (e.g., `gt` becomes `from`/`to` objects after API round-trip). | Perpetual diffs in production; both test steps use `ExpectNonEmptyPlan: true` | Add `normalizeRangeQueries` helper and apply it to `filter_query` | Pending |
| 3 | **`opensearch_index` rollover alias drift** — 2 test steps use `ExpectNonEmptyPlan: true` because the resolved rollover alias differs between config and read. | Non-empty plan on rollover alias managed indices | Normalize or suppress the rollover alias field when it is computed by the API | Pending |

### 🟡 Medium Priority

| # | Issue | Impact | Proposed Fix | Status |
|---|-------|--------|------------|--------|
| 4 | **Diff suppression gaps for `ingest_pipeline`, `snapshot_repository`, and `script`** — `ingest_pipeline` has zero normalization (raw JSON compare); `snapshot_repository` and `script` have no `DiffSuppressFunc` at all. | API-added fields (`version`, formatting) cause perpetual diffs | Add normalization to `diffSuppressIngestPipeline`; add `DiffSuppressFunc` schemas to `snapshot_repository` and `script` | Pending |
| 5 | **Version-gated test permanently skipped** — `TestAccOpensearchDashboardObject_Rejected` has `allowed = false` hardcoded, so it never runs even on OpenSearch 2.x+ where it should execute. | Lost test coverage for a 2.x validation | Implement actual version check using `conf.osVersion` in `PreCheck` | Pending |

### 🟢 Low Priority (Code Quality / Technical Debt)

| # | Issue | Impact | Proposed Fix | Status |
|---|-------|--------|------------|--------|
| 6 | **Dead `allowed` booleans in ~5 other test files** — `component_template`, `composable_index_template`, `data_stream`, `ism_policy_mapping`, and `sm_policy` tests have stub version checks that never dynamically evaluate. | False sense of version safety | Remove unused booleans or implement real version checking | Pending |
| 7 | **Shared `NormalizeJSONBody` utility unused** — all 9 diff suppress functions duplicate the same JSON unmarshal/normalize/compare boilerplate instead of calling the reusable helper. | Code duplication | Refactor all diff suppress functions to use `NormalizeJSONBody()` | Pending |
| 8 | **Empty `analysis` map added unconditionally** — in `resource_opensearch_index.go`, `settings["analysis"] = analysis` executes even when no analysis fields are configured. | Adds empty map to index settings; potential drift | Only add the `analysis` key when at least one analysis sub-field is present | Pending |
| 9 | **Typos and grammar** — `ojbect`, `controling`, `percentabge`, incomplete sentence in index destroy error message. | Minor documentation/comment quality | Fix spelling across `monitor.go`, `cluster_settings.go`, `index.go` | Pending |

---

## AGENTS Rules (MUST FOLLOW)

### Test Modification Rules

1. **NEVER remove tests without explicit user approval**
   - If tests are failing, investigate and fix the root cause
   - Do not delete or comment out tests to make the suite pass
   - Document any tests that cannot be fixed with detailed reasoning

2. **ALWAYS wait for answers to questions**
   - When asking the user a question, STOP and wait for their response
   - NEVER assume the answer or proceed without confirmation
   - If uncertain about how to proceed, ask rather than guess

3. **Preserve all existing functionality**
   - Do not remove functionality to make tests pass
   - If there's a conflict between test design and functionality, ask the user
   - Fix the underlying issue, not the symptom

4. **When adding import tests:**
   - First check if the resource has an Importer defined
   - If import fails due to pre-existing bugs, document the bug rather than remove the test
   - Attempt to fix the underlying import functionality before considering removal

### Violation Consequences

Failure to follow these rules may result in:
- Loss of test coverage
- Hidden bugs going undetected
- Reduced confidence in the codebase
- Technical debt accumulation
