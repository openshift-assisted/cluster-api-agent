package v1alpha1

import clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

const (
	InfraEnvFailedReason                                          = "InfraEnvFailed"
	PropagatingLiveISOURLFailedReason                             = "PropagatingLiveISOURLFailed"
	CreatingSecretFailedReason                                    = "CreatingSecretFailed"
	WaitingForLiveISOURLReason                                    = "WaitingForLiveISOURL"
	WaitingForInstallCompleteReason                               = "WaitingForInstallComplete"
	WaitingForAssistedInstallerReason                             = "WaitingForAssistedInstaller"
	WaitingForClusterInfrastructureReason                         = "WaitingForClusterInfrastructure"
	DataSecretAvailableCondition          clusterv1.ConditionType = "DataSecretAvailable"
	OpenshiftAssistedConfigLabel                                  = "bootstrap.cluster.x-k8s.io/openshiftAssistedConfig"
)
