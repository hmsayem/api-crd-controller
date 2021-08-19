### About
This repository implements a simple controller for watching server resources as defined with a CustomResourceDefinition (CRD).

### Code Generation
The `update-codegen` script will automatically generate the following files &
directories:

* `pkg/apis/hmsayem.com/v1/zz_generated.deepcopy.go`
* `pkg/client/`
### Fetch server-controller and its dependencies
```sh
git clone https://github.com/hmsayem/server-controller.git
cd server-controller
```
### How to Run

**Prerequisite**: Since the sample-controller uses `apps/v1` deployments, the Kubernetes cluster version should be greater than 1.9.

```sh
# assumes you have a working kubeconfig, not required if operating in-cluster
go build -o server-controller .
./server-controller -kubeconfig=$HOME/.kube/config

# create a CustomResourceDefinition
kubectl create -f artifacts/crd.yaml

# create a custom resource of type Foo
kubectl create -f artifacts/server.yaml

# check deployments created through the custom resource
kubectl get deployments
```