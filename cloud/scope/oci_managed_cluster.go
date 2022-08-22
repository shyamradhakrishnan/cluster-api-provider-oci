package scope

import (
	infrastructurev1beta1 "github.com/oracle/cluster-api-provider-oci/api/v1beta1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

type OCIManagedCluster struct {
	OCIManagedCluster *infrastructurev1beta1.OCIManagedCluster
}

func (c OCIManagedCluster) GetOCIResourceIdentifier() string {
	return c.OCIManagedCluster.Spec.OCIResourceIdentifier
}

func (c OCIManagedCluster) GetName() string {
	return c.OCIManagedCluster.Name
}

func (c OCIManagedCluster) GetOCIClusterStatus() *infrastructurev1beta1.OCIClusterStatus {
	return nil
}

func (c OCIManagedCluster) GetDefinedTags() map[string]map[string]string {
	return c.OCIManagedCluster.Spec.DefinedTags
}

func (c OCIManagedCluster) GetCompartmentId() string {
	return c.OCIManagedCluster.Spec.CompartmentId
}

func (c OCIManagedCluster) GetFreeformTags() map[string]string {
	return c.OCIManagedCluster.Spec.FreeformTags
}

func (c OCIManagedCluster) GetDRG() *infrastructurev1beta1.DRG {
	return c.OCIManagedCluster.Spec.NetworkSpec.VCNPeering.DRG
}

func (c OCIManagedCluster) GetVCNPeering() *infrastructurev1beta1.VCNPeering {
	return c.OCIManagedCluster.Spec.NetworkSpec.VCNPeering
}

func (c OCIManagedCluster) GetAPIServerLB() *infrastructurev1beta1.LoadBalancer {
	return nil
}

func (c OCIManagedCluster) GetNetworkSpec() *infrastructurev1beta1.NetworkSpec {
	return &c.OCIManagedCluster.Spec.NetworkSpec
}

func (c OCIManagedCluster) SetControlPlaneEndpoint(endpoint clusterv1.APIEndpoint) {
	c.OCIManagedCluster.Spec.ControlPlaneEndpoint = endpoint
}
