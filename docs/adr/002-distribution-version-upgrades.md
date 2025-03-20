# ADR: Using distributionVersion to Drive Upgrades

## Status

Proposed

## Context

Cluster API contracts generally expect upgrades driven by the Version field, which represents the kubernetes version. However, OpenShift versions cannot be mapped directly to kubernetes versions. Furthermore, CAPI expects each Machine group (controlplane, each MD) to have their independent version. OpenShift does not support in-place upgrades under this model. Instead, the upgrade process must be centrally managed and coordinated through a specific field indicating the desired target version.

The proposed `distributionVersion` field (see: [CAPI PR #11029](https://github.com/kubernetes-sigs/cluster-api/pull/11029)) has been introduced to address the lack of direct mapping between kubernetesVersion and OpenShift versions.

However, we cannot drive upgrades by different sets of Machines (ControlPlane, MachineDeployments, etc.). OpenShift upgrades are centralized and managed exclusively through the Cluster Version Operator, meaning a unified approach is required. Therefore, it is necessary to treat the `distributionVersion` as representative of the entire cluster's version, not just a specific Machine group.

Currently, we are setting the `distributionVersion` field on the ControlPlane resource. However, we semantically mean the "cluster version", not just the ControlPlane’s `distributionVersion`. Ideally, this field should be placed on the Cluster object, which would provide a more accurate representation of the entire cluster’s desired state.

This is effectively an in-place upgrade

If discussions with the upstream community are successful, we hope to see CORE CAPI adopt changes to accommodate our upgrade flow. This would allow us to move the `distributionVersion` field to the Cluster object (or a similar place) to better comply with CAPI contracts.

## Decision

- Continue to use `distributionVersion` to track OpenShift versions and drive upgrades.
- Centralize upgrade handling by associating the `distributionVersion` field with the ControlPlane resource until upstream consensus is reached.
- Treat upgrade as in-place
- Pursue upstream discussions to enable placing `distributionVersion` on the Cluster object for better compatibility and adherence to CAPI contracts.

## Consequences

- Diverges from the standard CAPI upgrade contract.
- Current approach forces `distributionVersion` changes on the ControlPlane resource to represent the whole cluster’s version, which is not ideal.
- Risk of inconsistency if upstream changes are not accepted or if future changes conflict with our approach.
- No version status available for Machines not controlled by the Openshift Assisted ControlPlane controller
