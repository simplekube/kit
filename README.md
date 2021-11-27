# kit
Kubernetes' operations made simple. This is done by exposing ready to use functions & structures.

# Build & test
- make

# Setup envtest
- install envtest by following https://github.com/kubernetes-sigs/controller-runtime/tree/v0.10.3/tools/setup-envtest
- download kubectl, apiserver & etcd binaries
```shell
setup-envtest --bin-dir=/usr/local/kubebuilder/bin use
Version: 1.22.1
OS/Arch: darwin/amd64
md5: 0yL+nN2utkMPWXaoStXMKg==
Path: /usr/local/kubebuilder/bin/k8s/1.22.1-darwin-amd64

ls -ltr /usr/local/kubebuilder/bin/k8s/1.22.1-darwin-amd64
```
- move these binaries to the default lookup path `/usr/local/kubebuilder/bin/`

## History
This was conceptualised in [JIMDO](https://www.jimdo.com/) to build a compliance library for Kubernetes installations   