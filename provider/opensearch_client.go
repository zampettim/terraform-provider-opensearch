package provider

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/opensearch-project/opensearch-go/v4"
	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
)

type OpenSearchClient struct {
	*opensearchapi.Client
	conf *ProviderConf
}

func (c *OpenSearchClient) Perform(req *http.Request) (*http.Response, error) {
	return c.Client.Client.Perform(req)
}

func NewOpenSearchClient(conf *ProviderConf) (*OpenSearchClient, error) {
	cfg := opensearch.Config{
		Addresses:            []string{conf.rawUrl},
		MaxRetries:           conf.maxRetries,
		EnableRetryOnTimeout: true,
		RetryOnStatus:        []int{http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout},
		RetryBackoff: func(attempt int) time.Duration {
			return time.Duration(1<<attempt) * time.Duration(conf.retryBackoffInitialMs) * time.Millisecond
		},
	}

	if conf.parsedUrl != nil && conf.parsedUrl.User != nil {
		username := conf.parsedUrl.User.Username()
		password, _ := conf.parsedUrl.User.Password()
		cfg.Username = username
		cfg.Password = password
	}
	if conf.username != "" && conf.password != "" {
		cfg.Username = conf.username
		cfg.Password = conf.password
	}
	if conf.token != "" && !conf.signAWSRequests && cfg.Username == "" {
		cfg.Header = make(http.Header)
		cfg.Header.Set("Authorization", conf.tokenName+" "+conf.token)
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	tlsConfig := &tls.Config{InsecureSkipVerify: conf.insecure}
	transport.TLSClientConfig = tlsConfig

	if conf.cacertFile != "" {
		caCert, _, err := readPathOrContent(conf.cacertFile)
		if err != nil {
			return nil, err
		}
		caCertPool, err := x509.SystemCertPool()
		if err != nil {
			caCertPool = x509.NewCertPool()
		}
		if !caCertPool.AppendCertsFromPEM([]byte(caCert)) {
			return nil, fmt.Errorf("failed to append certificates from cacert_file: no valid certificates found")
		}
		tlsConfig.RootCAs = caCertPool
	}

	if conf.certPemPath != "" && conf.keyPemPath != "" {
		certPem, _, err := readPathOrContent(conf.certPemPath)
		if err != nil {
			return nil, err
		}
		keyPem, _, err := readPathOrContent(conf.keyPemPath)
		if err != nil {
			return nil, err
		}
		cert, err := tls.X509KeyPair([]byte(certPem), []byte(keyPem))
		if err != nil {
			return nil, err
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}
	if conf.hostOverride != "" {
		tlsConfig.ServerName = conf.hostOverride
	}
	if conf.proxy != "" {
		proxyURL, err := url.Parse(conf.proxy)
		if err != nil {
			return nil, err
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}
	cfg.Transport = transport
	if conf.certPemPath != "" || conf.hostOverride != "" || conf.proxy != "" {
		cfg.Transport = &hostOverrideTransport{base: transport, hostOverride: conf.hostOverride}
	}

	if conf.signAWSRequests {
		awsRegion, awsService := resolveOpenSearchAWSConfig(conf)
		if awsRegion != "" {
			originalRegion, originalService := conf.awsRegion, conf.awsSig4Service
			conf.awsRegion, conf.awsSig4Service = awsRegion, awsService
			signer, err := newAWSSigner(conf)
			conf.awsRegion, conf.awsSig4Service = originalRegion, originalService
			if err != nil {
				return nil, err
			}
			cfg.Signer = signer
		}
	}

	client, err := opensearchapi.NewClient(opensearchapi.Config{Client: cfg})
	if err != nil {
		return nil, fmt.Errorf("error creating opensearch client: %w", err)
	}

	return &OpenSearchClient{
		Client: client,
		conf:   conf,
	}, nil
}

func resolveOpenSearchAWSConfig(conf *ProviderConf) (string, string) {
	region, service := conf.awsRegion, conf.awsSig4Service
	if service == "" {
		service = "es"
	}
	if conf.parsedUrl == nil {
		return region, service
	}
	if match := awsUrlRegexp.FindStringSubmatch(conf.parsedUrl.Hostname()); match != nil {
		if region == "" {
			region = match[1]
		}
		return region, "es"
	}
	if match := awsOpensearchServerlessUrlRegexp.FindStringSubmatch(conf.parsedUrl.Hostname()); match != nil {
		if region == "" {
			region = match[1]
		}
		return region, "aoss"
	}
	return region, service
}

type hostOverrideTransport struct {
	base         http.RoundTripper
	hostOverride string
}

func (t *hostOverrideTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.hostOverride != "" {
		req.Host = t.hostOverride
	}
	return t.base.RoundTrip(req)
}

type ServerInfo struct {
	Version struct {
		Number       string
		Distribution string
	}
}

func (c *OpenSearchClient) Info(ctx context.Context, options map[string]string) (*ServerInfo, error) {
	request := &opensearchapi.InfoReq{}
	for key, value := range options {
		switch key {
		case "pretty":
			request.Params.Pretty = value == "true"
		case "human":
			request.Params.Human = value == "true"
		case "error_trace":
			request.Params.ErrorTrace = value == "true"
		case "filter_path":
			request.Params.FilterPath = []string{value}
		}
	}

	response, err := c.Client.Info(ctx, request)
	if err != nil {
		return nil, err
	}

	return &ServerInfo{Version: struct {
		Number       string
		Distribution string
	}{
		Number:       response.Version.Number,
		Distribution: response.Version.Distribution,
	}}, nil
}
