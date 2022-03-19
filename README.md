# kit
Kubernetes' operations made simple. This is done by exposing ready to use
functions.

## Motivation
Project kit is based on the experiences we had building & conforming Kubernetes
operators across projects such as [OpenEBS](https://github.com/openebs),
[Metac](https://github.com/AmitKumarDas/metac) & building platforms at 
organisations such as [MayaData](https://mayadata.io/) & 
[JIMDO](https://www.jimdo.com/).

There was a need for utility functions to solve higher order challenges faced by
teams using Kubernetes.

Take for example the following challenges faced by the platform team:
- Will migration of Linkerd from version A to B result in outages of all services dependent on Linkerd?
- Is there a way to assert running of existing services when their underlying service mesh is swapped with another?
- How to compare performance between two releases of a given service?

This project is an attempt to solve above challenges by exposing Kubernetes features
into **atomic** implementations. Teams can then compose them as building blocks
that solves their problems.

## Use as a library
This project can be used as a library by various out-of-tree golang projects

## Build & Test
- make

### Setup test environment aka envtest
- Install envtest binary
  - refer: [setup-envtest](https://github.com/kubernetes-sigs/controller-runtime/tree/v0.10.3/tools/setup-envtest)
- Run below to download `kubectl`, `apiserver` & `etcd` binaries
```shell
setup-envtest --bin-dir=/usr/local/kubebuilder/bin use
```
- On Mac, you get similar output
```shell
Version: 1.22.1
OS/Arch: darwin/amd64
md5: 0yL+nN2utkMPWXaoStXMKg==
Path: /usr/local/kubebuilder/bin/k8s/1.22.1-darwin-amd64
```
- Verify presence of images
```shell
ls -ltr /usr/local/kubebuilder/bin/k8s/1.22.1-darwin-amd64
```
- Move these binaries to the default lookup path used by controller-runtime
```shell
mv /usr/local/kubebuilder/bin/k8s/1.22.1-darwin-amd64/* /usr/local/kubebuilder/bin/
```

## References
- https://github.com/banzaicloud/k8s-objectmatcher
