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

package utils

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo/v2" //nolint:golint,revive
)

const (
	prometheusOperatorVersion = "v0.68.0"
	prometheusOperatorURL     = "https://github.com/prometheus-operator/prometheus-operator/" +
		"releases/download/%s/bundle.yaml"

	certmanagerVersion = "v1.5.3"
	certmanagerURLTmpl = "https://github.com/jetstack/cert-manager/releases/download/%s/cert-manager.yaml"
)

/*
	 var workloads = map[string]func() error{
		"cert-manager": InstallCertManager,
	}
*/
func Install(kubeconfig string) error {
	verb := "apply"
	err := InstallCertManager(kubeconfig)
	if err != nil {
		warnError(err)
		return err
	}
	err = InstallNginxIngress(kubeconfig, verb)
	if err != nil {
		warnError(err)
		return err
	}
	err = InstallAssistedServiceCRDs(kubeconfig, verb)
	if err != nil {
		warnError(err)
		return err
	}
	err = AssistedService(kubeconfig, "kustomize")
	if err != nil {
		warnError(err)
		return err
	}
	err = AgentServiceConfig(kubeconfig, verb)
	if err != nil {
		warnError(err)
		return err
	}
	err = InstallBootstrapProvider(kubeconfig, verb)
	if err != nil {
		warnError(err)
		return err
	}
	err = InstallControlPlaneProvider(kubeconfig, verb)
	if err != nil {
		warnError(err)
		return err
	}
	return nil
}

func Uninstall(kubeconfig string) error {
	verb := "delete"
	err := UninstallCertManager(kubeconfig)
	if err != nil {
		warnError(err)
		return err
	}
	err = InstallNginxIngress(kubeconfig, verb)
	if err != nil {
		warnError(err)
		return err
	}
	err = InstallAssistedServiceCRDs(kubeconfig, verb)
	if err != nil {
		warnError(err)
		return err
	}
	err = AssistedService(kubeconfig, verb)
	if err != nil {
		warnError(err)
		return err
	}
	err = AgentServiceConfig(kubeconfig, verb)
	if err != nil {
		warnError(err)
		return err
	}
	err = InstallBMOIronic(kubeconfig, verb)
	if err != nil {
		warnError(err)
		return err
	}
	err = InstallBootstrapProvider(kubeconfig, verb)
	if err != nil {
		warnError(err)
		return err
	}
	err = InstallControlPlaneProvider(kubeconfig, verb)
	if err != nil {
		warnError(err)
		return err
	}
	return nil
}

func InstallBMOIronic(kubeconfig, verb string) error {
	url := "https://github.com/jianzzha/sylva-poc/manifests/bmo?ref=main"
	cmd := exec.Command("kubectl", "--kubeconfig", kubeconfig, verb, "-k", url)
	if _, err := Run(cmd); err != nil {
		return err
	}
	url = "https://github.com/jianzzha/sylva-poc/manifests/ironic?ref=main"
	cmd = exec.Command("kubectl", "--kubeconfig", kubeconfig, verb, "-k", url)
	if _, err := Run(cmd); err != nil {
		return err
	}
	return nil
}

func warnError(err error) {
	_, _ = fmt.Fprintf(GinkgoWriter, "warning: %v\n", err)
}

// InstallPrometheusOperator installs the prometheus Operator to be used to export the enabled metrics.
func InstallPrometheusOperator() error {
	url := fmt.Sprintf(prometheusOperatorURL, prometheusOperatorVersion)
	cmd := exec.Command("kubectl", "create", "-f", url)
	_, err := Run(cmd)
	return err
}

// Run executes the provided command within this context
func Run(cmd *exec.Cmd) ([]byte, error) {
	/* 	dir, _ := GetProjectDir()
	   	cmd.Dir = dir

	   	if err := os.Chdir(cmd.Dir); err != nil {
	   		_, _ = fmt.Fprintf(GinkgoWriter, "chdir dir: %s\n", err)
	   	} */

	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	command := strings.Join(cmd.Args, " ")
	_, _ = fmt.Fprintf(GinkgoWriter, "running: %s\n", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("%s failed with error: (%v) %s", command, err, string(output))
	}

	return output, nil
}

// UninstallPrometheusOperator uninstalls the prometheus
func UninstallPrometheusOperator() {
	url := fmt.Sprintf(prometheusOperatorURL, prometheusOperatorVersion)
	cmd := exec.Command("kubectl", "delete", "-f", url)
	if _, err := Run(cmd); err != nil {
		warnError(err)
	}
}

// UninstallCertManager uninstalls the cert manager
func UninstallCertManager(kubeconfig string) error {
	url := fmt.Sprintf(certmanagerURLTmpl, certmanagerVersion)
	cmd := exec.Command("kubectl", "--kubeconfig", kubeconfig, "delete", "-f", url)
	if _, err := Run(cmd); err != nil {
		return err
	}
	return nil
}
func InstallAssistedServiceCRDs(kubeconfig, verb string) error {
	// Install assisted service CRDs
	CRDs := []string{
		"https://raw.githubusercontent.com/openshift/assisted-service/master/hack/crds/hive.openshift.io_clusterdeployments.yaml",
		"https://raw.githubusercontent.com/openshift/assisted-service/master/hack/crds/hive.openshift.io_clusterimagesets.yaml",
		"https://raw.githubusercontent.com/openshift/assisted-service/master/hack/crds/metal3.io_baremetalhosts.yaml",
		"https://raw.githubusercontent.com/openshift/assisted-service/master/hack/crds/metal3.io_preprovisioningimages.yaml",
	}
	for _, crd := range CRDs {
		cmd := exec.Command("kubectl", "--kubeconfig", kubeconfig, "apply", "-f", crd)
		if _, err := Run(cmd); err != nil {
			return err
		}
	}
	return nil
}

func AssistedService(kubeconfig, verb string) error {
	// Install assisted service
	cmd := exec.Command("kubectl", "--kubeconfig", kubeconfig, verb, "-k", "https://github.com/openshift/assisted-service/config/default?ref=master")
	if _, err := Run(cmd); err != nil {
		return err
	}
	return nil
}

func UninstallAssistedService(kubeconfig string) error {
	// Install assisted service
	cmd := exec.Command("kubectl", "--kubeconfig", kubeconfig, "delete", "-k", "https://github.com/openshift/assisted-service/config/default?ref=master")
	if _, err := Run(cmd); err != nil {
		return err
	}
	return nil
}

func AgentServiceConfig(kubeconfig, verb string) error {
	cmd := exec.Command("kubectl", "--kubeconfig", kubeconfig, verb, "-f", "https://github.com/openshift/assisted-service/config/default?ref=master")
	if _, err := Run(cmd); err != nil {
		return err
	}
	return nil
}

func InstallNginxIngress(kubeconfig, verb string) error {
	url := "https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml"
	cmd := exec.Command("kubectl", "--kubeconfig", kubeconfig, verb, "-f", url)
	if _, err := Run(cmd); err != nil {
		return err
	}
	return nil
}

func InstallBootstrapProvider(kubeconfig, verb string) error {
	url := "https://raw.githubusercontent.com/openshift-assisted/cluster-api-agent/master/bootstrap-components.yaml"
	cmd := exec.Command("kubectl", "--kubeconfig", kubeconfig, verb, "-f", url)
	if _, err := Run(cmd); err != nil {
		return err
	}
	return nil
}

func InstallControlPlaneProvider(kubeconfig, verb string) error {
	url := "https://raw.githubusercontent.com/openshift-assisted/cluster-api-agent/master/controlplane-components.yaml"
	cmd := exec.Command("kubectl", "--kubeconfig", kubeconfig, verb, "-f", url)
	if _, err := Run(cmd); err != nil {
		return err
	}
	return nil
}

// InstallCertManager installs the cert manager bundle.
func InstallCertManager(kubeconfig string) error {
	url := fmt.Sprintf(certmanagerURLTmpl, certmanagerVersion)
	cmd := exec.Command("kubectl", "--kubeconfig", kubeconfig, "apply", "-f", url)
	if _, err := Run(cmd); err != nil {
		return err
	}
	// Wait for cert-manager-webhook to be ready, which can take time if cert-manager
	// was re-installed after uninstalling on a cluster.
	cmd = exec.Command("kubectl", "wait", "deployment.apps/cert-manager-webhook",
		"--for", "condition=Available",
		"--namespace", "cert-manager",
		"--timeout", "5m",
	)

	_, err := Run(cmd)
	return err
}

// LoadImageToKindCluster loads a local docker image to the kind cluster
func LoadImageToKindClusterWithName(name string) error {
	cluster := "kind"
	if v, ok := os.LookupEnv("KIND_CLUSTER"); ok {
		cluster = v
	}
	kindOptions := []string{"load", "docker-image", name, "--name", cluster}
	cmd := exec.Command("kind", kindOptions...)
	_, err := Run(cmd)
	return err
}

// GetNonEmptyLines converts given command output string into individual objects
// according to line breakers, and ignores the empty elements in it.
func GetNonEmptyLines(output string) []string {
	var res []string
	elements := strings.Split(output, "\n")
	for _, element := range elements {
		if element != "" {
			res = append(res, element)
		}
	}

	return res
}

// GetProjectDir will return the directory where the project is
func GetProjectDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return wd, err
	}
	wd = strings.Replace(wd, "/test/e2e", "", -1)
	return wd, nil
}
