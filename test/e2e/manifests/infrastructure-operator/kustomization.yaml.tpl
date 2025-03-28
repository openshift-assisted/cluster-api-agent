apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - https://raw.githubusercontent.com/openshift/assisted-service/9964f0870d5df042a782bb9c6394835d05ad807a/hack/crds/hive.openshift.io_clusterdeployments.yaml
  - https://raw.githubusercontent.com/openshift/assisted-service/9964f0870d5df042a782bb9c6394835d05ad807a/hack/crds/hive.openshift.io_clusterimagesets.yaml
  - https://github.com/openshift/assisted-service/config/default?ref=v2.38.1

# 07/24/24 build
images:
  - name: quay.io/edge-infrastructure/assisted-service
    newName: quay.io/edge-infrastructure/assisted-service
    digest: ${ASSISTED_SERVICE_IMAGE}

patches:
  - target:
      group: apps
      version: v1
      kind: Deployment
      name: infrastructure-operator
      namespace: assisted-installer
    patch: |-
      - op: replace
        path: /spec/template/spec/containers/0/env/0/value
        #value: quay.io/edge-infrastructure/assisted-service@sha256:7d1a1739abcb331ba2229fee5fc102256ac2d9bad813d467419131cb04f00948
        #value: quay.io/edge-infrastructure/assisted-service@sha256:db6a010640b5039ed251a1df255d8d08e2f79ac87ace9d7bf1e9cfa2f23e9def
        value: ${ASSISTED_SERVICE_IMAGE}
      - op: replace
        path: /spec/template/spec/containers/0/env/1/value
        value: quay.io/edge-infrastructure/assisted-service-el8@sha256:bb487fc7121a26be374792ccb6eeed7ac8a521fa78876eec87807a5da7da5121
      - op: replace
        path: /spec/template/spec/containers/0/env/2/value
        value: ${ASSISTED_IMAGE_SERVICE_IMAGE}
      - op: replace
        path: /spec/template/spec/containers/0/env/3/value
        value: quay.io/sclorg/postgresql-12-c8s@sha256:663089471e999a4175341ac4d97dcff9cd15ec5f2e96b2309dc8de806106198b
      - op: replace
        path: /spec/template/spec/containers/0/env/4/value
        value: ${ASSISTED_INSTALLER_AGENT_IMAGE}
      - op: replace
        path: /spec/template/spec/containers/0/env/5/value
        value: ${ASSISTED_INSTALLER_CONTROLLER_IMAGE}
      - op: replace
        path: /spec/template/spec/containers/0/env/6/value
        value: ${ASSISTED_INSTALLER_IMAGE}

  - target:
      version: v1
      kind: Namespace
    patch: |-
      - op: replace
        path: /metadata/labels/pod-security.kubernetes.io~1enforce
        value: baseline

labels:
  - pairs:
      clusterctl.cluster.x-k8s.io: ""
