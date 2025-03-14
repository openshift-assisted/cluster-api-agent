package controller_test

import (
	"context"
	"github.com/golang/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-assisted/cluster-api-agent/controlplane/internal/controller"
	"github.com/openshift-assisted/cluster-api-agent/controlplane/internal/workloadclient"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("KubeconfigReconciler", func() {
	var (
		fakeClient          client.Client
		scheme              *runtime.Scheme
		secret              *corev1.Secret
		kubeconfigData      []byte
		ctrl                *gomock.Controller
		ctx                 context.Context
		req                 reconcile.Request
		reconciler          *controller.KubeconfigReconciler
		mockClientGenerator *workloadclient.MockClientGenerator
	)

	BeforeEach(func() {
		_ = corev1.AddToScheme(scheme)
		ctrl = gomock.NewController(GinkgoT())

		mockClientGenerator = workloadclient.NewMockClientGenerator(ctrl)
		mockClientGenerator.EXPECT().GetWorkloadClusterClient(gomock.Any())
		fakeClient = fake.NewClientBuilder().WithScheme(scheme).Build()
		reconciler = &controller.KubeconfigReconciler{
			Client:          fakeClient,
			ClientGenerator: mockClientGenerator,
			Scheme:          scheme,
		}

		kubeconfigData = []byte(`apiVersion: v1
clusters:
- cluster:
    server: https://127.0.0.1:6443
  name: local
contexts:
- context:
    cluster: local
    user: admin
  name: local
current-context: local
kind: Config
preferences: {}
users:
- name: admin
  user:
    client-certificate-data: <cert>
    client-key-data: <key>`)

		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-secret",
				Namespace: "default",
				Labels: map[string]string{
					"watch-label": "ctrl-plane-kubeconfig",
				},
			},
			Data: map[string][]byte{
				"value": kubeconfigData,
			},
		}

		ctx = context.TODO()
		req = reconcile.Request{
			NamespacedName: client.ObjectKey{
				Name:      "test-secret",
				Namespace: "default",
			},
		}
	})

	JustBeforeEach(func() {
		Expect(fakeClient.Create(ctx, secret)).To(Succeed())
	})

	Context("when the Secret is valid", func() {
		It("should reconcile successfully", func() {
			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Requeue).To(BeTrue())
		})
	})

	Context("when the Secret is missing the value key", func() {
		BeforeEach(func() {
			delete(secret.Data, "value")
		})

		It("should ignore the Secret", func() {
			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())
		})
	})

	Context("when the kubeconfig is invalid", func() {
		BeforeEach(func() {
			secret.Data["value"] = []byte("invalid kubeconfig")
		})

		It("should return an error", func() {
			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("when the client certificate is expired", func() {
		BeforeEach(func() {
			expiredKubeconfig := kubeconfigData
			expiredKubeconfig = []byte(`apiVersion: v1
clusters:
- cluster:
    server: https://127.0.0.1:6443
  name: local
contexts:
- context:
    cluster: local
    user: admin
  name: local
current-context: local
kind: Config
preferences: {}
users:
- name: admin
  user:
    client-certificate-data: <expired-cert>
    client-key-data: <key>`)

			secret.Data["value"] = expiredKubeconfig
		})

		It("should return an error", func() {
			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).To(HaveOccurred())
		})
	})
})
