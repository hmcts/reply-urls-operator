//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
Copyright 2022.

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
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ReplyURLSync) DeepCopyInto(out *ReplyURLSync) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ReplyURLSync.
func (in *ReplyURLSync) DeepCopy() *ReplyURLSync {
	if in == nil {
		return nil
	}
	out := new(ReplyURLSync)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ReplyURLSync) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ReplyURLSyncList) DeepCopyInto(out *ReplyURLSyncList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ReplyURLSync, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ReplyURLSyncList.
func (in *ReplyURLSyncList) DeepCopy() *ReplyURLSyncList {
	if in == nil {
		return nil
	}
	out := new(ReplyURLSyncList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ReplyURLSyncList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ReplyURLSyncSpec) DeepCopyInto(out *ReplyURLSyncSpec) {
	*out = *in
	if in.TenantID != nil {
		in, out := &in.TenantID, &out.TenantID
		*out = new(string)
		**out = **in
	}
	if in.ClientID != nil {
		in, out := &in.ClientID, &out.ClientID
		*out = new(string)
		**out = **in
	}
	if in.DomainFilter != nil {
		in, out := &in.DomainFilter, &out.DomainFilter
		*out = new(string)
		**out = **in
	}
	if in.IngressClassFilter != nil {
		in, out := &in.IngressClassFilter, &out.IngressClassFilter
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ReplyURLSyncSpec.
func (in *ReplyURLSyncSpec) DeepCopy() *ReplyURLSyncSpec {
	if in == nil {
		return nil
	}
	out := new(ReplyURLSyncSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ReplyURLSyncStatus) DeepCopyInto(out *ReplyURLSyncStatus) {
	*out = *in
	if in.SyncedHosts != nil {
		in, out := &in.SyncedHosts, &out.SyncedHosts
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ReplyURLSyncStatus.
func (in *ReplyURLSyncStatus) DeepCopy() *ReplyURLSyncStatus {
	if in == nil {
		return nil
	}
	out := new(ReplyURLSyncStatus)
	in.DeepCopyInto(out)
	return out
}
