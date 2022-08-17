package base

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v63/common"
	"github.com/pkg/errors"
)

var (
	// list of required headers for generation and parsing.
	requiredHeaders = []string{"date", "authorization"}
)

type Client struct {
	endpoint string
	logger   *logr.Logger
	signer   common.HTTPRequestSigner
}

// NewBaseClient creates a new base client
func NewBaseClient(configProvider common.ConfigurationProvider, logger *logr.Logger) (*Client, error) {
	region, err := configProvider.Region()
	if err != nil {
		return nil, errors.New("more than one resources are affected by the work request to create the cluster")
	}

	endpoint := common.StringToRegion(region).EndpointForTemplate("containerengine", "containerengine.{region}.{secondLevelDomain}")
	signer := common.DefaultRequestSigner(configProvider)

	return &Client{
		endpoint: endpoint,
		logger:   logger,
		signer:   signer,
	}, err
}

func (c *Client) GenerateToken(ctx context.Context, clusterID string) (string, error) {
	endpoint := fmt.Sprintf(
		"https://%s/cluster_request/%s",
		c.endpoint,
		clusterID)

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("date", time.Now().UTC().Format(http.TimeFormat))
	err = c.signer.Sign(req)
	if err != nil {
		return "", err
	}
	url := req.URL
	query := url.Query()
	for _, header := range requiredHeaders {
		query.Set(header, req.Header.Get(header))
	}
	url.RawQuery = query.Encode()
	return base64.URLEncoding.EncodeToString([]byte(url.String())), nil
}
