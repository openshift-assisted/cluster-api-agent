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
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	//"sigs.k8s.io/controller-runtime/pkg/log"
	metal3 "github.com/metal3-io/cluster-api-provider-metal3/api/v1beta1"
	bootstrapv1beta1 "github.com/openshift-assisted/cluster-api-agent/bootstrap/api/v1beta1"
	aiv1beta1 "github.com/openshift/assisted-service/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/controller-runtime/pkg/handler"
)

//TODO: Use caches

// AgentBootstrapConfigReconciler reconciles a AgentBootstrapConfigSpec object
type AgentBootstrapConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const (
	agentBootstrapConfigLabel = "bootstrap.cluster.x-k8s.io/agentBootstrapConfig"
)

func (r *AgentBootstrapConfigReconciler) getMachineTemplate(ctx context.Context, machineDeployment clusterv1.MachineDeployment) *metal3.Metal3MachineTemplate {
	log := ctrl.LoggerFrom(ctx)
	machineTemplateRef := machineDeployment.Spec.Template.Spec.InfrastructureRef
	log.Info("machine template ref", "groupversionkind", machineTemplateRef.GroupVersionKind())
	if machineTemplateRef.GroupVersionKind() == metal3.GroupVersion.WithKind("Metal3MachineTemplate") {
		nsName := types.NamespacedName{
			Name:      machineTemplateRef.Name,
			Namespace: machineDeployment.Namespace,
		}
		machineTemplate := &metal3.Metal3MachineTemplate{}
		if err := r.Client.Get(ctx, nsName, machineTemplate); err != nil {
			if apierrors.IsNotFound(err) {
				log.Info("machine template not found", "namespacedname", nsName)
				return nil
			}
			return nil
		}
		return machineTemplate
	}
	return nil
}

//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machines;machines/status,verbs=get;list;watch;create;update;patch;delete
//metal3machinetemplates" in API group "infrastructure.cluster.x-k8s.io"
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=metal3machines;metal3machines/status,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=bootstrap.cluster.x-k8s.io,resources=agentbootstrapconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=bootstrap.cluster.x-k8s.io,resources=agentbootstrapconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=bootstrap.cluster.x-k8s.io,resources=agentbootstrapconfigs/finalizers,verbs=update
// +kubebuilder:rbac:groups=agent-install.openshift.io,resources=infraenvs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=agent-install.openshift.io,resources=infraenvs/status,verbs=get;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the AgentBootstrapConfigSpec object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.2/pkg/reconcile
func (r *AgentBootstrapConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	config := &bootstrapv1beta1.AgentBootstrapConfig{}
	log.Info("Getting AgentBootstrapConfig", "namespacedname", req.NamespacedName)
	if err := r.Client.Get(ctx, req.NamespacedName, config); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Get infraenv name given the agentbootstrap config
	// 1 infra env per machine deployment
	// 1 infra env per control plane cluster
	infraEnvName, err := r.GetInfraEnvName(config)
	if err != nil {
		log.Error(err, "couldn't get infraenv name for agentbootstrapconfig", "name", config.Name)
		return ctrl.Result{}, err
	}
	log.Info("computed infraEnvName", "name", infraEnvName)

	if infraEnvName == "" {
		log.Info("no infraenv name for agentbootstrapconfig", "name", config.Name)
		return ctrl.Result{}, nil
	}

	// Query for InfraEnv based on name/namespace and set it if it exists or create it if it doesn't
	infraEnv := &aiv1beta1.InfraEnv{}
	if err := r.Client.Get(ctx, client.ObjectKey{Name: infraEnvName, Namespace: config.Namespace}, infraEnv); err != nil {
		if apierrors.IsNotFound(err) {
			// Create infraenv
			infraEnv, err = r.createInfraEnv(ctx, config, infraEnvName)
			log.Info("Created infra env", "name", infraEnv.Name, "namespace", infraEnv.Namespace)
			//TODO: make this more efficient
			if !apierrors.IsAlreadyExists(err) {
				log.Error(err, "couldn't create infraenv", "name", config.Name)
				return ctrl.Result{}, err
			}
		} else {
			log.Error(err, "couldn't get infraenv for agentbootstrapconfig", "agentbootstrap config name", config.Name, "infra env name", infraEnv)
			return ctrl.Result{}, err
		}
	}

	// Set infraEnv if not already set
	if config.Spec.InfraEnvRef == nil {
		config.Spec.InfraEnvRef = &corev1.ObjectReference{Name: infraEnv.Name, Namespace: infraEnv.Namespace, Kind: "InfraEnv", APIVersion: infraEnv.APIVersion}
		if err := r.Client.Update(ctx, config); err != nil {
			log.Error(err, "couldn't update agent config", "name", config.Name)
			return ctrl.Result{}, err
		}
	}

	if config.Status.ISODownloadURL == "" {
		return ctrl.Result{}, nil
	}

	// Get the Machine associated with this agentbootstrapconfig
	// TODO: change the way we get this, for now it has the same name but that may not be the case - we should change to fetch by spec's reference to this agentbootstrapconfig
	machine, err := util.GetMachineByName(ctx, r.Client, config.Namespace, config.Name)
	if err != nil {
		log.Error(err, "couldn't get machine associated with agentbootstrapconfig", "name", config.Name)
		return ctrl.Result{}, err
	}

	// Get Metal3 Machine owned by this Machine that is related to this agentbootstrapconfig
	metal3Machine := &metal3.Metal3Machine{}
	if err := r.Client.Get(ctx, types.NamespacedName{Name: machine.Spec.InfrastructureRef.Name, Namespace: machine.Namespace}, metal3Machine); err != nil {
		log.Error(err, "couldn't get metal3machine associated with machine and agentbootstrapconfig", "agentbootstrapconfig name", config.Name, "machine name", machine.Name)
		return ctrl.Result{}, err
	}

	// TODO: check if it's a control plane or worker
	log.Info("Found metal3 machine owned by machine, adding infraenv ISO URL", "name", metal3Machine.Name, "namespace", metal3Machine.Namespace)
	// Add infra env ISO URL from status to Metal3 Machine
	// TODO: check if URL changes and only update if changed
	metal3Machine.Spec.Image.URL = config.Status.ISODownloadURL
	if err := r.Client.Update(ctx, metal3Machine); err != nil {
		log.Error(err, "couldn't update metal3 machine", "name", metal3Machine.Name, "namespace", metal3Machine.Namespace)
		return ctrl.Result{}, err
	}

	log.Info("Finished adding URLs to metal3 machines")

	//	machine.Spec.Image.URL = config.Status.ISODownloadURL

	/* machineDeployments := clusterv1.MachineDeploymentList{}
	if err := r.Client.List(context.Background(), &machineDeployments, client.InNamespace(req.Namespace)); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	for _, md := range machineDeployments.Items {
		if !r.doesMachineDeploymentBelongToUs(md, config) {
			continue
		}
		log.Info("found machine deployment", "name", md.Name, "namespace", md.Namespace)
		machineTemplate := r.getMachineTemplate(ctx, md)
		if machineTemplate == nil {
			log.Info("Machine template not metal3 or not found")
			continue
		}
		log.Info("found machine template", "name", machineTemplate.Name, "namespace", machineTemplate.Namespace)

	} */

	// agentboostrapconfig
	// machinedeployment
	// infraref (machinetemplate)

	// TODO(user): your logic here

	return ctrl.Result{}, nil
}

func (r *AgentBootstrapConfigReconciler) GetInfraEnvName(config *bootstrapv1beta1.AgentBootstrapConfig) (string, error) {
	nameFormat := "%s-%s"
	clusterName, ok := config.Labels[clusterv1.ClusterNameLabel]
	if !ok {
		return "", fmt.Errorf("cluster name label does not exist on agent bootstrap config %s", config.Name)
	}
	_, isControlPlane := config.Labels[clusterv1.MachineControlPlaneLabel]
	if isControlPlane {
		return fmt.Sprintf(nameFormat, clusterName, "control-plane"), nil
	}

	// Otherwise get machine deployment name
	machineDeploymentName, ok := config.Labels[clusterv1.MachineDeploymentNameLabel]
	if !ok {
		return "", fmt.Errorf("machine deployment name label does not exist on agent bootstrap config %s", config.Name)

	}
	return fmt.Sprintf(nameFormat, clusterName, machineDeploymentName), nil
}

func (r *AgentBootstrapConfigReconciler) doesMachineDeploymentBelongToUs(machineDeployment clusterv1.MachineDeployment, config *bootstrapv1beta1.AgentBootstrapConfig) bool {
	return machineDeployment.Spec.Template.Spec.Bootstrap.ConfigRef.Name == config.Name &&
		machineDeployment.Spec.Template.Spec.Bootstrap.ConfigRef.GroupVersionKind() == bootstrapv1beta1.GroupVersion.WithKind("AgentBootstrapConfigSpec")

}

// SetupWithManager sets up the controller with the Manager.
func (r *AgentBootstrapConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&bootstrapv1beta1.AgentBootstrapConfig{}).
		Watches(
			&clusterv1.Machine{},
			handler.EnqueueRequestsFromMapFunc(r.FilterMachine),
		).Complete(r)
}

// Filter machine owned by this agentbootstrapconfig
func (r *AgentBootstrapConfigReconciler) FilterMachine(_ context.Context, o client.Object) []ctrl.Request {
	result := []ctrl.Request{}
	m, ok := o.(*clusterv1.Machine)
	if !ok {
		panic(fmt.Sprintf("Expected a Machine but got a %T", o))
	}
	// m.Spec.ClusterName

	if m.Spec.Bootstrap.ConfigRef != nil && m.Spec.Bootstrap.ConfigRef.GroupVersionKind() == bootstrapv1beta1.GroupVersion.WithKind("AgentBootstrapConfigSpec") {
		name := client.ObjectKey{Namespace: m.Namespace, Name: m.Spec.Bootstrap.ConfigRef.Name}
		result = append(result, ctrl.Request{NamespacedName: name})
	}
	return result
}

func (r *AgentBootstrapConfigReconciler) createInfraEnv(ctx context.Context, config *bootstrapv1beta1.AgentBootstrapConfig, infraEnvName string) (*aiv1beta1.InfraEnv, error) {
	infraEnv := &aiv1beta1.InfraEnv{
		ObjectMeta: metav1.ObjectMeta{
			Name:      infraEnvName,
			Namespace: config.Namespace,
			Labels: map[string]string{
				agentBootstrapConfigLabel: config.Name,
			},
		},
	}

	clusterName, ok := config.Labels[clusterv1.ClusterNameLabel]
	if ok {
		infraEnv.Labels[clusterv1.ClusterNameLabel] = clusterName
	}

	var pullSecret *corev1.LocalObjectReference
	if config.Spec.PullSecretRef != nil {
		pullSecret = config.Spec.PullSecretRef
	} else {
		//TODO: create logic for placeholder pull secret
	}
	infraEnv.Spec = aiv1beta1.InfraEnvSpec{
		PullSecretRef: pullSecret,
		ClusterRef: &aiv1beta1.ClusterReference{
			Name:      config.Spec.ClusterDeploymentRef.Name,
			Namespace: config.Spec.ClusterDeploymentRef.Namespace,
		},
	}

	return infraEnv, r.Create(ctx, infraEnv)
}
