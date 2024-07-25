package utils

import (
	metal3 "github.com/metal3-io/cluster-api-provider-metal3/api/v1beta1"
	bootstrapv1alpha1 "github.com/openshift-assisted/cluster-api-agent/bootstrap/api/v1alpha1"
	controlplanev1alpha1 "github.com/openshift-assisted/cluster-api-agent/controlplane/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

func ClusterCR() *clusterv1.Cluster {
	return &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
		Spec: clusterv1.ClusterSpec{
			ClusterNetwork: &clusterv1.ClusterNetwork{
				Pods: &clusterv1.NetworkRanges{
					CIDRBlocks: []string{"192.168.0.0/22"},
				},
				Services: &clusterv1.NetworkRanges{
					CIDRBlocks: []string{"10.96.0.0/12"},
				},
			},
			ControlPlaneRef: &corev1.ObjectReference{
				APIVersion: controlplanev1alpha1.GroupVersion.String(),
				Kind:       "AgentControlPlane",
				Name:       "test",
				Namespace:  "test",
			},
		},
	}
}

func Metal3ClusterCR() *metal3.Metal3Cluster {
	cluster := &metal3.Metal3Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sno",
			Namespace: "test",
		},
	}
	return cluster
}

func AgentControlPlaneCR() *controlplanev1alpha1.AgentControlPlane {
	cp := &controlplanev1alpha1.AgentControlPlane{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
	}
	return cp
}

func AgentBootstrapConfigCR() *bootstrapv1alpha1.AgentBootstrapConfig {
	abc := &bootstrapv1alpha1.AgentBootstrapConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
	}
	return abc
}
