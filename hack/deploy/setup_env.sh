#!/bin/bash

set -o nounset
set -o pipefail
set -o errexit
set -o xtrace

function setup_kind_cluster() {
    kind create cluster --config=artifacts/kind-config.yaml
}

function setup_nginx() {
    kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml
}

function setup_cert_manager(){
    kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.15.1/cert-manager.yaml
}

function setup_assisted_service() {
    if [ -d ]
    cd ${ASSISTED_SERVICE_REPO}
    # apply CRDs
    kubectl apply -f hack/crds/*
    kustomize build config/default/ | kubectl apply -f -
    kubectl apply -f artifacts/agent-service-config.yaml
}

function setup_capi() {
    # quick start installs capi, cert manager, and capm3
    clusterctl init --core cluster-api:v1.7.0-rc.1 \
    --bootstrap kubeadm:v1.7.0-rc.1 \
    --control-plane kubeadm:v1.7.0-rc.1 -v5
    clusterctl init --infrastructure metal3
}

function setup_env() {
    setup_kind_cluster
    setup_nginx
    setup_capi
    setup_assisted_service
}
