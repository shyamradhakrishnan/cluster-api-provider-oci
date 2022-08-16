package scope

import (
	infrastructurev1beta1 "github.com/oracle/cluster-api-provider-oci/api/v1beta1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

type OCIClusterBase interface {
	GetOCIResourceIdentifier() string
	GetDefinedTags() map[string]map[string]string
	GetCompartmentId() string
	GetFreeformTags() map[string]string
	GetDRG() *infrastructurev1beta1.DRG
	GetVCNPeering() *infrastructurev1beta1.VCNPeering
	GetVCN() *infrastructurev1beta1.VCN
	GetName() string
	GetAPIServerLB() *infrastructurev1beta1.LoadBalancer
	GetNetworkSpec() *infrastructurev1beta1.NetworkSpec
	SetControlPlaneEndpoint(endpoint clusterv1.APIEndpoint)
	GetOCIClusterStatus() *infrastructurev1beta1.OCIClusterStatus
}
