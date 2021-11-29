module k8s.tests

go 1.16

replace (
	github.com/go-logr/logr => github.com/go-logr/logr v0.4.0
	github.com/simplekube/kit => ../..
	k8s.io/klog/v2 => k8s.io/klog/v2 v2.10.0
)

require (
	github.com/pkg/errors v0.9.1
	github.com/simplekube/kit v0.0.0-00010101000000-000000000000
	k8s.io/api v0.22.4
	k8s.io/apimachinery v0.22.4
	sigs.k8s.io/controller-runtime v0.10.3
)
