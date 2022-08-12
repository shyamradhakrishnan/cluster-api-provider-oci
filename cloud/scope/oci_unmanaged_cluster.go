package scope

import (
	infrastructurev1beta1 "github.com/oracle/cluster-api-provider-oci/api/v1beta1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

type OCIUnmanagedCluster struct {
	OCICluster *infrastructurev1beta1.OCICluster
}

func (c OCIUnmanagedCluster) GetOCIResourceIdentifier() string {
	return c.OCICluster.Spec.OCIResourceIdentifier
}

func (c OCIUnmanagedCluster) GetName() string {
	return c.OCICluster.Name
}

func (c OCIUnmanagedCluster) GetOCIClusterStatus() *infrastructurev1beta1.OCIClusterStatus {
	return &c.OCICluster.Status
}

func (c OCIUnmanagedCluster) GetDefinedTags() map[string]map[string]string {
	return c.OCICluster.Spec.DefinedTags
}

func (c OCIUnmanagedCluster) GetCompartmentId() string {
	return c.OCICluster.Spec.CompartmentId
}

func (c OCIUnmanagedCluster) GetFreeformTags() map[string]string {
	return c.OCICluster.Spec.FreeformTags
}

func (c OCIUnmanagedCluster) GetDRG() *infrastructurev1beta1.DRG {
	return c.OCICluster.Spec.NetworkSpec.VCNPeering.DRG
}

func (c OCIUnmanagedCluster) GetVCNPeering() *infrastructurev1beta1.VCNPeering {
	return c.OCICluster.Spec.NetworkSpec.VCNPeering
}

func (c OCIUnmanagedCluster) GetVCN() *infrastructurev1beta1.VCN {
	return &c.OCICluster.Spec.NetworkSpec.Vcn
}

func (c OCIUnmanagedCluster) GetAPIServerLB() *infrastructurev1beta1.LoadBalancer {
	return &c.OCICluster.Spec.NetworkSpec.APIServerLB
}

func (c OCIUnmanagedCluster) GetNetworkSpec() *infrastructurev1beta1.NetworkSpec {
	return &c.OCICluster.Spec.NetworkSpec
}

func (c OCIUnmanagedCluster) SetControlPlaneEndpoint(endpoint clusterv1.APIEndpoint) {
	c.OCICluster.Spec.ControlPlaneEndpoint = endpoint
}
