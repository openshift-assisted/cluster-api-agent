#!/bin/bash

set -o nounset
set -o pipefail
set -o errexit
set -o xtrace

function install_helm() {
    curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3
    chmod 700 get_helm.sh
    ./get_helm.sh
}

function install_go() {
    wget https://go.dev/dl/go1.21.9.linux-amd64.tar.gz
    sudo tar -C /usr/local -xzf go1.21.9.linux-amd64.tar.gz 
    echo "export PATH=$PATH:/usr/local/go/bin" >> ~/.bashrc
}

function install_kubectl() {
    curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
    sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl
}

function install_clusterctl() {
    curl -L https://github.com/kubernetes-sigs/cluster-api/releases/download/v1.7.3/clusterctl-linux-amd64 -o clusterctl
    sudo install -o root -g root -m 0755 clusterctl /usr/local/bin/clusterctl
}

function install_kind() {
	if command -v kind  > /dev/null 2>&1 ; then
        echo "Kind $(kind version) already installed"
		return
	fi

    if [ -z "${KIND_VERSION:-}" ]; then
        echo "KIND_VERSION must be set"
        return
    fi

	echo "Installing Kind version $KIND_VERSION"
	sudo curl --retry 5 --connect-timeout 30 -L https://kind.sigs.k8s.io/dl/v$KIND_VERSION/kind-linux-amd64 -o /usr/local/bin/kind
	sudo chmod +x /usr/local/bin/kind
	echo "Installed Kind successfully!"
}

function install_tools() {
    install_go
    install_kind
    install_kubectl
    install_clusterctl
}