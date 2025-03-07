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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DNSNameSpec defines the desired state of DNSName
type DNSNameSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Type is the type of the DNSName
	// +kubebuilder:validation:Enum=CNAME;A
	Type DNSRecordType `json:"type"`

	// Domain is the source domain of the DNSName
	// +kubebuilder:validation:Format=hostname
	Domain string `json:"domain"`

	// Target is the target of the DNSName
	// +kubebuilder:validation:Format=hostname
	Target string `json:"target,omitempty"`

	// IP is the IPv4 or IPv6 of the type A DNSName (only applies to A records)
	TargetIP *IPAddressStr `json:"targetIP,omitempty"`

	// TTL is the TTL of the DNSName (only applies to CNAME records)
	// +kubebuilder:validation:Minimum=0
	TTL *int32 `json:"ttl,omitempty"`
}

// DNSNameStatus defines the observed state of DNSName
type DNSNameStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// DNSName is the Schema for the dnsnames API
type DNSName struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DNSNameSpec   `json:"spec,omitempty"`
	Status DNSNameStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DNSNameList contains a list of DNSName
type DNSNameList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DNSName `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DNSName{}, &DNSNameList{})
}
