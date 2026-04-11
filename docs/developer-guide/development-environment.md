# Setting Up the Development Environment

## Required Tools Overview

You will need to install the following tools with the specified minimum versions:

* Git (v2.0.0+)
* Go (version specified in `go.mod` - check with `go version`)
* Docker (v20.10.0+) Or Podman (v3.0.0+)
* Kind (v0.11.0+) Or Minikube (v1.23.0+) Or K3d (v5.7.3+)



## Install Required Tools

### Install Git

Obviously, you will need a `git` client for pulling source code and pushing back your changes.

<https://github.com/git-guides/install-git>


### Install Go

You will need a Go SDK and related tools (such as GNU `make`) installed and working on your development environment.

<https://go.dev/doc/install/>

Install Go with a version equal to or greater than the version listed in `go.mod` (verify go version with `go version`).  
We will assume that your Go workspace is at `~/go`.

Verify: run `go version`

### Install Docker or Podman

#### Installation guide for docker

<https://docs.docker.com/engine/install/>

You will need a working Docker runtime environment, to be able to build and run images. Athena is using multi-stage builds. 

Verify: run `docker version`

#### Installation guide for podman

<https://podman.io/docs/installation>

### Install a Local K8s Cluster

You won't need a fully blown multi-master, multi-node cluster, but you will need something like K3S, K3d, Minikube, Kind or microk8s. You will also need a working Kubernetes client (`kubectl`) configuration in your development environment. The configuration must reside in `~/.kube/config`.

#### Kind(NOT RECOMMENDED)

##### [Installation guide](https://kind.sigs.k8s.io/docs/user/quick-start)

You can use `kind` to run Kubernetes inside Docker. But pointing to any other development cluster works fine as well as long as Athena can reach it.

##### Start the Cluster
```shell
kind create cluster
```

If `kind create cluster` fails with an error similar to `failed to lock config file: open ~/.kube/config.lock: permission denied`, your `~/.kube` directory is likely owned by `root` or otherwise not writable by your current user.

Fix the permissions and export the `kind` kubeconfig again:

```shell
sudo mkdir -p ~/.kube
sudo chown -R "$(id -un)":"$(id -gn)" ~/.kube
chmod 700 ~/.kube
kind export kubeconfig --name kind
```

You can then verify the cluster is reachable with:

```shell
kubectl config current-context
kubectl get nodes
```

#### K3d(RECOMMENDED)

##### [Installation guide](https://k3d.io/stable/#installation)

```shell
wget -q -O - https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash
```

##### Start the Cluster

```shell
k3d cluster create athena
```

You can then verify the cluster is reachable with:

```shell
kubectl config current-context
kubectl get nodes
```

### Verify cluster installation

* Run `kubectl version` 

## Fork and Clone the Repository
1. Fork the Athena repository to your personal GitHub Account
2. Clone the forked repository:
```shell
git clone https://github.com/YOUR-USERNAME/athena.git
```
   Please note that the local build process uses GOPATH and that path should not be used, unless the Athena repository was directly cloned in it.

3. While everyone has their own Git workflow, the author of this document recommends to create a remote called `upstream` in your local copy pointing to the original Athena repository. This way, you can easily keep your local branches up-to-date by merging in latest changes from the Athena repository, i.e. by doing a `git pull upstream master` in your locally checked out branch.
   To create the remote, run:
   ```shell
   cd athena
   git remote add upstream https://github.com/useryege/athena.git
   ```

## Install Additional Required Development Tools

```shell
make install-go-tools-local
make install-codegen-tools-local
```

## Install Latest Athena on Your Local Cluster

```shell
kubectl create namespace athena &&
kubectl apply -n athena --server-side --force-conflicts -f https://raw.githubusercontent.com/useryege/athena/master/manifests/install.yaml
```

Set kubectl config to avoid specifying the namespace in every kubectl command.  

```shell
kubectl config set-context --current --namespace=athena
```

