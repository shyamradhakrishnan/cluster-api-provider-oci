package base

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v63/common"
	"github.com/pkg/errors"
)

type Client struct {
	endpoint string
	client   common.BaseClient
	logger   *logr.Logger
}

// NewBaseClient creates a new base client
func NewBaseClient(configProvider common.ConfigurationProvider, logger *logr.Logger) (*Client, error) {
	region, err := configProvider.Region()
	if err != nil {
		return nil, errors.New("more than one resources are affected by the work request to create the cluster")
	}

	endpoint := common.StringToRegion(region).EndpointForTemplate("containerengine", "https://containerengine.{region}.oci.{secondLevelDomain}")

	baseClient, err := common.NewClientWithConfig(configProvider)
	baseClient.Host = endpoint

	return &Client{
		endpoint: endpoint,
		client:   baseClient,
		logger:   logger,
	}, err
}

func (c *Client) GenerateToken(ctx context.Context, clusterID string) (string, error) {
	endpoint := fmt.Sprintf(
		"https://%s/cluster_request/%s",
		c.endpoint,
		clusterID)
	c.logger.Info(fmt.Sprintf("Containerengine endpoint is %s", endpoint))
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	resp, err := c.client.Call(ctx, req)
	if err != nil {
		return "", err
	}
	body, err := ioutil.ReadAll(resp.Body)
	return base64.URLEncoding.EncodeToString([]byte(body)), nil
}
