package controller

import (
	"context"
	v1beta12 "github.com/metal3-io/cluster-api-provider-metal3/api/v1beta1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/cluster-api/util"
	"strings"

	"github.com/openshift-assisted/cluster-api-agent/bootstrap/api/v1beta1"
	aiv1beta1 "github.com/openshift/assisted-service/api/v1beta1"
	hivev1 "github.com/openshift/hive/apis/hive/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	clusterNameKey = ".spec.clusterName"
)

// AgentReconciler reconciles an Agent object
type AgentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *AgentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&aiv1beta1.Agent{}).
		Complete(r)
}

// Add ignition override for:
/* CUSTOM_KUBELET_LABELS
Add ignition override to the agent CR that creates this file:
/etc/systemd/system/kubelet.service.d/30-capi-provider-env.conf
[Service]
Environment="CUSTOM_KUBELET_LABELS=metal3.io/uuid=<bmh uid>"

touch file in
/run/cluster-api/bootstrap-success.complete
when provisioning complete
*/
func (r *AgentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	agent := &aiv1beta1.Agent{}
	err := r.Get(ctx, req.NamespacedName, agent)
	if err != nil {
		log.Error(err, "unable to fetch Agent")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// if we find an agent, we must ensure it is controlled by our provider
	clusterDeploymentKey := client.ObjectKey{
		Namespace: agent.Spec.ClusterDeploymentName.Namespace,
		Name:      agent.Spec.ClusterDeploymentName.Name,
	}
	clusterDeployment := &hivev1.ClusterDeployment{}
	if err := r.Client.Get(ctx, clusterDeploymentKey, clusterDeployment); err != nil {
		log.Error(err, "unable to fetch Agent")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	clusterName, ok := clusterDeployment.Labels[clusterv1.ClusterNameLabel]
	if !ok {
		log.Error(err, "clusterdeployment does not belong to a CAPI cluster")
		return ctrl.Result{}, nil
	}
	agentBootstrapConfigList := v1beta1.AgentBootstrapConfigList{}
	if err := r.Client.List(ctx, &agentBootstrapConfigList, client.MatchingLabels{clusterv1.ClusterNameLabel: clusterName}); err != nil {
		log.Error(err, "agentboostrapconfig not found for cluster", "cluster", clusterName)
		return ctrl.Result{}, err
	}
	machines, err := getMachinesByCluster(ctx, r.Client, clusterName)
	if err != nil {
		log.Error(err, "could not retrieve machines")
		return ctrl.Result{}, err
	}
	log.Info("found machines", "cluster", clusterName, "number", len(machines.Items))

	metal3Machines := v1beta12.Metal3MachineList{}
	err = listByCluster(ctx, r.Client, &metal3Machines, clusterName)
	if err != nil {
		log.Error(err, "could not retrieve list of metal3machines")
		return ctrl.Result{}, err
	}

	// machine knows about role
	// for each machine, find its owned metal3machine (it knows about matching agent)
	for _, iface := range agent.Status.Inventory.Interfaces {
		for _, addr := range iface.IPV4Addresses {
			parts := strings.Split(addr, "/")
			if len(parts) > 1 {
				//var m3m *v1beta12.Metal3Machine
				m3m, err := getMachineByMatchingIP(metal3Machines, parts[0])
				if err != nil {
					log.Error(err, "could not match any metal3machine to IP", "ip", parts[0], "cluster", clusterName)
				}
				m, err := r.getMachineOwner(ctx, *m3m)
			}
		}
	}
	//
	agent.Spec.Role = "whatever"
	agent.Spec.IgnitionConfigOverrides = ""
	agent.Spec.Approved = true
	if err := r.Client.Update(ctx, agent); err != nil {
		log.Error(err, "couldn't update agent", "name", agent.Name, "namespace", agent.Namespace)
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *AgentReconciler) getMachineOwner(ctx context.Context, m3machine v1beta12.Metal3Machine) (*clusterv1.Machine, error) {
	log := ctrl.LoggerFrom(ctx)

	machine := clusterv1.Machine{}
	for _, ref := range m3machine.OwnerReferences {
		if ref.Kind == machine.GroupVersionKind().Kind && ref.APIVersion == machine.GroupVersionKind().GroupVersion().String() {
			log.Info("found owner", "name", ref.Name, "namespace", m3machine.Namespace)
			if err := r.Client.Get(ctx, types.NamespacedName{
				Namespace: ref.Name,
				Name:      m3machine.Namespace,
			}, &machine); err != nil {
				return nil, err
			}
			return &machine, nil
		}
	}
	return nil, nil
}

func getOwnedMetal3Machine(machine clusterv1.Machine, metal3Machines v1beta12.Metal3MachineList) *v1beta12.Metal3Machine {
	for _, metal3machine := range metal3Machines.Items {
		if util.IsOwnedByObject(&machine, &metal3machine) {
			return &metal3machine
		}
	}
	return nil
}

func getMachinesByCluster(ctx context.Context, c client.Client, clusterName string) (*clusterv1.MachineList, error) {
	log := ctrl.LoggerFrom(ctx)

	machines := &clusterv1.MachineList{}
	if err := c.List(ctx, machines, client.MatchingFields{clusterNameKey: clusterName}); err != nil {
		log.Error(err, "couldn't get machines associated with cluster", "cluster", clusterName)
		return nil, err
	}
	return machines, nil
}

func listByCluster(ctx context.Context, c client.Client, list client.ObjectList, clusterName string) error {
	if err := c.List(ctx, list, client.MatchingLabels{clusterv1.ClusterNameLabel: clusterName}); err != nil {
		return err
	}
	return nil
}

func getListByLabel(ctx context.Context, c client.Client, list client.ObjectList, clusterName string) error {
	log := ctrl.LoggerFrom(ctx)

	if err := c.List(ctx, list, client.MatchingLabels{clusterv1.ClusterNameLabel: clusterName}); err != nil {
		log.Error(err, "couldn't list objects associated with cluster")
		return err
	}
	return nil
}

func getMetal3MachinesByLabel(ctx context.Context, c client.Client, clusterName string) (*v1beta12.Metal3MachineList, error) {
	log := ctrl.LoggerFrom(ctx)

	machines := v1beta12.Metal3MachineList{}
	if err := c.List(ctx, &machines, client.MatchingLabels{clusterv1.ClusterNameLabel: clusterName}); err != nil {
		log.Error(err, "couldn't get metal3 associated with cluster", "cluster", clusterName)
		return nil, err
	}
	return &machines, nil
}

func getMachinesByLabel(ctx context.Context, c client.Client, clusterName string) (*clusterv1.MachineList, error) {
	log := ctrl.LoggerFrom(ctx)

	machines := clusterv1.MachineList{}
	if err := c.List(ctx, &machines, client.MatchingLabels{clusterv1.ClusterNameLabel: clusterName}); err != nil {
		log.Error(err, "couldn't get metal3 associated with cluster", "cluster", clusterName)
		return nil, err
	}
	return &machines, nil
}
