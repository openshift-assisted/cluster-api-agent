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

package v1alpha1

import (
	bootstrapv1beta1 "github.com/openshift-assisted/cluster-api-agent/bootstrap/api/v1alpha1"
	hiveext "github.com/openshift/assisted-service/api/hiveextension/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type OpenshiftAssistedControlPlaneMachineTemplate struct {
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	ObjectMeta clusterv1.ObjectMeta `json:"metadata,omitempty"`

	// InfrastructureRef is a required reference to a custom resource
	// offered by an infrastructure provider.
	InfrastructureRef corev1.ObjectReference `json:"infrastructureRef"`

	// NodeDrainTimeout is the total amount of time that the controller will spend on draining a controlplane node
	// The default value is 0, meaning that the node can be drained without any time limitations.
	// NOTE: NodeDrainTimeout is different from `kubectl drain --timeout`
	// +optional
	NodeDrainTimeout *metav1.Duration `json:"nodeDrainTimeout,omitempty"`

	// NodeVolumeDetachTimeout is the total amount of time that the controller will spend on waiting for all volumes
	// to be detached. The default value is 0, meaning that the volumes can be detached without any time limitations.
	// +optional
	NodeVolumeDetachTimeout *metav1.Duration `json:"nodeVolumeDetachTimeout,omitempty"`

	// NodeDeletionTimeout defines how long the machine controller will attempt to delete the Node that the Machine
	// hosts after the Machine is marked for deletion. A duration of 0 will retry deletion indefinitely.
	// If no value is provided, the default value for this property of the Machine resource will be used.
	// +optional
	NodeDeletionTimeout *metav1.Duration `json:"nodeDeletionTimeout,omitempty"`
}

// OpenshiftAssistedControlPlaneSpec defines the desired state of OpenshiftAssistedControlPlane
type OpenshiftAssistedControlPlaneSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Config specs for the OpenshiftAssistedControlPlane
	Config                      OpenshiftAssistedControlPlaneConfigSpec      `json:"config,omitempty"`
	MachineTemplate             OpenshiftAssistedControlPlaneMachineTemplate `json:"machineTemplate"`
	OpenshiftAssistedConfigSpec bootstrapv1beta1.OpenshiftAssistedConfigSpec `json:"openshiftAssistedConfigSpec,omitempty"`
	Replicas                    int32                                        `json:"replicas,omitempty"`
	Version                     string                                       `json:"version"`
}

// OpenshiftAssistedControlPlaneConfigSpec defines configuration for the agent-provisioned cluster
type OpenshiftAssistedControlPlaneConfigSpec struct {
	// From AgentClusterInstall https://github.com/openshift/assisted-service/blob/master/api/hiveextension/v1beta1/agentclusterinstall_types.go

	// APIVIPs are the virtual IPs used to reach the OpenShift cluster's API.
	// Enter one IP address for single-stack clusters, or up to two for dual-stack clusters (at
	// most one IP address per IP stack used). The order of stacks should be the same as order
	// of subnets in Cluster Networks, Service Networks, and Machine Networks.
	// +kubebuilder:validation:MaxItems=2
	// +optional
	APIVIPs []string `json:"apiVIPs,omitempty"`

	// IngressVIPs are the virtual IPs used for cluster ingress traffic.
	// Enter one IP address for single-stack clusters, or up to two for dual-stack clusters (at
	// most one IP address per IP stack used). The order of stacks should be the same as order
	// of subnets in Cluster Networks, Service Networks, and Machine Networks.
	// +kubebuilder:validation:MaxItems=2
	// +optional
	IngressVIPs []string `json:"ingressVIPs,omitempty"`

	// ManifestsConfigMapRefs is an array of references to user-provided manifests ConfigMaps to
	// add to or replace manifests that are generated by the installer.
	// Manifest names in each ConfigMap should be unique across all referenced ConfigMaps.
	// +optional
	ManifestsConfigMapRefs []hiveext.ManifestsConfigMapReference `json:"manifestsConfigMapRefs,omitempty"`

	// DiskEncryption is the configuration to enable/disable disk encryption for cluster nodes.
	// +optional
	DiskEncryption *hiveext.DiskEncryption `json:"diskEncryption,omitempty"`

	// Proxy defines the proxy settings used for the install config
	// +optional
	Proxy *hiveext.Proxy `json:"proxy,omitempty"`

	// Set to true to allow control plane nodes to be schedulable
	// +optional
	MastersSchedulable bool `json:"mastersSchedulable,omitempty"`

	// SSHAuthorizedKey ssh key for accessing the cluster nodes after reboot
	SSHAuthorizedKey string `json:"sshAuthorizedKey,omitempty"`

	// From ClusterDeployment

	// ClusterName is the friendly name of the cluster. It is used for subdomains,
	// some resource tagging, and other instances where a friendly name for the
	// cluster is useful.
	// If not defined ClusterName will be set as the CAPI ClusterName
	// +optional
	ClusterName string `json:"clusterName"`

	// BaseDomain is the base domain to which the cluster should belong.
	// +required
	BaseDomain string `json:"baseDomain"`

	// PullSecretRef references pull secret necessary for the cluster installation
	PullSecretRef *corev1.LocalObjectReference `json:"pullSecretRef,omitempty"`

	ReleaseImage string `json:"releaseImage"`

	// ImageRegistryRef is a reference to a configmap containing both the additional
	// image registries and their corresponding certificate bundles to be used in the spoke cluster
	ImageRegistryRef *corev1.LocalObjectReference `json:"imageRegistryRef,omitempty"`
}

// OpenshiftAssistedControlPlaneStatus defines the observed state of OpenshiftAssistedControlPlane
type OpenshiftAssistedControlPlaneStatus struct {
	// ClusterDeploymentRef references the ClusterDeployment used to create the cluster
	ClusterDeploymentRef *corev1.ObjectReference `json:"clusterDeploymentRef,omitempty"`

	// Selector is the label selector in string format to avoid introspection
	// by clients, and is used to provide the CRD-based integration for the
	// scale subresource and additional integrations for things like kubectl
	// describe.. The string will be in the same format as the query-param syntax.
	// More info about label selectors: http://kubernetes.io/docs/user-guide/labels#label-selectors
	// +optional
	Selector string `json:"selector,omitempty"`

	// Total number of non-terminated machines targeted by this control plane
	// (their labels match the selector).
	// +optional
	Replicas int32 `json:"replicas"`

	// Version represents the minimum Kubernetes version for the control plane machines
	// in the cluster.
	// +optional
	Version *string `json:"version,omitempty"`

	// Total number of non-terminated machines targeted by this control plane
	// that have the desired template spec.
	// +optional
	UpdatedReplicas int32 `json:"updatedReplicas"`

	// Total number of fully running and ready control plane machines.
	// +optional
	ReadyReplicas int32 `json:"readyReplicas"`

	// Total number of unavailable machines targeted by this control plane.
	// This is the total number of machines that are still required for
	// the deployment to have 100% available capacity. They may either
	// be machines that are running but not yet ready or machines
	// that still have not been created.
	// +optional
	UnavailableReplicas int32 `json:"unavailableReplicas"`

	// Initialized denotes whether or not the control plane has the
	// uploaded kubeadm-config configmap.
	// +optional
	Initialized bool `json:"initialized"`

	// Ready denotes that the OpenshiftAssistedControlPlane API Server became ready during initial provisioning
	// to receive requests.
	// NOTE: this field is part of the Cluster API contract and it is used to orchestrate provisioning.
	// The value of this field is never updated after provisioning is completed. Please use conditions
	// to check the operational state of the control plane.
	// +optional
	Ready bool `json:"ready"`

	// FailureReason indicates that there is a terminal problem reconciling the
	// state, and will be set to a token value suitable for
	// programmatic interpretation.
	// +optional
	FailureReason *string `json:"failureReason,omitempty"`

	// ErrorMessage indicates that there is a terminal problem reconciling the
	// state, and will be set to a descriptive error message.
	// +optional
	FailureMessage *string `json:"failureMessage,omitempty"`

	// Conditions defines current service state of the KubeadmControlPlane.
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resources:shortName=aocp
// +kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas,selectorpath=.status.selector
// +kubebuilder:printcolumn:name="Cluster",type="string",JSONPath=".metadata.labels['cluster\\.x-k8s\\.io/cluster-name']",description="Cluster"
// +kubebuilder:printcolumn:name="Initialized",type=boolean,JSONPath=".status.initialized",description="This denotes whether or not the control plane has the uploaded kubeadm-config configmap"
// +kubebuilder:printcolumn:name="API Server Available",type=boolean,JSONPath=".status.ready",description="KubeadmControlPlane API Server is ready to receive requests"
// +kubebuilder:printcolumn:name="Desired",type=integer,JSONPath=".spec.replicas",description="Total number of machines desired by this control plane",priority=10
// +kubebuilder:printcolumn:name="Replicas",type=integer,JSONPath=".status.replicas",description="Total number of non-terminated machines targeted by this control plane"
// +kubebuilder:printcolumn:name="Ready",type=integer,JSONPath=".status.readyReplicas",description="Total number of fully running and ready control plane machines"
// +kubebuilder:printcolumn:name="Updated",type=integer,JSONPath=".status.updatedReplicas",description="Total number of non-terminated machines targeted by this control plane that have the desired template spec"
// +kubebuilder:printcolumn:name="Unavailable",type=integer,JSONPath=".status.unavailableReplicas",description="Total number of unavailable machines targeted by this control plane"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Time duration since creation of KubeadmControlPlane"
// +kubebuilder:printcolumn:name="Version",type=string,JSONPath=".spec.version",description="OpenShift version associated with this control plane"

// OpenshiftAssistedControlPlane is the Schema for the openshiftassistedcontrolplane API
type OpenshiftAssistedControlPlane struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenshiftAssistedControlPlaneSpec   `json:"spec,omitempty"`
	Status OpenshiftAssistedControlPlaneStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OpenshiftAssistedControlPlaneList contains a list of OpenshiftAssistedControlPlane
type OpenshiftAssistedControlPlaneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OpenshiftAssistedControlPlane `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OpenshiftAssistedControlPlane{}, &OpenshiftAssistedControlPlaneList{})
}

// GetConditions returns the set of conditions for this object.
func (in *OpenshiftAssistedControlPlane) GetConditions() clusterv1.Conditions {
	return in.Status.Conditions
}

// SetConditions sets the conditions on this object.
func (in *OpenshiftAssistedControlPlane) SetConditions(conditions clusterv1.Conditions) {
	in.Status.Conditions = conditions
}
