package containerengine

import (
	"context"

	"github.com/oracle/oci-go-sdk/v63/containerengine"
)

type Client interface {
	CreateCluster(ctx context.Context, request containerengine.CreateClusterRequest) (response containerengine.CreateClusterResponse, err error)
	GetCluster(ctx context.Context, request containerengine.GetClusterRequest) (response containerengine.GetClusterResponse, err error)
	UpdateCluster(ctx context.Context, request containerengine.UpdateClusterRequest) (response containerengine.UpdateClusterResponse, err error)
	ListClusters(ctx context.Context, request containerengine.ListClustersRequest) (response containerengine.ListClustersResponse, err error)
	GetWorkRequest(ctx context.Context, request containerengine.GetWorkRequestRequest) (response containerengine.GetWorkRequestResponse, err error)
	DeleteCluster(ctx context.Context, request containerengine.DeleteClusterRequest) (response containerengine.DeleteClusterResponse, err error)
	CreateKubeconfig(ctx context.Context, request containerengine.CreateKubeconfigRequest) (response containerengine.CreateKubeconfigResponse, err error)
}
