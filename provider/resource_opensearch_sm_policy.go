package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/structure"
)

var openSearchSMPolicySchema = map[string]*schema.Schema{
	"policy_name": {
		Description: "The name of the SM policy.",
		Type:        schema.TypeString,
		Required:    true,
		ForceNew:    true,
	},
	"body": {
		Description:      "The policy document.",
		Type:             schema.TypeString,
		Required:         true,
		DiffSuppressFunc: smDiffSuppressPolicy,
		StateFunc: func(v interface{}) string {
			json, _ := structure.NormalizeJsonString(v)
			return json
		},
	},
	"primary_term": {
		Description: "The primary term of the SM policy version.",
		Type:        schema.TypeInt,
		Optional:    true,
		Computed:    true,
	},
	"seq_no": {
		Description: "The sequence number of the SM policy version.",
		Type:        schema.TypeInt,
		Optional:    true,
		Computed:    true,
	},
}

func resourceOpenSearchSMPolicy() *schema.Resource {
	return &schema.Resource{
		Description: "Provides an OpenSearch Snapshot Management (SM) policy. Please refer to the OpenSearch SM documentation for details.",
		Create:      resourceOpensearchSMPolicyCreate,
		Read:        resourceOpensearchSMPolicyRead,
		Update:      resourceOpensearchSMPolicyUpdate,
		Delete:      resourceOpensearchSMPolicyDelete,
		Schema:      openSearchSMPolicySchema,
		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
				var err = d.Set("policy_name", d.Id())
				if err != nil {
					return nil, err
				}

				d.SetId(fmt.Sprintf("%s-sm-policy", d.Id()))
				return []*schema.ResourceData{d}, nil
			},
		},
	}
}

func resourceOpensearchSMPolicyCreate(d *schema.ResourceData, m interface{}) error {
	policyResponse, err := resourceOpensearchPostPutSMPolicy(d, m, "POST")

	if err != nil {
		log.Printf("[INFO] Failed to create OpenSearchPolicy: %+v", err)
		return err
	}

	d.SetId(policyResponse.PolicyID)
	return resourceOpensearchSMPolicyRead(d, m)
}

func resourceOpensearchSMPolicyRead(d *schema.ResourceData, m interface{}) error {
	policyResponse, err := resourceOpensearchGetSMPolicy(d.Get("policy_name").(string), m)

	if err != nil {
		if isNotFound(err) {
			log.Printf("[WARN] OpenSearch Policy (%s) not found, removing from state", d.Get("policy_name").(string))
			d.SetId("")
			return nil
		}
		return err
	}

	bodyString, err := json.Marshal(policyResponse.Policy)
	if err != nil {
		return err
	}

	bodyStringNormalized, _ := structure.NormalizeJsonString(string(bodyString))

	if err := d.Set("policy_name", policyResponse.Policy["name"]); err != nil {
		return fmt.Errorf("error setting policy_name: %s", err)
	}
	if err := d.Set("body", bodyStringNormalized); err != nil {
		return fmt.Errorf("error setting body: %s", err)
	}
	if err := d.Set("primary_term", policyResponse.PrimaryTerm); err != nil {
		return fmt.Errorf("error setting primary_term: %s", err)
	}
	if err := d.Set("seq_no", policyResponse.SeqNo); err != nil {
		return fmt.Errorf("error setting seq_no: %s", err)
	}

	return nil
}

func resourceOpensearchSMPolicyUpdate(d *schema.ResourceData, m interface{}) error {
	if _, err := resourceOpensearchPostPutSMPolicy(d, m, "PUT"); err != nil {
		return err
	}

	return resourceOpensearchSMPolicyRead(d, m)
}

func resourceOpensearchSMPolicyDelete(d *schema.ResourceData, m interface{}) error {
	path := fmt.Sprintf("/_plugins/_sm/policies/%s", d.Get("policy_name").(string))

	client, err := getOpenSearchClient(m.(*ProviderConf))
	if err != nil {
		return err
	}

	// Execute request with retry logic
	opts := ismRetryOptions(client.config, "sm policy")
	resp, err := doRequestWithRetry(opts, func() (*http.Request, error) {
		return http.NewRequest("DELETE", client.config.rawUrl+path, nil)
	}, client.Client.Client.Perform)
	if err != nil {
		return fmt.Errorf("error deleting policy: %+v : %+v", path, err)
	}
	defer resp.Body.Close()

	// Check for successful deletion (2xx status codes)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	return fmt.Errorf("error deleting policy: received status code %d", resp.StatusCode)
}

func resourceOpensearchGetSMPolicy(policyName string, m interface{}) (SMPolicyResponse, error) {
	response := new(SMPolicyResponse)

	path := fmt.Sprintf("/_plugins/_sm/policies/%s", policyName)

	client, err := getOpenSearchClient(m.(*ProviderConf))
	if err != nil {
		return *response, err
	}

	// Build request
	req, err := http.NewRequest("GET", client.config.rawUrl+path, nil)
	if err != nil {
		return *response, fmt.Errorf("error building GET request: %w", err)
	}

	// Execute request
	resp, err := client.Client.Client.Perform(req)
	if err != nil {
		return *response, fmt.Errorf("error getting policy: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return *response, fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return *response, fmt.Errorf("policy not found: %s", policyName)
	}

	if resp.StatusCode != http.StatusOK {
		return *response, fmt.Errorf("error getting policy: received status code %d, body: %s", resp.StatusCode, string(body))
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return *response, fmt.Errorf("error unmarshalling policy body: %+v: %+v", err, body)
	}

	normalizePolicy(response.Policy)

	return *response, nil
}

func resourceOpensearchPostPutSMPolicy(d *schema.ResourceData, m interface{}, method string) (*SMPolicyResponse, error) {
	response := new(SMPolicyResponse)
	policyJSON := d.Get("body").(string)
	seq := d.Get("seq_no").(int)
	primTerm := d.Get("primary_term").(int)
	params := url.Values{}

	if seq >= 0 && primTerm > 0 {
		params.Set("if_seq_no", strconv.Itoa(seq))
		params.Set("if_primary_term", strconv.Itoa(primTerm))
	}

	path := fmt.Sprintf("/_plugins/_sm/policies/%s?%s", d.Get("policy_name").(string), params.Encode())

	client, err := getOpenSearchClient(m.(*ProviderConf))
	if err != nil {
		return nil, err
	}

	// Execute request with retry logic
	opts := ismRetryOptions(client.config, "sm policy")
	resp, err := doRequestWithRetry(opts, func() (*http.Request, error) {
		req, err := http.NewRequest(method, client.config.rawUrl+path, strings.NewReader(policyJSON))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		return req, nil
	}, client.Client.Client.Perform)
	if err != nil {
		return response, fmt.Errorf("error %s policy: %+v : %+v", method, path, err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return response, fmt.Errorf("error reading response body: %w", err)
	}

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return response, fmt.Errorf("error %s policy: received status code %d, body: %s", method, resp.StatusCode, string(body))
	}

	if err := json.Unmarshal(body, response); err != nil {
		return response, fmt.Errorf("error unmarshalling policy body: %+v: %+v", err, body)
	}

	return response, nil
}

type SMPolicyResponse struct {
	PolicyID    string                 `json:"_id"`
	Version     int                    `json:"_version"`
	PrimaryTerm int                    `json:"_primary_term"`
	SeqNo       int                    `json:"_seq_no"`
	Policy      map[string]interface{} `json:"sm_policy"`
}

func smDiffSuppressPolicy(k, old, new string, d *schema.ResourceData) bool {
	var oo, no interface{}
	if err := json.Unmarshal([]byte(old), &oo); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(new), &no); err != nil {
		return false
	}

	om, ok := oo.(map[string]interface{})
	if ok {
		normalizePolicy(om)
	}

	nm, ok := no.(map[string]interface{})
	if ok {
		normalizePolicy(nm)
	}

	// Suppress diff of autogenerated fields by copying them to the old object
	if name, ok := om["name"]; ok {
		nm["name"] = name
	}

	if enabled_time, ok := om["enabled_time"]; ok {
		nm["enabled_time"] = enabled_time
	}

	if schedule, ok := om["schedule"]; ok {
		nm["schedule"] = schedule
	}

	return reflect.DeepEqual(oo, no)
}
