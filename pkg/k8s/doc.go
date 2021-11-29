// Package k8s provides the building blocks that can be used to execute
// Kubernetes API call and in turn result in building higher order
// implementations of health checks, policy checks, integration testing,
// end to end testing, etc.
//
// These are the references which were studied while implementing this package
//
// - https://github.com/kubernetes/community/blob/master/contributors/devel/sig-testing/e2e-tests.md
// - https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md
// - https://docs.google.com/document/d/11JKqcnUOrw5Lk98f_ylJXBXyxWSW1z3CZu27OLX1CbM/ - E2E Test Framework 2 Design
// - https://docs.google.com/document/d/1ZtN58kU8SKmDDkxeBKxR9Un76eqhszCVcZuqhs-fLIU/ - K8sClient K8s library
// - https://cluster-api.sigs.k8s.io/developer/e2e.html
// - https://www.eficode.com/blog/testing-kubernetes-deployments-within-ci-pipelines
// - https://blog.mayadata.io/testing-kubernetes-operators
// - https://rancher.com/blog/2020/kubernetes-security-vulnerabilities
// - https://github.com/stakater/tronador - On demand dynamic test environments on K8s
// - https://d2iq.com/blog/running-kind-inside-a-kubernetes-cluster-for-continuous-integration
// - https://jpetazzo.github.io/2015/09/03/do-not-use-docker-in-docker-for-ci/
// - https://github.com/yannh/kubeconform
// - https://github.com/chaos-mesh/chaos-mesh
// - https://kodfabrik.com/journal/a-good-makefile-for-go
// - https://github.com/banzaicloud/k8s-objectmatcher/tree/master/tests
// - https://github.com/linkerd/linkerd2/tree/main/pkg/healthcheck
package k8s
