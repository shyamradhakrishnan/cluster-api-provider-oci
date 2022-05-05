package computemanagement

import (
	"context"

	"github.com/oracle/oci-go-sdk/v63/core"
)

type ComputeClient interface {
	CreateInstancePool(ctx context.Context, request core.CreateInstancePoolRequest) (response core.CreateInstancePoolResponse, err error)
	GetInstancePool(ctx context.Context, request core.GetInstancePoolInstanceRequest) (response core.GetInstancePoolInstanceResponse, err error)
	TerminateInstancePool(ctx context.Context, request core.TerminateInstancePoolRequest) (response core.TerminateInstancePoolResponse, err error)
	//UpdateInstancePool
	//GetInstancePoolInstance
}
