package scope

import (
	infrastructurev1beta1 "github.com/oracle/cluster-api-provider-oci/api/v1beta1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

type OCIManagedCluster struct {
	OCICluster *infrastructurev1beta1.OCIManagedCluster
}

func (c OCIManagedCluster) GetOCIResourceIdentifier() string {
	return c.OCICluster.Spec.OCIResourceIdentifier
}

func (c OCIManagedCluster) GetName() string {
	return c.OCICluster.Name
}

func (c OCIManagedCluster) GetOCIClusterStatus() *infrastructurev1beta1.OCIClusterStatus {
	return nil
}

func (c OCIManagedCluster) GetDefinedTags() map[string]map[string]string {
	return c.OCICluster.Spec.DefinedTags
}

func (c OCIManagedCluster) GetCompartmentId() string {
	return c.OCICluster.Spec.CompartmentId
}

func (c OCIManagedCluster) GetFreeformTags() map[string]string {
	return c.OCICluster.Spec.FreeformTags
}

func (c OCIManagedCluster) GetDRG() *infrastructurev1beta1.DRG {
	return c.OCICluster.Spec.NetworkSpec.VCNPeering.DRG
}

func (c OCIManagedCluster) GetVCNPeering() *infrastructurev1beta1.VCNPeering {
	return c.OCICluster.Spec.NetworkSpec.VCNPeering
}

func (c OCIManagedCluster) GetVCN() *infrastructurev1beta1.VCN {
	return &c.OCICluster.Spec.NetworkSpec.Vcn
}

func (c OCIManagedCluster) GetAPIServerLB() *infrastructurev1beta1.LoadBalancer {
	return nil
}

func (c OCIManagedCluster) GetNetworkSpec() *infrastructurev1beta1.NetworkSpec {
	return &c.OCICluster.Spec.NetworkSpec
}

func (c OCIManagedCluster) SetControlPlaneEndpoint(endpoint clusterv1.APIEndpoint) {
	c.OCICluster.Spec.ControlPlaneEndpoint = endpoint
}
