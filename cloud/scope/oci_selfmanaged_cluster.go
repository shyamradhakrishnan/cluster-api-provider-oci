package scope

import (
	infrastructurev1beta1 "github.com/oracle/cluster-api-provider-oci/api/v1beta1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

type OCISelfManagedCluster struct {
	OCICluster *infrastructurev1beta1.OCICluster
}

func (c OCISelfManagedCluster) GetOCIResourceIdentifier() string {
	return c.OCICluster.Spec.OCIResourceIdentifier
}

func (c OCISelfManagedCluster) GetName() string {
	return c.OCICluster.Name
}

func (c OCISelfManagedCluster) GetOCIClusterStatus() *infrastructurev1beta1.OCIClusterStatus {
	return &c.OCICluster.Status
}

func (c OCISelfManagedCluster) GetDefinedTags() map[string]map[string]string {
	return c.OCICluster.Spec.DefinedTags
}

func (c OCISelfManagedCluster) GetCompartmentId() string {
	return c.OCICluster.Spec.CompartmentId
}

func (c OCISelfManagedCluster) GetFreeformTags() map[string]string {
	return c.OCICluster.Spec.FreeformTags
}

func (c OCISelfManagedCluster) GetDRG() *infrastructurev1beta1.DRG {
	return c.OCICluster.Spec.NetworkSpec.VCNPeering.DRG
}

func (c OCISelfManagedCluster) GetVCNPeering() *infrastructurev1beta1.VCNPeering {
	return c.OCICluster.Spec.NetworkSpec.VCNPeering
}

func (c OCISelfManagedCluster) GetAPIServerLB() *infrastructurev1beta1.LoadBalancer {
	return &c.OCICluster.Spec.NetworkSpec.APIServerLB
}

func (c OCISelfManagedCluster) GetNetworkSpec() *infrastructurev1beta1.NetworkSpec {
	return &c.OCICluster.Spec.NetworkSpec
}

func (c OCISelfManagedCluster) SetControlPlaneEndpoint(endpoint clusterv1.APIEndpoint) {
	c.OCICluster.Spec.ControlPlaneEndpoint = endpoint
}
