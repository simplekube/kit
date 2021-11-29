module github.com/simplekube/kit

go 1.16

require (
	github.com/google/go-cmp v0.5.6
	github.com/hashicorp/go-multierror v1.1.1
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.7.0
	k8s.io/api v0.22.4
	k8s.io/apimachinery v0.22.4
	k8s.io/client-go v0.22.4
	k8s.io/klog/v2 v2.30.0 // indirect
	sigs.k8s.io/cli-utils v0.26.1
	sigs.k8s.io/controller-runtime v0.10.3
)

replace (
	github.com/go-logr/logr => github.com/go-logr/logr v0.4.0
	k8s.io/klog/v2 => k8s.io/klog/v2 v2.10.0
)
