package base

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v63/common"
	"github.com/pkg/errors"
)

var (
	// list of required headers for generation and parsing.
	requiredHeaders = []string{"date", "authorization"}

	optionalHeaders = []string{"opc-obo-token", "x-cross-tenancy-request"}
)

type Client struct {
	endpoint string
	client   common.BaseClient
	logger   *logr.Logger
	signer   common.HTTPRequestSigner
}

// NewBaseClient creates a new base client
func NewBaseClient(configProvider common.ConfigurationProvider, logger *logr.Logger) (*Client, error) {
	region, err := configProvider.Region()
	if err != nil {
		return nil, errors.New("more than one resources are affected by the work request to create the cluster")
	}

	endpoint := common.StringToRegion(region).EndpointForTemplate("containerengine", "containerengine.{region}.oci.{secondLevelDomain}")

	baseClient, err := common.NewClientWithConfig(configProvider)

	signer := common.DefaultRequestSigner(configProvider)
	baseClient.Host = endpoint

	return &Client{
		endpoint: endpoint,
		client:   baseClient,
		logger:   logger,
		signer:   signer,
	}, err
}

func (c *Client) GenerateToken(ctx context.Context, clusterID string) (string, error) {
	endpoint := fmt.Sprintf(
		"https://%s/cluster_request/%s",
		c.endpoint,
		clusterID)
	c.logger.Info(fmt.Sprintf("Containerengine endpoint is %s", endpoint))

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	req.Header.Set("date", time.Now().UTC().Format(http.TimeFormat))
	if err != nil {
		return "", err
	}

	err = c.signer.Sign(req)

	for k, header := range req.Header {
		c.logger.Info(fmt.Sprintf("header parameters", k, header))
	}
	if err != nil {
		return "", err
	}
	url := req.URL
	query := url.Query()
	for _, header := range requiredHeaders {
		query.Set(header, req.Header.Get(header))
	}
	url.RawQuery = query.Encode()
	resp, err := http.Get(url.String())
	if err != nil {
		return "", err
	}
	body, err := ioutil.ReadAll(resp.Body)
	c.logger.Info(fmt.Sprintf("Token is %s", string(body)))
	return string(body), nil
}
