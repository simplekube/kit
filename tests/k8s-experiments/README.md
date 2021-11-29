### k8s-experiments
K8s-experiments showcases ways to implement Kubernetes end-to-end tests as:
- a go binary
- go test

### Steps to run this as a binary inside a Kubernetes Pod
_Note: Below steps expect Docker & k3d to be installed_
_Note: Below steps clean up previous binary, image & Kubernetes artifact(s)_
_Note: Below order should be maintained_

- make k3d-push
- make run
- make logs

### Commands useful for debugging
- verify if `make binary` works
- verify if `make image` works
- k3d cluster list
- docker ps
- kubectl get no
- kubectl get ns
- kubectl get po -A
- kubectl -n e2e-system logs e2e // same as `make logs`