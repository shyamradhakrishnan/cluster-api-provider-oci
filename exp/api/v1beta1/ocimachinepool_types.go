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
	"sigs.k8s.io/cluster-api/errors"
)

// +kubebuilder:object:generate=true
// +groupName=infrastructure.cluster.x-k8s.io

// Constants block.
const (
	// MachinePoolFinalizer is the finalizer for the machine pool.
	MachinePoolFinalizer = "ocimachinepool.infrastructure.cluster.x-k8s.io"
)

// OCIMachinePoolSpec defines the desired state of OCIMachinePool
type OCIMachinePoolSpec struct {
	// ProviderID is the ARN of the associated InstancePool
	// +optional
	ProviderID *string `json:"providerID,omitempty"`

	// MinSize defines the minimum size of the group.
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=1
	MinSize int32 `json:"minSize"`

	// MaxSize defines the maximum size of the group.
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=1
	MaxSize int32 `json:"maxSize"`

	// Custom metadata key/value pairs that you provide, such as the SSH public key
	// required to connect to the instance.
	Metadata map[string]string `json:"metadata,omitempty"`

	//TODO Add
	// Ad
	// subnet
	// tags
	// MachineTemplateSpec
	// MachineDeploymentStrategy
	// ProviderIDList

	InstanceConfiguration InstanceConfiguration `json:"instanceConfiguration,omitempty"`
}

type InstanceConfiguration struct {
	// displayName
	//freeformTags
	InstanceConfigurationId *string         `json:"instanceConfigurationId,omitempty"`
	InstanceDetails         InstanceDetails `json:"instanceDetails,omitempty"`
}

type InstanceDetails struct {
	Shape         string        `json:"shape,omitempty"`
	SourceDetails LaunchDetails `json:"launchDetails,omitempty"`
}

// LaunchDetails Instance launch details for creating an instance from an instance configuration
// https://docs.oracle.com/en-us/iaas/api/#/en/iaas/20160918/datatypes/InstanceConfigurationLaunchInstanceDetails
type LaunchDetails struct {
	//availabilityDomain
	//displayName
	//freeformTags
	//shapeConfig

	// Custom metadata key/value pairs that you provide, such as the SSH public key
	// required to connect to the instance.
	Metadata map[string]string `json:"metadata,omitempty"`

	Shape         string        `json:"shape,omitempty"`
	SourceDetails SourceDetails `json:"sourceDetails,omitempty"`
}

// https://docs.oracle.com/en-us/iaas/api/#/en/iaas/20160918/datatypes/InstanceConfigurationLaunchInstanceShapeConfigDetails
// type ShapeConfig struct {
// }

// https://docs.oracle.com/en-us/iaas/api/#/en/iaas/20160918/datatypes/InstanceConfigurationLaunchInstanceShapeConfigDetails
// type ShapeConfigDetails struct {
// }

// SourceDetails source details for instance launched from instance configuration
// https://docs.oracle.com/en-us/iaas/api/#/en/iaas/20160918/datatypes/InstanceConfigurationInstanceSourceViaImageDetails
type SourceDetails struct {
	// bootVolumeSizeInGBs

	// OCID of the image to be used to launch the instance
	ImageId string `json:"imageId,omitempty"`
}

// OCIMachinePoolStatus defines the observed state of OCIMachinePool
type OCIMachinePoolStatus struct {
	// Ready is true when the provider resource is ready.
	// +optional
	Ready bool `json:"ready"`

	// Replicas is the most recently observed number of replicas
	// +optional
	Replicas int32 `json:"replicas"`

	// The ID of the Instance Configuration
	InstanceConfigurationId string `json:"instanceConfigurationId,omitempty"`

	// Todo
	// ReadyReplicas
	// UnavailableReplicas
	// InfrastructureReady

	// Conditions defines current service state of the OCIMachinePool.
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`

	FailureReason *errors.MachineStatusError `json:"failureReason,omitempty"`

	FailureMessage *string `json:"failureMessage,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

type OCIMachinePool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OCIMachinePoolSpec   `json:"spec,omitempty"`
	Status OCIMachinePoolStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// OCIMachinePoolList contains a list of OCIMachinePool.
type OCIMachinePoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OCIMachinePool `json:"items"`
}

// GetConditions returns the list of conditions for an OCIMachine API object.
func (m *OCIMachinePool) GetConditions() clusterv1.Conditions {
	return m.Status.Conditions
}

// SetConditions will set the given conditions on an OCIMachine object.
func (m *OCIMachinePool) SetConditions(conditions clusterv1.Conditions) {
	m.Status.Conditions = conditions
}

func init() {
	SchemeBuilder.Register(&OCIMachinePool{}, &OCIMachinePoolList{})
}
