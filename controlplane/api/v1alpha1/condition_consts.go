package v1alpha1

import clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

const (
	// MachinesCreatedCondition documents that the machines controlled by the OpenshiftControlplane are created.
	// When this condition is false, it indicates that there was an error when cloning the infrastructure/bootstrap template or
	// when generating the machine object.
	MachinesCreatedCondition clusterv1.ConditionType = "MachinesCreated"

	// InfrastructureTemplateCloningFailedReason (Severity=Error) documents a OpenshiftControlplane failing to
	// clone the infrastructure template.
	InfrastructureTemplateCloningFailedReason = "InfrastructureTemplateCloningFailed"

	// BootstrapTemplateCloningFailedReason (Severity=Error) documents a OpenshiftControlplane failing to
	// clone the bootstrap template.
	BootstrapTemplateCloningFailedReason = "BootstrapTemplateCloningFailed"

	// MachineGenerationFailedReason (Severity=Error) documents a OpenshiftControlplane failing to
	// generate a machine object.
	MachineGenerationFailedReason = "MachineGenerationFailed"
)
