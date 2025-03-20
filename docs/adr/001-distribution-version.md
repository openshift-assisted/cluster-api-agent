# ADR: Handling CAPI Contracts Discrepancy Using distributionVersion

## Status

Proposed

## Context

Cluster API (CAPI) contracts rely heavily on the kubernetesVersion field to define Kubernetes versioning and manage upgrades. However, when dealing with OpenShift, there is no direct one-to-one mapping between the kubernetesVersion and OpenShift’s versioning scheme. Because of this, we had to introduce a custom field named `distributionVersion`.

This discrepancy necessitates an alternative approach to accurately capture the version of the distribution deployed. The introduction of this field allows compatibility with OpenShift’s unique versioning scheme while still adhering to CAPI’s framework as closely as possible. Specifics of this problem are discussed in detail in the upstream issue: [CAPI Issue #11816](https://github.com/kubernetes-sigs/cluster-api/issues/11816).

## Decision

We will introduce the `distributionVersion` field as a custom extension to the relevant CAPI objects, enabling accurate version tracking for OpenShift deployments. The rationale behind this approach is to provide a reliable and consistent way to capture the version information of OpenShift that cannot be represented by kubernetesVersion alone.

## Consequences

- This approach diverges from the CAPI contract but is necessary to support OpenShift versioning.
- It requires ongoing maintenance of this custom field until it is potentially adopted upstream.
- Upstream adoption of the `distributionVersion` field would allow for better compatibility and a reduction in discrepancies with CAPI contracts.
