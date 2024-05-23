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

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AgentBootstrapConfigSpec defines the desired state of AgentBootstrapConfig
type AgentBootstrapConfigSpec struct {
	// Here we can add details to configure infraenv
	// InfraEnvRef references the infra env to generate the ISO
	InfraEnvRef      *corev1.ObjectReference      `json:"infraEnvRef,omitempty"`
	PullSecretRef    *corev1.LocalObjectReference `json:"pullSecretRef,omitempty"`
	SSHAuthorizedKey string                       `json:"sshAuthorizedKey,omitempty"`
}

// AgentBootstrapConfigStatus defines the observed state of AgentBootstrapConfig
type AgentBootstrapConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	ISODownloadURL string `json:"isoDownloadURL,omitempty"`

	// AgentRef is the agent that this AgentBootstrapConfig is attached to
	AgentRef *corev1.LocalObjectReference `json:"agentRef,omitempty"`

	// Ready indicates the BootstrapData field is ready to be consumed
	// +optional
	Ready bool `json:"ready"`

	// DataSecretName is the name of the secret that stores the bootstrap data script.
	// +optional
	DataSecretName *string `json:"dataSecretName,omitempty"`

	// FailureReason will be set on non-retryable errors
	// +optional
	FailureReason string `json:"failureReason,omitempty"`

	// FailureMessage will be set on non-retryable errors
	// +optional
	FailureMessage string `json:"failureMessage,omitempty"`

	// ObservedGeneration is the latest generation observed by the controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions defines current service state of the KubeadmConfig.
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// AgentBootstrapConfig is the Schema for the agentbootstrapconfigs API
type AgentBootstrapConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AgentBootstrapConfigSpec   `json:"spec,omitempty"`
	Status AgentBootstrapConfigStatus `json:"status,omitempty"`
}

// GetConditions returns the set of conditions for this object.
func (c *AgentBootstrapConfig) GetConditions() clusterv1.Conditions {
	return c.Status.Conditions
}

// SetConditions sets the conditions on this object.
func (c *AgentBootstrapConfig) SetConditions(conditions clusterv1.Conditions) {
	c.Status.Conditions = conditions
}

//+kubebuilder:object:root=true

// AgentBootstrapConfigList contains a list of AgentBootstrapConfig
type AgentBootstrapConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AgentBootstrapConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AgentBootstrapConfig{}, &AgentBootstrapConfigList{})
}
