# kit
Kubernetes' operations made simple. This is done by exposing ready to use functions & structures.

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

## History
This was conceptualised in [JIMDO](https://www.jimdo.com/) to build a compliance library for Kubernetes installations   