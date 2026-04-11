# Try Athena Locally

> [!TIP]
> This guide assumes you have a grounding in the tools that Athena is based on. Please read [understanding the basics](understand_the_basics.md) to learn about these tools.


Follow these steps to install `Kind` for local development and set it up with Athena.

To run an Athena development environment review the [developer guide for running locally](./developer-guide/running-locally.md).

## Install Kind

Install Kind Following Instructions [here](https://kind.sigs.k8s.io/docs/user/quick-start#installation).

##  Create a Kind Cluster
Once Kind is installed, create a new Kubernetes cluster with:
```bash
kind create cluster --name athena-cluster
```
This will create a local Kubernetes cluster named `athena-cluster`.

## Set Up kubectl to Use the Kind Cluster
After creating the cluster, set `kubectl` to use your new `kind` cluster:
```bash
kubectl cluster-info --context kind-athena-cluster
```
This command verifies that `kubectl` is pointed to the right cluster.

## Install Athena on the Cluster
You can now install Athena on your `kind` cluster. First, apply the Athena manifest to create the necessary resources:
```bash
kubectl create namespace athena
kubectl apply -n athena --server-side --force-conflicts -f https://raw.githubusercontent.com/useryege/athena/stable/manifests/install.yaml
```

> [!NOTE]
> The `--server-side --force-conflicts` flags are required because some Athena CRDs exceed the size limit for client-side apply. See the [getting started guide](getting_started.md) for more details.

## Expose Athena API Server
By default, Athena's API server is not exposed outside the cluster. You need to expose it to access the UI locally. For development purposes, you can use Kubectl 'port-forward'.
```bash
kubectl port-forward svc/athena-server -n athena 8080:443
```
This will forward port 8080 on your local machine to the Athena API server’s port 443 inside the Kubernetes cluster.

## Access Athena UI
Now, you can open your browser and navigate to http://localhost:8080 to access the Athena UI.

### Log in to Athena
To log in to the Athena UI, you'll need the default admin password. You can retrieve it from the Kubernetes cluster:
```bash
kubectl -n athena get secret athena-initial-admin-secret -o jsonpath='{.data.password}' | base64 -d
```
Use the admin username and the retrieved password to log in.

You can now move on to step #2 in the [Getting Started Guide](getting_started.md).
