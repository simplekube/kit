### About
K8s-experiments showcases ways to implement Kubernetes end-to-end tests using kit as a go binary

### Steps to run k8s-experiments as a Kubernetes Pod
_Note: Below steps expect Docker & k3d to be installed_
_Note: Below steps clean up previous binary, image & Kubernetes artifact(s)_
_Note: Below order should be maintained_

- make k3d-push
- make run
- make logs

### Troubleshooting
- verify if `make binary` works
- verify if `make image` works
- k3d cluster list
- docker ps
- kubectl get no
- kubectl get ns
- kubectl get po -A
- kubectl -n e2e-system logs e2e // same as `make logs`
