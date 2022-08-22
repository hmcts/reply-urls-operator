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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// RedirectUriSyncSpec defines the desired state of RedirectUriSync
type RedirectUriSyncSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	TenantID           *string `json:"tenantID"`
	ClientID           *string `json:"clientID"`
	DomainFilter       *string `json:"domainFilter,omitempty"`
	IngressClassFilter *string `json:"ingressClassFilter,omitempty"`
}

// RedirectUriSyncStatus defines the observed state of RedirectUriSync
type RedirectUriSyncStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	SyncedHosts []string `json:"syncedHosts"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// RedirectUriSync is the Schema for the redirecturisyncs API
type RedirectUriSync struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RedirectUriSyncSpec   `json:"spec,omitempty"`
	Status RedirectUriSyncStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RedirectUriSyncList contains a list of RedirectUriSync
type RedirectUriSyncList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RedirectUriSync `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RedirectUriSync{}, &RedirectUriSyncList{})
}
