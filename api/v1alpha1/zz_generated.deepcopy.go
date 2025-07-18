//go:build !ignore_autogenerated

/*
Copyright 2025.

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
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachineSpec) DeepCopyInto(out *MachineSpec) {
	*out = *in
	if in.InstallDisk != nil {
		in, out := &in.InstallDisk, &out.InstallDisk
		*out = new(string)
		**out = **in
	}
	if in.Image != nil {
		in, out := &in.Image, &out.Image
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachineSpec.
func (in *MachineSpec) DeepCopy() *MachineSpec {
	if in == nil {
		return nil
	}
	out := new(MachineSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MetalSpec) DeepCopyInto(out *MetalSpec) {
	*out = *in
	if in.Machines != nil {
		in, out := &in.Machines, &out.Machines
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.MachineSpec != nil {
		in, out := &in.MachineSpec, &out.MachineSpec
		*out = new(MachineSpec)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MetalSpec.
func (in *MetalSpec) DeepCopy() *MetalSpec {
	if in == nil {
		return nil
	}
	out := new(MetalSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TalosCluster) DeepCopyInto(out *TalosCluster) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TalosCluster.
func (in *TalosCluster) DeepCopy() *TalosCluster {
	if in == nil {
		return nil
	}
	out := new(TalosCluster)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TalosCluster) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TalosClusterList) DeepCopyInto(out *TalosClusterList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]TalosCluster, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TalosClusterList.
func (in *TalosClusterList) DeepCopy() *TalosClusterList {
	if in == nil {
		return nil
	}
	out := new(TalosClusterList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TalosClusterList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TalosClusterSpec) DeepCopyInto(out *TalosClusterSpec) {
	*out = *in
	if in.ControlPlane != nil {
		in, out := &in.ControlPlane, &out.ControlPlane
		*out = new(TalosControlPlaneSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.ControlPlaneRef != nil {
		in, out := &in.ControlPlaneRef, &out.ControlPlaneRef
		*out = new(v1.LocalObjectReference)
		**out = **in
	}
	if in.Worker != nil {
		in, out := &in.Worker, &out.Worker
		*out = new(TalosWorkerSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.WorkerRef != nil {
		in, out := &in.WorkerRef, &out.WorkerRef
		*out = new(v1.LocalObjectReference)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TalosClusterSpec.
func (in *TalosClusterSpec) DeepCopy() *TalosClusterSpec {
	if in == nil {
		return nil
	}
	out := new(TalosClusterSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TalosClusterStatus) DeepCopyInto(out *TalosClusterStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]metav1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TalosClusterStatus.
func (in *TalosClusterStatus) DeepCopy() *TalosClusterStatus {
	if in == nil {
		return nil
	}
	out := new(TalosClusterStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TalosControlPlane) DeepCopyInto(out *TalosControlPlane) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TalosControlPlane.
func (in *TalosControlPlane) DeepCopy() *TalosControlPlane {
	if in == nil {
		return nil
	}
	out := new(TalosControlPlane)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TalosControlPlane) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TalosControlPlaneList) DeepCopyInto(out *TalosControlPlaneList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]TalosControlPlane, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TalosControlPlaneList.
func (in *TalosControlPlaneList) DeepCopy() *TalosControlPlaneList {
	if in == nil {
		return nil
	}
	out := new(TalosControlPlaneList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TalosControlPlaneList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TalosControlPlaneSpec) DeepCopyInto(out *TalosControlPlaneSpec) {
	*out = *in
	in.MetalSpec.DeepCopyInto(&out.MetalSpec)
	if in.StorageClassName != nil {
		in, out := &in.StorageClassName, &out.StorageClassName
		*out = new(string)
		**out = **in
	}
	if in.PodCIDR != nil {
		in, out := &in.PodCIDR, &out.PodCIDR
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.ServiceCIDR != nil {
		in, out := &in.ServiceCIDR, &out.ServiceCIDR
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.ConfigRef != nil {
		in, out := &in.ConfigRef, &out.ConfigRef
		*out = new(v1.ConfigMapKeySelector)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TalosControlPlaneSpec.
func (in *TalosControlPlaneSpec) DeepCopy() *TalosControlPlaneSpec {
	if in == nil {
		return nil
	}
	out := new(TalosControlPlaneSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TalosControlPlaneStatus) DeepCopyInto(out *TalosControlPlaneStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]metav1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TalosControlPlaneStatus.
func (in *TalosControlPlaneStatus) DeepCopy() *TalosControlPlaneStatus {
	if in == nil {
		return nil
	}
	out := new(TalosControlPlaneStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TalosMachine) DeepCopyInto(out *TalosMachine) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TalosMachine.
func (in *TalosMachine) DeepCopy() *TalosMachine {
	if in == nil {
		return nil
	}
	out := new(TalosMachine)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TalosMachine) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TalosMachineList) DeepCopyInto(out *TalosMachineList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]TalosMachine, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TalosMachineList.
func (in *TalosMachineList) DeepCopy() *TalosMachineList {
	if in == nil {
		return nil
	}
	out := new(TalosMachineList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TalosMachineList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TalosMachineSpec) DeepCopyInto(out *TalosMachineSpec) {
	*out = *in
	if in.MachineSpec != nil {
		in, out := &in.MachineSpec, &out.MachineSpec
		*out = new(MachineSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.ControlPlaneRef != nil {
		in, out := &in.ControlPlaneRef, &out.ControlPlaneRef
		*out = new(v1.ObjectReference)
		**out = **in
	}
	if in.WorkerRef != nil {
		in, out := &in.WorkerRef, &out.WorkerRef
		*out = new(v1.ObjectReference)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TalosMachineSpec.
func (in *TalosMachineSpec) DeepCopy() *TalosMachineSpec {
	if in == nil {
		return nil
	}
	out := new(TalosMachineSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TalosMachineStatus) DeepCopyInto(out *TalosMachineStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]metav1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TalosMachineStatus.
func (in *TalosMachineStatus) DeepCopy() *TalosMachineStatus {
	if in == nil {
		return nil
	}
	out := new(TalosMachineStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TalosWorker) DeepCopyInto(out *TalosWorker) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TalosWorker.
func (in *TalosWorker) DeepCopy() *TalosWorker {
	if in == nil {
		return nil
	}
	out := new(TalosWorker)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TalosWorker) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TalosWorkerList) DeepCopyInto(out *TalosWorkerList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]TalosWorker, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TalosWorkerList.
func (in *TalosWorkerList) DeepCopy() *TalosWorkerList {
	if in == nil {
		return nil
	}
	out := new(TalosWorkerList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TalosWorkerList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TalosWorkerSpec) DeepCopyInto(out *TalosWorkerSpec) {
	*out = *in
	in.MetalSpec.DeepCopyInto(&out.MetalSpec)
	if in.StorageClassName != nil {
		in, out := &in.StorageClassName, &out.StorageClassName
		*out = new(string)
		**out = **in
	}
	out.ControlPlaneRef = in.ControlPlaneRef
	if in.ConfigRef != nil {
		in, out := &in.ConfigRef, &out.ConfigRef
		*out = new(v1.ConfigMapKeySelector)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TalosWorkerSpec.
func (in *TalosWorkerSpec) DeepCopy() *TalosWorkerSpec {
	if in == nil {
		return nil
	}
	out := new(TalosWorkerSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TalosWorkerStatus) DeepCopyInto(out *TalosWorkerStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]metav1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TalosWorkerStatus.
func (in *TalosWorkerStatus) DeepCopy() *TalosWorkerStatus {
	if in == nil {
		return nil
	}
	out := new(TalosWorkerStatus)
	in.DeepCopyInto(out)
	return out
}
