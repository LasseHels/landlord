# Landlord
Landlord simulates eviction storms in an AKS cluster.

## Motivation

Landlord was built for an incident wargame in which spot instances are continuously evicted.

Do not run Landlord in production.

## Authentication

In order to run, Landlord needs access to:
* A Kubernetes cluster.
* The Azure API.

### Kubernetes

Landlord supports only [out-of-cluster](https://github.com/kubernetes/client-go/tree/master/examples/out-of-cluster-client-configuration) Kubernetes authentication.

When using out-of-cluster authentication, Landlord will look for a kubeconfig file in the `$HOME/.kube` directory.
Other kubeconfig files can be specified by setting the `KUBECONFIG` environment variable.
See [kubeconfig documentation](https://kubernetes.io/docs/concepts/configuration/organize-cluster-access-kubeconfig/).

### Azure

Landlord will authenticate to Azure using local CLI credentials.
See https://github.com/hashicorp/go-azure-sdk/blob/main/sdk/auth/README.md#example-authenticating-using-the-azure-cli.
