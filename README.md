# kit
Kubernetes' operations made simple. This is done by exposing ready to use
functions & structures.

Built on top of [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime)

## Build & test
- make

### Setup test environment aka envtest
- Install envtest binary
  - refer: [setup-envtest](https://github.com/kubernetes-sigs/controller-runtime/tree/v0.10.3/tools/setup-envtest)
- Run below to download `kubectl`, `apiserver` & `etcd` binaries

```shell
setup-envtest --bin-dir=/usr/local/kubebuilder/bin use
Version: 1.22.1
OS/Arch: darwin/amd64
md5: 0yL+nN2utkMPWXaoStXMKg==
Path: /usr/local/kubebuilder/bin/k8s/1.22.1-darwin-amd64

# verify presence of images
ls -ltr /usr/local/kubebuilder/bin/k8s/1.22.1-darwin-amd64
```

- Move these binaries to the default lookup path used by controller-runtime
  - i.e. `/usr/local/kubebuilder/bin/`

## Motivation
Project kit is based on the experiences we had building & conforming Kubernetes
operators across projects such as [OpenEBS](https://github.com/openebs),
[Metac](https://github.com/AmitKumarDas/metac) & building platforms at organisations
such as [MayaData](https://mayadata.io/) & [JIMDO](https://www.jimdo.com/).

There was a need for a self servicing DevOps platform for teams across the org.
In all the cases, it was decided to use Kubernetes as its substrate since latter
has found wider acceptance in the infrastructure community. However, given the
ever expanding projects within Kubernetes ecosystem, it has become a challenge
to choose, build & **conform** if these selected projects solve the platform's
opertional needs.

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

This module is an attempt to solve above challenges by adopting Kubernetes best
practices & turning them into programmable implementations.

## Use as a library
This project can be used as a library by various out-of-tree golang projects

## References
- TODO
