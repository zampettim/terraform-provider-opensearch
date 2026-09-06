package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/opensearch-project/opensearch-go/v4"
)

type OpenSearchClient struct {
	client *opensearch.Client
	conf   *ProviderConf
}

func (c *OpenSearchClient) Perform(req *http.Request) (*http.Response, error) {
	return c.client.Perform(req)
}

func NewOpenSearchClient(conf *ProviderConf) (*OpenSearchClient, error) {
	cfg := opensearch.Config{
		Addresses: []string{conf.rawUrl},
	}

	if conf.username != "" && conf.password != "" {
		cfg.Username = conf.username
		cfg.Password = conf.password
	} else if conf.parsedUrl.User != nil {
		username := conf.parsedUrl.User.Username()
		password, _ := conf.parsedUrl.User.Password()
		cfg.Username = username
		cfg.Password = password
	}

	client, err := opensearch.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("error creating opensearch client: %w", err)
	}

	return &OpenSearchClient{
		client: client,
		conf:   conf,
	}, nil
}

type ServerInfo struct {
	Version struct {
		Number       string
		Distribution string
	}
}

func (c *OpenSearchClient) Info(ctx context.Context, options map[string]string) (*ServerInfo, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.conf.rawUrl, nil)
	if err != nil {
		return nil, err
	}

	query := request.URL.Query()
	for key, value := range options {
		query.Set(key, value)
	}
	request.URL.RawQuery = query.Encode()

	res, err := c.Perform(request)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("info request failed with status %s", res.Status)
	}

	var info ServerInfo
	if err := json.NewDecoder(res.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("error decoding info response: %w", err)
	}

	return &info, nil
}
