/*
 Copyright (c) 2021, 2022 Oracle and/or its affiliates.

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

      https://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package scope

import (
	"context"
	"fmt"

	"github.com/oracle/cluster-api-provider-oci/cloud/ociutil"
	expinfra1 "github.com/oracle/cluster-api-provider-oci/exp/api/v1beta1"
	"github.com/oracle/oci-go-sdk/v63/common"
	"github.com/oracle/oci-go-sdk/v63/core"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2/klogr"
	"k8s.io/utils/pointer"
	capierrors "sigs.k8s.io/cluster-api/errors"
	expclusterv1 "sigs.k8s.io/cluster-api/exp/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/patch"

	"github.com/go-logr/logr"
	infrastructurev1beta1 "github.com/oracle/cluster-api-provider-oci/api/v1beta1"
	"github.com/oracle/cluster-api-provider-oci/cloud/services/compute"
	"github.com/oracle/cluster-api-provider-oci/cloud/services/computemanagement"
	nlb "github.com/oracle/cluster-api-provider-oci/cloud/services/networkloadbalancer"
	"github.com/oracle/cluster-api-provider-oci/cloud/services/vcn"
	"github.com/pkg/errors"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const OCIMachinePoolKind = "OCIMachinePool"

// MachineScopeParams defines the params need to create a new MachineScope
type MachinePoolScopeParams struct {
	Logger                    *logr.Logger
	Cluster                   *clusterv1.Cluster
	MachinePool               *expclusterv1.MachinePool
	Client                    client.Client
	ComputeClient             compute.ComputeClient
	ComputeManagementClient   computemanagement.Client
	OCICluster                *infrastructurev1beta1.OCICluster
	OCIMachinePool            *expinfra1.OCIMachinePool
	VCNClient                 vcn.Client
	NetworkLoadBalancerClient nlb.NetworkLoadBalancerClient
	Machine                   *clusterv1.Machine
	OCIMachine                *infrastructurev1beta1.OCIMachine
}

type MachinePoolScope struct {
	*logr.Logger
	Client                    client.Client
	patchHelper               *patch.Helper
	Cluster                   *clusterv1.Cluster
	MachinePool               *expclusterv1.MachinePool
	ComputeClient             compute.ComputeClient
	ComputeManagementClient   computemanagement.Client
	OCICluster                *infrastructurev1beta1.OCICluster
	OCIMachinePool            *expinfra1.OCIMachinePool
	VCNClient                 vcn.Client
	NetworkLoadBalancerClient nlb.NetworkLoadBalancerClient
	Machine                   *clusterv1.Machine
	OCIMachine                *infrastructurev1beta1.OCIMachine
}

// NewMachinePoolScope creates a MachinePoolScope given the MachinePoolScopeParams
func NewMachinePoolScope(params MachinePoolScopeParams) (*MachinePoolScope, error) {
	if params.MachinePool == nil {
		return nil, errors.New("failed to generate new scope from nil MachinePool")
	}
	if params.OCICluster == nil {
		return nil, errors.New("failed to generate new scope from nil OCICluster")
	}

	if params.Logger == nil {
		log := klogr.New()
		params.Logger = &log
	}
	helper, err := patch.NewHelper(params.OCIMachinePool, params.Client)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init patch helper")
	}

	return &MachinePoolScope{
		Logger:                    params.Logger,
		Client:                    params.Client,
		ComputeClient:             params.ComputeClient,
		ComputeManagementClient:   params.ComputeManagementClient,
		Cluster:                   params.Cluster,
		OCICluster:                params.OCICluster,
		patchHelper:               helper,
		MachinePool:               params.MachinePool,
		OCIMachinePool:            params.OCIMachinePool,
		VCNClient:                 params.VCNClient,
		NetworkLoadBalancerClient: params.NetworkLoadBalancerClient,
		Machine:                   params.Machine,
	}, nil
}

// PatchObject persists the cluster configuration and status.
func (m *MachinePoolScope) PatchObject(ctx context.Context) error {
	return m.patchHelper.Patch(ctx, m.OCIMachinePool)
}

// Close closes the current scope persisting the cluster configuration and status.
func (m *MachinePoolScope) Close(ctx context.Context) error {
	return m.PatchObject(ctx)
}

// HasFailed returns true when the OCIMachinePool's Failure reason or Failure message is populated.
func (m *MachinePoolScope) HasFailed() bool {
	return m.OCIMachinePool.Status.FailureReason != nil || m.OCIMachinePool.Status.FailureMessage != nil
}

// GetInstanceConfigurationId returns the MachinePoolScope instance configuration id.
func (m *MachinePoolScope) GetInstanceConfigurationId() string {
	return m.OCIMachinePool.Status.InstanceConfigurationId
}

// SetInstanceConfigurationIdStatus sets the MachinePool InstanceConfigurationId status.
func (m *MachinePoolScope) SetInstanceConfigurationIdStatus(id string) {
	m.OCIMachinePool.Status.InstanceConfigurationId = id
}

// SetFailureMessage sets the OCIMachine status error message.
func (m *MachinePoolScope) SetFailureMessage(v error) {
	m.OCIMachinePool.Status.FailureMessage = pointer.StringPtr(v.Error())
}

// SetFailureReason sets the OCIMachine status error reason.
func (m *MachinePoolScope) SetFailureReason(v capierrors.MachineStatusError) {
	m.OCIMachinePool.Status.FailureReason = &v
}

// SetReady sets the OCIMachine Ready Status.
func (m *MachinePoolScope) SetReady() {
	m.OCIMachinePool.Status.Ready = true
}

func (m *MachinePoolScope) GetWorkerMachineSubnet() *string {
	for _, subnet := range m.OCICluster.Spec.NetworkSpec.Vcn.Subnets {
		fmt.Println("---- subnet", subnet.Name)
		if subnet.Role == infrastructurev1beta1.WorkerRole {
			// if a subnet name is defined, use the correct subnet
			return subnet.ID
		}
	}
	return nil
}

// GetBootstrapData returns the bootstrap data from the secret in the Machine's bootstrap.dataSecretName.
func (m *MachinePoolScope) GetBootstrapData() (string, error) {
	if m.MachinePool.Spec.Template.Spec.Bootstrap.DataSecretName == nil {
		return "", errors.New("error retrieving bootstrap data: linked MachinePool's bootstrap.dataSecretName is nil")
	}

	secret := &corev1.Secret{}
	key := types.NamespacedName{Namespace: m.MachinePool.Namespace, Name: *m.MachinePool.Spec.Template.Spec.Bootstrap.DataSecretName}
	if err := m.Client.Get(context.TODO(), key, secret); err != nil {
		return "", errors.Wrapf(err, "failed to retrieve bootstrap data secret for OCIMachinePool %s/%s", m.MachinePool.Namespace, m.MachinePool.Name)
	}

	value, ok := secret.Data["value"]
	if !ok {
		return "", errors.New("error retrieving bootstrap data: secret value key is missing")
	}
	return string(value), nil
}

// TODO: pull the following funcs out into different file

// IsResourceCreatedByClusterAPI determines if the instance was created by the cluster using the
// tags created at instance launch.
func (s *MachinePoolScope) IsResourceCreatedByClusterAPI(resourceFreeFormTags map[string]string) bool {
	tagsAddedByClusterAPI := ociutil.BuildClusterTags(string(s.OCICluster.GetOCIResourceIdentifier()))
	for k, v := range tagsAddedByClusterAPI {
		if resourceFreeFormTags[k] != v {
			return false
		}
	}
	return true
}

// GetInstanceConfigurationByDisplayName returns the existing LaunchTemplate or nothing if it doesn't exist.
// For now by name until we need the input to be something different.
func (m *MachinePoolScope) GetInstanceConfigurationIdBy(ctx context.Context) (string, error) {
	req := core.ListInstanceConfigurationsRequest{
		SortBy:        core.ListInstanceConfigurationsSortByDisplayname,
		CompartmentId: common.String(m.OCICluster.Spec.CompartmentId)}
	// TODO: will want to paginate to make sure we hit all the configurations (testing now assumes very few configs in compartment)
	resp, err := m.ComputeManagementClient.ListInstanceConfigurations(ctx, req)
	if err != nil {
		return "", err
	}
	if len(resp.Items) == 0 {
		return "", nil
	}
	for _, instance := range resp.Items {
		if m.IsResourceCreatedByClusterAPI(instance.FreeformTags) {
			return *instance.Id, nil
		}
	}
	return "", nil
}

// TODO: unexport this later
func (m *MachinePoolScope) GetFreeFormTags(ociCluster infrastructurev1beta1.OCICluster) map[string]string {
	tags := ociutil.BuildClusterTags(ociCluster.GetOCIResourceIdentifier())
	// first use cluster level tags, then override with machine level tags
	if ociCluster.Spec.FreeformTags != nil {
		for k, v := range ociCluster.Spec.FreeformTags {
			tags[k] = v
		}
	}

	return tags
}

func (m *MachinePoolScope) GetWorkerMachineNSG() *string {
	for _, nsg := range m.OCICluster.Spec.NetworkSpec.Vcn.NetworkSecurityGroups {
		if nsg.Role == infrastructurev1beta1.WorkerRole {
			// if an NSG name is defined, use the correct NSG
			return nsg.ID
		}
	}
	return nil
}
