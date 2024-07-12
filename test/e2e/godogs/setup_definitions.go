package godogs

import (
	"context"
	"fmt"
	"os"

	"github.com/cucumber/godog"
	"github.com/openshift-assisted/cluster-api-agent/test/utils"
	"github.com/thoas/go-funk"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type Installer struct {
	Client    client.Client
	Workloads []string
}

func (i *Installer) kubernetesClusterExists(ctx context.Context, exists string) error {
	kubeconfig := os.Getenv("KUBECONFIG")
	config := ctrl.GetConfigOrDie()
	if exists == "a" && kubeconfig == "" && config == nil {
		return fmt.Errorf("no KUBECONFIG set for existing Kubernetes cluster")
	}
	mgr, err := ctrl.NewManager(config, manager.Options{})
	if err != nil {
		return fmt.Errorf("failed to create new client")
	}
	i.Client = mgr.GetClient()
	return nil
}

func (i *Installer) workloadsToInstall(ctx context.Context, whichAreInstalled string) error {
	if whichAreInstalled == "no" {
		i.Workloads = []string{"cert-manager", "assisted-installer", "capi", "controlplane", "bootstrap", "capm3"}
		podList := &corev1.PodList{}
		i.Client.List(ctx, podList)
		for _, pod := range podList.Items {
			if funk.ContainsString(i.Workloads, pod.Name) {
				return fmt.Errorf("workload (%s) exists when it shouldn't", pod.Name)
			}
		}
	}
	return nil
}

func (i *Installer) installWorkloads(ctx context.Context) {
	for _, workload := range i.Workloads {
		utils.Install(workload)
	}

}

func (i *Installer) workloadsSuccessfullyDeployed(ctx context.Context) error {
	for w := 0; w < len(i.Workloads); {
		wl := i.Workloads[w]
		pod := &corev1.Pod{}
		err := i.Client.Get(ctx, client.ObjectKey{Name: wl, Namespace: wl}, pod)
		if err != nil {
			return fmt.Errorf("failed getting workload %s", wl)
		}
		if pod.Status.Phase != "Running" {
			continue
		}
		w++
	}
	return nil
}

func InitializeScenario(ctx *godog.ScenarioContext) {
	installer := &Installer{}
	ctx.Given(`^[a-z]+ Kubernetes cluster$`, installer.kubernetesClusterExists)
	ctx.When(`^[a-z]+ workloads are installed$`, installer.workloadsToInstall)
	ctx.Then(`^I want to install [a-z]+ workloads on the Kubernetes cluster$`, installer.installWorkloads)
	ctx.Step(`^check that [a-z]+ successfully running$`, installer.workloadsSuccessfullyDeployed)
}
