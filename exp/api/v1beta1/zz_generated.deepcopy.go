//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
Copyright (c) 2022 Oracle and/or its affiliates.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by controller-gen. DO NOT EDIT.

package v1beta1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
	apiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/errors"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InstanceConfiguration) DeepCopyInto(out *InstanceConfiguration) {
	*out = *in
	if in.InstanceConfigurationId != nil {
		in, out := &in.InstanceConfigurationId, &out.InstanceConfigurationId
		*out = new(string)
		**out = **in
	}
	out.InstanceDetails = in.InstanceDetails
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InstanceConfiguration.
func (in *InstanceConfiguration) DeepCopy() *InstanceConfiguration {
	if in == nil {
		return nil
	}
	out := new(InstanceConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InstanceDetails) DeepCopyInto(out *InstanceDetails) {
	*out = *in
	out.SourceDetails = in.SourceDetails
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InstanceDetails.
func (in *InstanceDetails) DeepCopy() *InstanceDetails {
	if in == nil {
		return nil
	}
	out := new(InstanceDetails)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LaunchDetails) DeepCopyInto(out *LaunchDetails) {
	*out = *in
	out.SourceDetails = in.SourceDetails
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LaunchDetails.
func (in *LaunchDetails) DeepCopy() *LaunchDetails {
	if in == nil {
		return nil
	}
	out := new(LaunchDetails)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OCIMachinePool) DeepCopyInto(out *OCIMachinePool) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OCIMachinePool.
func (in *OCIMachinePool) DeepCopy() *OCIMachinePool {
	if in == nil {
		return nil
	}
	out := new(OCIMachinePool)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *OCIMachinePool) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OCIMachinePoolList) DeepCopyInto(out *OCIMachinePoolList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]OCIMachinePool, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OCIMachinePoolList.
func (in *OCIMachinePoolList) DeepCopy() *OCIMachinePoolList {
	if in == nil {
		return nil
	}
	out := new(OCIMachinePoolList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *OCIMachinePoolList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OCIMachinePoolSpec) DeepCopyInto(out *OCIMachinePoolSpec) {
	*out = *in
	if in.ProviderID != nil {
		in, out := &in.ProviderID, &out.ProviderID
		*out = new(string)
		**out = **in
	}
	in.InstanceConfiguration.DeepCopyInto(&out.InstanceConfiguration)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OCIMachinePoolSpec.
func (in *OCIMachinePoolSpec) DeepCopy() *OCIMachinePoolSpec {
	if in == nil {
		return nil
	}
	out := new(OCIMachinePoolSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OCIMachinePoolStatus) DeepCopyInto(out *OCIMachinePoolStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make(apiv1beta1.Conditions, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.FailureReason != nil {
		in, out := &in.FailureReason, &out.FailureReason
		*out = new(errors.MachineStatusError)
		**out = **in
	}
	if in.FailureMessage != nil {
		in, out := &in.FailureMessage, &out.FailureMessage
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OCIMachinePoolStatus.
func (in *OCIMachinePoolStatus) DeepCopy() *OCIMachinePoolStatus {
	if in == nil {
		return nil
	}
	out := new(OCIMachinePoolStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SourceDetails) DeepCopyInto(out *SourceDetails) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SourceDetails.
func (in *SourceDetails) DeepCopy() *SourceDetails {
	if in == nil {
		return nil
	}
	out := new(SourceDetails)
	in.DeepCopyInto(out)
	return out
}
