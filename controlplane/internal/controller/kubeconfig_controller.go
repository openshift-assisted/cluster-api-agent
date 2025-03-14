package controller

import (
	"context"
	"errors"
	"fmt"
	"github.com/openshift-assisted/cluster-api-agent/controlplane/internal/workloadclient"
	kubecfg "github.com/openshift-assisted/cluster-api-agent/pkg/kubeconfig"
	logutil "github.com/openshift-assisted/cluster-api-agent/util/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"time"
)

const certRenewDaysFromNotAfterDate = 30

// KubeconfigReconciler reconciles Secret objects
type KubeconfigReconciler struct {
	client.Client
	ClientGenerator workloadclient.ClientGenerator
	Scheme          *runtime.Scheme
}

//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile handles Secret resources
func (r *KubeconfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Get the Secret
	secret := &corev1.Secret{}
	if err := r.Get(ctx, req.NamespacedName, secret); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check if the Secret has the correct ControllerRef
	if owner := metav1.GetControllerOf(secret); owner == nil || owner.Kind != "OpenshiftAssistedControlPlane" {
		// Not owned by OpenshiftAssistedControlPlane, ignore it
		return ctrl.Result{}, nil
	}
	logger = logger.WithValues("name", secret.Name, "namespace", secret.Name)

	kubeconfig, ok := secret.Data["value"]
	if !ok {
		logger.V(logutil.InfoLevel).Info("ignoring secret with unexpected format: no `value` key found")
		return ctrl.Result{}, nil
	}

	config, err := clientcmd.Load(kubeconfig)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error parsing kubeconfig: %v", err)
	}
	username := config.Contexts[config.CurrentContext].AuthInfo

	notBefore, notAfter, err := kubecfg.GetClientCertDates(kubeconfig, username)
	if err != nil {
		return ctrl.Result{}, err
	}
	logger = logger.WithValues("notAfter", notAfter, "notBefore", notBefore, "username", username)

	if time.Now().After(notAfter) {
		return ctrl.Result{}, fmt.Errorf("kubeconfig client-certificate expired. Please update it manually with a valid client-certificate")
	}

	renewDate := notAfter.AddDate(0, 0, -certRenewDaysFromNotAfterDate)
	// renew from 1 week before notAfter date
	if time.Now().Before(renewDate) {
		logger.Info("renew date still ahead in the future, requeuing")
		return ctrl.Result{Requeue: true, RequeueAfter: renewDate.Sub(time.Now())}, nil
	}

	if time.Now().Before(notBefore) {
		logger.Info("trying to handle certificate before notBefore date, requeuing")
		return ctrl.Result{
			Requeue:      true,
			RequeueAfter: notBefore.Sub(time.Now()),
		}, nil
	}
	// renew cert
	k8sClient, err := r.ClientGenerator.GetWorkloadClusterClient(kubeconfig)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error initializing workload cluster k8s client: %v", err)
	}
	newKubeconfig, err := kubecfg.RegenerateClientCertWithCSR(ctx, k8sClient, kubeconfig, username)
	if err != nil {
		if errors.Is(err, kubecfg.ErrCSRProcessing) {
			// CSR is still processing: we need to reconcile again
			return ctrl.Result{
				Requeue:      true,
				RequeueAfter: time.Second * 10,
			}, nil
		}
		return ctrl.Result{}, fmt.Errorf("error renewing client-certificate: %v", err)
	}
	updatedNotBefore, _, err := kubecfg.GetClientCertDates(kubeconfig, username)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error extracting dates from updated client-certificate: %v", err)
	}
	secret.Data[kubeconfigSecretKey] = newKubeconfig
	if err := r.Client.Update(ctx, secret); err != nil {
		return ctrl.Result{}, fmt.Errorf("error updating kubeconfig with updated client-certificate: %v", err)
	}
	// Re-queue request based on notBefore time
	return ctrl.Result{
		Requeue:      true,
		RequeueAfter: updatedNotBefore.Sub(time.Now()),
	}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KubeconfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Secret{}).
		WithEventFilter(predicate.Funcs{
			DeleteFunc: func(e event.DeleteEvent) bool {
				// Ignore all deletion events
				return false
			},
			CreateFunc: func(e event.CreateEvent) bool {
				return isKubeconfigToBeWatched(e.Object)
			},
			UpdateFunc: func(e event.UpdateEvent) bool {
				return isKubeconfigToBeWatched(e.ObjectNew)
			},
		}).
		Complete(r)
}

func isKubeconfigToBeWatched(obj client.Object) bool {
	labels := obj.GetLabels()
	if value, ok := labels[clusterv1.WatchLabel]; ok {
		return value == ctrlPlaneKubeconfig
	}
	return false
}
