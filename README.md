# kit
Kubernetes' operations made simple. This is done by exposing ready to use 
functions & structures. This is built on top of [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime) 
library.

# Build & test
- make

# Setup envtest
- Install envtest binary
  - refer: https://github.com/kubernetes-sigs/controller-runtime/tree/v0.10.3/tools/setup-envtest
- Run below to download kubectl, apiserver & etcd binaries
```shell
setup-envtest --bin-dir=/usr/local/kubebuilder/bin use
Version: 1.22.1
OS/Arch: darwin/amd64
md5: 0yL+nN2utkMPWXaoStXMKg==
Path: /usr/local/kubebuilder/bin/k8s/1.22.1-darwin-amd64

# verify
ls -ltr /usr/local/kubebuilder/bin/k8s/1.22.1-darwin-amd64
```
- Move these binaries to the default lookup path 
  - i.e. `/usr/local/kubebuilder/bin/`

## Motivation
Project kit was conceptualised in [JIMDO](https://www.jimdo.com/) to build a 
compliance library for Kubernetes installations

There was a need for a self servicing DevOps platform for teams across JIMDO. 
It decided to use Kubernetes as its substrate since latter has found wider 
acceptance in the infrastructure community. However, given the ever expanding 
projects within Kubernetes ecosystem, it has become a challenge to choose, build
& conform if these selected projects solve platform's needs.

Infrastructure should be in a position to answer questions like:

- "Will migration of Linkerd from v1 to v2 result in outages of all services
  dependent on Wonderland?",
- "Is there a way to assert swapping one service mesh with another to work as
  expected?"
- "How to measure performance between two versions of a service?" & so on

The platform's dependency on Kubernetes has brought in the complexity associated
with its controllers. Kubernetes' controllers are eventually consistent. They
work continuously to merge the desired state against the cluster's actual state.
Making these controllers to work as expected has been a big concern. Bugs often
leak into production unless the team has in-depth knowledge in building these
controllers. It is expected to have a thorough knowledge of Kubernetes native
architectural designs like:

- 3-way merge (i.e. Apply) & patch operations
- labels, annotations, sidecars, init containers,
- services, ports, metrics,
- rbac & service accounts,
- sub-resource APIs (i.e. status, scale, etc.),
- writing idempotent code which can often lead to remapping existing action
  based APIs to their idempotent equivalents,
- & so on.

Above problem gets amplified when a custom operator is dependent on a bunch of
Kubernetes as well third party operators each maintaining their own state &
dependencies.

This module is an attempt to solve above challenges by adopting Kubernetes best
practices & turning them into programmable implementations. It should be able to
detect issues within the Kubernetes system proactively, report misconfigurations
if any, analyse performance drifts, & so on. This should be able to consume
existing libraries that solve some of these problems. This module may eventually
offer product teams to assert their product's operational needs even before
running the same in production.

## Use as a library
This project can be used as a library by various out-of-tree golang projects

## References
- TODO

