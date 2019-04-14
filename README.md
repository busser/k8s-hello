# Kubernetes: Hello, World!

A simple HTTP server meant to illustrate basic Kubernetes functionnalities to
developers.

## Usage

Build the container image, push it to a registry, and deploy the server
to your Kubernetes cluster:

```bash
docker build --tag=busser/k8s-hello .
docker push busser/k8s-hello
kubectl apply --filename=k8s-hello.yaml
```

The deployment defined in `k8s-hello.yaml` requires resources as specified by
the resource quota in `compute-resources.yaml`.

The `labwork.md` file will walk you through how the `k8s-hello.yaml`file was
written.
