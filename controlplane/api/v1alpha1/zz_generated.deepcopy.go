//go:build !ignore_autogenerated

/*
Copyright 2024.

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

package v1alpha1

import (
	"github.com/openshift/assisted-service/api/hiveextension/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	apiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OpenshiftAssistedControlPlane) DeepCopyInto(out *OpenshiftAssistedControlPlane) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OpenshiftAssistedControlPlane.
func (in *OpenshiftAssistedControlPlane) DeepCopy() *OpenshiftAssistedControlPlane {
	if in == nil {
		return nil
	}
	out := new(OpenshiftAssistedControlPlane)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *OpenshiftAssistedControlPlane) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OpenshiftAssistedControlPlaneConfigSpec) DeepCopyInto(out *OpenshiftAssistedControlPlaneConfigSpec) {
	*out = *in
	if in.APIVIPs != nil {
		in, out := &in.APIVIPs, &out.APIVIPs
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.IngressVIPs != nil {
		in, out := &in.IngressVIPs, &out.IngressVIPs
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.ManifestsConfigMapRefs != nil {
		in, out := &in.ManifestsConfigMapRefs, &out.ManifestsConfigMapRefs
		*out = make([]v1beta1.ManifestsConfigMapReference, len(*in))
		copy(*out, *in)
	}
	if in.DiskEncryption != nil {
		in, out := &in.DiskEncryption, &out.DiskEncryption
		*out = new(v1beta1.DiskEncryption)
		(*in).DeepCopyInto(*out)
	}
	if in.Proxy != nil {
		in, out := &in.Proxy, &out.Proxy
		*out = new(v1beta1.Proxy)
		**out = **in
	}
	if in.PullSecretRef != nil {
		in, out := &in.PullSecretRef, &out.PullSecretRef
		*out = new(corev1.LocalObjectReference)
		**out = **in
	}
	if in.ImageRegistryRef != nil {
		in, out := &in.ImageRegistryRef, &out.ImageRegistryRef
		*out = new(corev1.LocalObjectReference)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OpenshiftAssistedControlPlaneConfigSpec.
func (in *OpenshiftAssistedControlPlaneConfigSpec) DeepCopy() *OpenshiftAssistedControlPlaneConfigSpec {
	if in == nil {
		return nil
	}
	out := new(OpenshiftAssistedControlPlaneConfigSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OpenshiftAssistedControlPlaneList) DeepCopyInto(out *OpenshiftAssistedControlPlaneList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]OpenshiftAssistedControlPlane, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OpenshiftAssistedControlPlaneList.
func (in *OpenshiftAssistedControlPlaneList) DeepCopy() *OpenshiftAssistedControlPlaneList {
	if in == nil {
		return nil
	}
	out := new(OpenshiftAssistedControlPlaneList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *OpenshiftAssistedControlPlaneList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OpenshiftAssistedControlPlaneMachineTemplate) DeepCopyInto(out *OpenshiftAssistedControlPlaneMachineTemplate) {
	*out = *in
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.InfrastructureRef = in.InfrastructureRef
	if in.NodeDrainTimeout != nil {
		in, out := &in.NodeDrainTimeout, &out.NodeDrainTimeout
		*out = new(v1.Duration)
		**out = **in
	}
	if in.NodeVolumeDetachTimeout != nil {
		in, out := &in.NodeVolumeDetachTimeout, &out.NodeVolumeDetachTimeout
		*out = new(v1.Duration)
		**out = **in
	}
	if in.NodeDeletionTimeout != nil {
		in, out := &in.NodeDeletionTimeout, &out.NodeDeletionTimeout
		*out = new(v1.Duration)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OpenshiftAssistedControlPlaneMachineTemplate.
func (in *OpenshiftAssistedControlPlaneMachineTemplate) DeepCopy() *OpenshiftAssistedControlPlaneMachineTemplate {
	if in == nil {
		return nil
	}
	out := new(OpenshiftAssistedControlPlaneMachineTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OpenshiftAssistedControlPlaneSpec) DeepCopyInto(out *OpenshiftAssistedControlPlaneSpec) {
	*out = *in
	in.Config.DeepCopyInto(&out.Config)
	in.MachineTemplate.DeepCopyInto(&out.MachineTemplate)
	in.OpenshiftAssistedConfigSpec.DeepCopyInto(&out.OpenshiftAssistedConfigSpec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OpenshiftAssistedControlPlaneSpec.
func (in *OpenshiftAssistedControlPlaneSpec) DeepCopy() *OpenshiftAssistedControlPlaneSpec {
	if in == nil {
		return nil
	}
	out := new(OpenshiftAssistedControlPlaneSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OpenshiftAssistedControlPlaneStatus) DeepCopyInto(out *OpenshiftAssistedControlPlaneStatus) {
	*out = *in
	if in.ClusterDeploymentRef != nil {
		in, out := &in.ClusterDeploymentRef, &out.ClusterDeploymentRef
		*out = new(corev1.ObjectReference)
		**out = **in
	}
	if in.Version != nil {
		in, out := &in.Version, &out.Version
		*out = new(string)
		**out = **in
	}
	if in.FailureReason != nil {
		in, out := &in.FailureReason, &out.FailureReason
		*out = new(string)
		**out = **in
	}
	if in.FailureMessage != nil {
		in, out := &in.FailureMessage, &out.FailureMessage
		*out = new(string)
		**out = **in
	}
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make(apiv1beta1.Conditions, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OpenshiftAssistedControlPlaneStatus.
func (in *OpenshiftAssistedControlPlaneStatus) DeepCopy() *OpenshiftAssistedControlPlaneStatus {
	if in == nil {
		return nil
	}
	out := new(OpenshiftAssistedControlPlaneStatus)
	in.DeepCopyInto(out)
	return out
}
