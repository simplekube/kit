# kit
Kubernetes' operations made simple. This is done by exposing ready to use
functions.

A very thin wrapper over [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime)

## Motivation
Project kit is based on the experiences we had building & conforming Kubernetes
operators across projects such as [OpenEBS](https://github.com/openebs),
[Metac](https://github.com/AmitKumarDas/metac) & building platforms at 
organisations such as [MayaData](https://mayadata.io/) & 
[JIMDO](https://www.jimdo.com/).

There was a need for utility functions to solve bigger challenges faced by teams
using Kubernetes.

Take for example following questions that an infrastructure platform should be
able to answer confidently:

- Will migration of Linkerd from v1 to v2 result in outages of all services
  dependent on the infrastructure?
- Is there a way to assert swapping one service mesh with another to work as
  expected?
- How to measure performance between two versions of a service?

The platform's dependency on Kubernetes has brought in the complexity associated
with its controllers. Kubernetes' controllers are eventually consistent. They
work continuously to merge the desired state against the cluster's actual state.
Making these controllers to work as expected has been a big concern. Bugs often
leak into production unless the team has in-depth knowledge in building these
controllers. It is expected to have a thorough knowledge of Kubernetes native
concepts like:

- 3-way merge (i.e. Apply)
- patch operations
- writing idempotent reconciliation code,
- & so on.

Above problem gets amplified when a custom operator is dependent on a bunch of
Kubernetes as well third party operators each maintaining their own state &
dependencies.

This project is an attempt to solve above challenges by adopting Kubernetes best
practices & turning them into programmable implementations.

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
