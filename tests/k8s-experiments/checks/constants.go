package checks

const (
	// EnvKeyE2eSuiteNamespace is the environment variable
	// that holds the namespace of the experiments suite
	EnvKeyE2eSuiteNamespace = "E2E_SUITE_NAMESPACE"
)

// These constants represent environment variables to enable or disable
// various checks. Each environment variable represents one check.
const (
	EnvKeyEnableIsK8sDeployIdempotent  = "ENABLE_IS_K8S_DEPLOY_IDEMPOTENT"
	EnvKeyEnableDoesK8sDeployPropagate = "ENABLE_DOES_K8S_DEPLOY_PROPAGATE"
	EnvKeyEnableDoesK8sDNSWork         = "ENABLE_DOES_K8S_DNS_WORK"
	EnvKeyEnableDoesK8sHPAWork         = "ENABLE_DOES_K8S_HPA_WORK"
)
