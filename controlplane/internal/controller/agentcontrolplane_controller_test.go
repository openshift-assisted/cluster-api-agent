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

package controller

import (
	"context"

	"sigs.k8s.io/cluster-api/util/conditions"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	hivev1 "github.com/openshift/hive/apis/hive/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	controlplanev1alpha1 "github.com/openshift-assisted/cluster-api-agent/controlplane/api/v1alpha1"
	testutils "github.com/openshift-assisted/cluster-api-agent/test/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

var _ = Describe("AgentControlPlane Controller", func() {
	Context("When reconciling a resource", func() {
		const (
			agentControlPlaneName = "test-resource"
			clusterName           = "test-cluster"
			namespace             = "test"
		)
		var (
			ctx                  = context.Background()
			typeNamespacedName   types.NamespacedName
			cluster              *clusterv1.Cluster
			controllerReconciler *AgentControlPlaneReconciler
			mockCtrl             *gomock.Controller
			k8sClient            client.Client
		)

		BeforeEach(func() {
			k8sClient = fakeclient.NewClientBuilder().
				WithScheme(testScheme).
				WithStatusSubresource(&controlplanev1alpha1.AgentControlPlane{}).Build()

			mockCtrl = gomock.NewController(GinkgoT())
			ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
			Expect(k8sClient.Create(ctx, ns)).To(Succeed())
			typeNamespacedName = types.NamespacedName{
				Name:      agentControlPlaneName,
				Namespace: namespace,
			}

			controllerReconciler = &AgentControlPlaneReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			By("creating a cluster")
			cluster = &clusterv1.Cluster{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: clusterName, Namespace: namespace}, cluster)
			if err != nil && errors.IsNotFound(err) {
				cluster = testutils.NewCluster(clusterName, namespace)
				err := k8sClient.Create(ctx, cluster)
				Expect(err).NotTo(HaveOccurred())
			}
		})

		AfterEach(func() {
			mockCtrl.Finish()
			k8sClient = nil
		})

		When("when a cluster owns this agent control plane", func() {
			It("should successfully create a cluster deployment", func() {
				By("setting the cluster as the owner ref on the agent control plane")

				agentControlPlane := getAgentControlPlane()
				agentControlPlane.SetOwnerReferences(
					[]metav1.OwnerReference{
						*metav1.NewControllerRef(cluster, clusterv1.GroupVersion.WithKind(clusterv1.ClusterKind)),
					},
				)
				Expect(k8sClient.Create(ctx, agentControlPlane)).To(Succeed())

				By("checking if the agent control plane created the cluster deployment after reconcile")
				_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: typeNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())
				err = k8sClient.Get(ctx, typeNamespacedName, agentControlPlane)
				Expect(err).NotTo(HaveOccurred())
				Expect(agentControlPlane.Status.ClusterDeploymentRef).NotTo(BeNil())
				Expect(agentControlPlane.Status.ClusterDeploymentRef.Name).Should(Equal(agentControlPlane.Name))

				By("checking that the cluster deployment was created")
				cd := &hivev1.ClusterDeployment{}
				err = k8sClient.Get(ctx, typeNamespacedName, cd)
				Expect(err).NotTo(HaveOccurred())
				Expect(cd).NotTo(BeNil())

				// assert ClusterDeployment properties
				Expect(cd.Spec.ClusterName).To(Equal(clusterName))
				Expect(cd.Spec.BaseDomain).To(Equal(agentControlPlane.Spec.AgentConfigSpec.BaseDomain))
				Expect(cd.Spec.PullSecretRef).To(Equal(agentControlPlane.Spec.AgentConfigSpec.PullSecretRef))
			})
		})

		When("a cluster owns this agent control plane", func() {
			It("should successfully create a cluster deployment and match AgentControlPlane properties", func() {
				By("setting the cluster as the owner ref on the agent control plane")

				agentControlPlane := getAgentControlPlane()
				agentControlPlane.Spec.AgentConfigSpec.ClusterName = "my-cluster"
				agentControlPlane.Spec.AgentConfigSpec.PullSecretRef = &corev1.LocalObjectReference{
					Name: "my-pullsecret",
				}
				agentControlPlane.Spec.AgentConfigSpec.BaseDomain = "example.com"
				agentControlPlane.SetOwnerReferences(
					[]metav1.OwnerReference{
						*metav1.NewControllerRef(cluster, clusterv1.GroupVersion.WithKind(clusterv1.ClusterKind)),
					},
				)
				Expect(k8sClient.Create(ctx, agentControlPlane)).To(Succeed())

				By("checking if the agent control plane created the cluster deployment after reconcile")
				_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: typeNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())
				err = k8sClient.Get(ctx, typeNamespacedName, agentControlPlane)
				Expect(err).NotTo(HaveOccurred())
				Expect(agentControlPlane.Status.ClusterDeploymentRef).NotTo(BeNil())
				Expect(agentControlPlane.Status.ClusterDeploymentRef.Name).Should(Equal(agentControlPlane.Name))

				By("checking that the cluster deployment was created")
				cd := &hivev1.ClusterDeployment{}
				err = k8sClient.Get(ctx, typeNamespacedName, cd)
				Expect(err).NotTo(HaveOccurred())
				Expect(cd).NotTo(BeNil())

				// assert ClusterDeployment properties
				Expect(cd.Spec.ClusterName).To(Equal(agentControlPlane.Spec.AgentConfigSpec.ClusterName))
				Expect(cd.Spec.BaseDomain).To(Equal(agentControlPlane.Spec.AgentConfigSpec.BaseDomain))
				Expect(cd.Spec.PullSecretRef).To(Equal(agentControlPlane.Spec.AgentConfigSpec.PullSecretRef))
			})
		})

		When("a pull secret isn't set on the AgentControlPlane", func() {
			It("should successfully create the ClusterDeployment with the default fake pull secret", func() {
				By("setting the cluster as the owner ref on the agent control plane")
				agentControlPlane := getAgentControlPlane()
				agentControlPlane.SetOwnerReferences(
					[]metav1.OwnerReference{
						*metav1.NewControllerRef(cluster, clusterv1.GroupVersion.WithKind(clusterv1.ClusterKind)),
					},
				)
				Expect(k8sClient.Create(ctx, agentControlPlane)).To(Succeed())

				By("checking if the agent control plane created the cluster deployment after reconcile")
				_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: typeNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())
				err = k8sClient.Get(ctx, typeNamespacedName, agentControlPlane)
				Expect(err).NotTo(HaveOccurred())
				Expect(agentControlPlane.Status.ClusterDeploymentRef).NotTo(BeNil())
				Expect(agentControlPlane.Status.ClusterDeploymentRef.Name).Should(Equal(agentControlPlane.Name))

				By("checking that the fake pull secret was created")
				pullSecret := &corev1.Secret{}
				err = k8sClient.Get(ctx, types.NamespacedName{Name: placeholderPullSecretName, Namespace: namespace}, pullSecret)
				Expect(err).NotTo(HaveOccurred())
				Expect(pullSecret).NotTo(BeNil())
				Expect(pullSecret.Data).NotTo(BeNil())
				Expect(pullSecret.Data).To(HaveKey(".dockerconfigjson"))

				By("checking that the cluster deployment was created and references the fake pull secret")
				cd := &hivev1.ClusterDeployment{}
				err = k8sClient.Get(ctx, typeNamespacedName, cd)
				Expect(err).NotTo(HaveOccurred())
				Expect(cd).NotTo(BeNil())
				Expect(cd.Spec.PullSecretRef).NotTo(BeNil())
				Expect(cd.Spec.PullSecretRef.Name).To(Equal(placeholderPullSecretName))

			})
		})
		It("should add a finalizer to the agent control plane if it's not being deleted", func() {
			By("setting the owner ref on the agent control plane")

			agentControlPlane := getAgentControlPlane()
			agentControlPlane.SetOwnerReferences(
				[]metav1.OwnerReference{
					*metav1.NewControllerRef(cluster, clusterv1.GroupVersion.WithKind(clusterv1.ClusterKind)),
				},
			)
			Expect(k8sClient.Create(ctx, agentControlPlane)).To(Succeed())

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred())

			By("checking if the agent control plane has the finalizer")
			err = k8sClient.Get(ctx, typeNamespacedName, agentControlPlane)
			Expect(err).NotTo(HaveOccurred())
			Expect(agentControlPlane.Finalizers).NotTo(BeEmpty())
			Expect(agentControlPlane.Finalizers).To(ContainElement(acpFinalizer))
		})
		When("an invalid version is set on the AgentControlPlane", func() {
			It("should return error", func() {
				By("setting the cluster as the owner ref on the agent control plane")
				agentControlPlane := getAgentControlPlane()
				agentControlPlane.Spec.Version = "4.12.0"
				agentControlPlane.SetOwnerReferences(
					[]metav1.OwnerReference{
						*metav1.NewControllerRef(cluster, clusterv1.GroupVersion.WithKind(clusterv1.ClusterKind)),
					},
				)
				Expect(k8sClient.Create(ctx, agentControlPlane)).To(Succeed())
				By("checking if the condition of invalid version")
				_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: typeNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())

				Expect(k8sClient.Get(ctx, typeNamespacedName, agentControlPlane)).To(Succeed())

				condition := conditions.Get(agentControlPlane,
					controlplanev1alpha1.MachinesCreatedCondition,
				)
				Expect(condition).NotTo(BeNil())
				Expect(condition.Message).To(Equal("version 4.12.0 is not supported, the minimum supported version is 4.14.0"))

			})
		})
	})
})

func getAgentControlPlane() *controlplanev1alpha1.AgentControlPlane {
	return &controlplanev1alpha1.AgentControlPlane{
		ObjectMeta: metav1.ObjectMeta{
			Name:      agentControlPlaneName,
			Namespace: namespace,
		},
		Spec: controlplanev1alpha1.AgentControlPlaneSpec{
			Version: "4.16.0",
		},
	}
}
