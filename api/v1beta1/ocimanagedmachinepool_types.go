/*
 *
 * Copyright (c) 2022, Oracle and/or its affiliates.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 * /
 *
 */
package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// OCIManagedMachinePoolSpec defines the desired state of an OCI managed machine pool.
// An OCIManagedMachinePool translates to an OKE NodePool.
// The properties are generated from https://docs.oracle.com/en-us/iaas/api/#/en/containerengine/20180222/datatypes/CreateNodePoolDetails
type OCIManagedMachinePoolSpec struct {

	// Name of the OKE NodePool. Will bet set to the MachinePool name if not defined.
	// +optional
	Name string `json:"name,omitempty"`

	// NodePoolNodeConfig defines the configuration of nodes in the node pool.
	// +optional
	NodePoolNodeConfig NodePoolNodeConfig `json:"nodePoolNodeConfig,omitempty"`

	// NodeEvictionNodePoolSettings defines the eviction settings.
	// +optional
	NodeEvictionNodePoolSettings NodeEvictionNodePoolSettings `json:"nodeEvictionNodePoolSettings,omitempty"`

	// NodeShape defines the name of the node shape of the nodes in the node pool.
	// +optional
	NodeShape string `json:"nodeShape,omitempty"`

	// NodeShapeConfig defines the configuration of the shape to launch nodes in the node pool.
	// +optional
	NodeShapeConfig NodeShapeConfig `json:"nodeShapeConfig,omitempty"`

	// NodeSourceViaImage defines the image configuration of the nodes in the nodepool.
	// +optional
	NodeSourceViaImage NodeSourceViaImage `json:"nodeSourceViaImage,omitempty"`

	// SshPublicKey defines the SSH public key on each node in the node pool on launch.
	// +optional
	SshPublicKey string `json:"sshPublicKey,omitempty"`

	// NodeMetadata defines a list of key/value pairs to add to each underlying OCI instance in the node pool on launch.
	// +optional
	NodeMetadata map[string]string `json:"nodeMetadata,omitempty"`
}

// NodePoolNodeConfig describes the configuration of nodes in the node pool.
type NodePoolNodeConfig struct {

	// IsPvEncryptionInTransitEnabled defines whether in transit encryption should be enabled on the nodes.
	// +optional
	IsPvEncryptionInTransitEnabled bool `json:"isPvEncryptionInTransitEnabled,omitempty"`

	// KmsKeyId  defines whether in transit encryption should be enabled on the nodes.
	// +optional
	KmsKeyId string `json:"kmsKeyId,omitempty"`

	// PlacementConfigs defines the placement configurations for the node pool.
	// +optional
	PlacementConfigs PlacementConfigs `json:"placementConfigs,omitempty"`

	// MemoryInGBs defines the total amount of memory available to each node, in gigabytes.
	// +optional
	MemoryInGBs string `json:"memoryInGBs,omitempty"`

	// NsgNames defines the names of NSGs which will be associated with the nodes. the NSGs are defined
	// in OCIManagedCluster object.
	// +optional
	NsgNames []string `json:"nsgNames,omitempty"`
}

// NodePoolPodNetworkOptionDetails describes the CNI related configuration of pods in the node pool.
type NodePoolPodNetworkOptionDetails struct {

	// CniType describes the CNI plugin used by this node pool. Allowed values are OCI_VCN_IP_NATIVE and FLANNEL_OVERLAY.
	// +optional
	CniType string `json:"cniType,omitempty"`

	// VcnIpNativePodNetworkOptions describes the network options specific to using the OCI VCN Native CNI
	// +optional
	VcnIpNativePodNetworkOptions string `json:"vcnIpNativePodNetworkOptions,omitempty"`
}

// VcnIpNativePodNetworkOptions defines the Network options specific to using the OCI VCN Native CNI
type VcnIpNativePodNetworkOptions struct {

	// MemoryInGBs defines the max number of pods per node in the node pool. This value will be limited by the
	// number of VNICs attachable to the node pool shape
	// +optional
	MaxPodsPerNode string `json:"maxPodsPerNode,omitempty"`
}

// PlacementConfigs defines the placement configurations for the node pool.
type PlacementConfigs struct {

	// AvailabilityDomain defines the availability domain in which to place nodes.
	// +optional
	AvailabilityDomain string `json:"availabilityDomain,omitempty"`

	// CapacityReservationId defines the OCID of the compute capacity reservation in which to place the compute instance.
	// +optional
	CapacityReservationId string `json:"capacityReservationId,omitempty"`

	// FaultDomains defines the list of fault domains in which to place nodes.
	// +optional
	FaultDomains []string `json:"faultDomains,omitempty"`

	// SubnetName defines the name of the subnet which need ot be associated with the Nodepool.
	// The subnets are defined in the OCiManagedCluster object.
	// +optional
	SubnetName []string `json:"subnetName,omitempty"`
}

// NodeEvictionNodePoolSettings defines the Node Eviction Details configuration.
type NodeEvictionNodePoolSettings struct {

	// EvictionGraceDuration defines the duration after which OKE will give up eviction of the pods on the node. PT0M will indicate you want to delete the node without cordon and drain. Default PT60M, Min PT0M, Max: PT60M. Format ISO 8601 e.g PT30M
	// +optional
	EvictionGraceDuration string `json:"capacityReservationId,omitempty"`

	// IsForceDeleteAfterGraceDuration defines if the underlying compute instance should be deleted if you cannot evict all the pods in grace period
	// +optional
	IsForceDeleteAfterGraceDuration bool `json:"isForceDeleteAfterGraceDuration,omitempty"`
}

// NodeShapeConfig defines the shape configuration of the nodes.
type NodeShapeConfig struct {

	// MemoryInGBs defines the total amount of memory available to each node, in gigabytes.
	// +optional
	MemoryInGBs string `json:"memoryInGBs,omitempty"`

	// Ocpus defines the total number of OCPUs available to each node in the node pool.
	// +optional
	Ocpus string `json:"ocpus,omitempty"`
}

// NodeSourceViaImage defines the Details of the image running on the node.
type NodeSourceViaImage struct {

	// BootVolumeSizeInGBs defines the size of the boot volume in GBs.
	// +optional
	BootVolumeSizeInGBs int64 `json:"bootVolumeSizeInGBs,omitempty"`

	// ImageId defines the OCID of the image used to boot the node.
	// +optional
	ImageId string `json:"imageId,omitempty"`
}

// OCIManagedMachinePoolStatus defines the observed state of OCIManagedMachinePool
type OCIManagedMachinePoolStatus struct {
	// +optional
	Ready bool `json:"ready"`
	// NetworkSpec encapsulates all things related to OCI network.
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// OCIManagedMachinePool is the Schema for the ocimanagedmachinepool API.
type OCIManagedMachinePool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OCIManagedMachinePoolSpec   `json:"spec,omitempty"`
	Status OCIManagedMachinePoolStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OCIManagedMachinePoolList contains a list of OCIManagedMachinePool.
type OCIManagedMachinePoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OCICluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OCIManagedMachinePool{}, &OCIManagedMachinePoolList{})
}
