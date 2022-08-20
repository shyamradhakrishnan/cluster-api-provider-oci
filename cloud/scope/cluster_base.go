package scope

import (
	infrastructurev1beta1 "github.com/oracle/cluster-api-provider-oci/api/v1beta1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

type OCIClusterAccessor interface {
	GetOCIResourceIdentifier() string
	GetDefinedTags() map[string]map[string]string
	GetCompartmentId() string
	GetFreeformTags() map[string]string
	GetName() string
	GetNetworkSpec() *infrastructurev1beta1.NetworkSpec
	SetControlPlaneEndpoint(endpoint clusterv1.APIEndpoint)
	GetOCIClusterStatus() *infrastructurev1beta1.OCIClusterStatus
}
