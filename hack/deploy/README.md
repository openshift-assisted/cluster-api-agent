# Deploy Folder

Contains scripts for setting up the environment and deploying the cluster-api-openshift-baremetal controllers.

## E2E Testing

1. Install Kind cluster with ingress enabled and ports opened (see ./artifacts/kind-config.yaml)
2. Install workloads:
    - assisted-service (https://github.com/openshift/assisted-service/blob/master/docs/dev/operator-on-kind.md)
      1. Deploy CRDs
      2. Deploy cert-manager
      3. Deploy nginx ingress
      4. Deploy infrastructure operator
      5. Create agentserviceconfig 
    - capi and capm3
    - baremetal operator
    - ironic

3. Install our operator + capm3 + capi
    ````
    mkdir -p $XDG_CONFIG_HOME/cluster-api/
    cat <<EOF>$XDG_CONFIG_HOME/cluster-api/clusterctl.yaml
    providers:
    - name: "openshift-agent"
        url: "https://github.com/openshift-assisted/cluster-api-agent/releases/latest/download/bootstrap-components.yaml"
        type: "BootstrapProvider"
    - name: "openshift-agent"
        url: "https://github.com/openshift-assisted/cluster-api-agent/releases/latest/download/controlplane-components.yaml"
        type: "ControlPlaneProvider"
    EOF
    ````
    ````
    clusterctl init --bootstrap openshift-agent --control-plane openshift-agent -i  metal3:v1.7.0 --config $XDG_CONFIG_HOME/cluster-api/clusterctl.yaml
    ````
    
4. Create libvirt VMs and BMHs
5. Apply our CRs for testing