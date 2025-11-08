# talos-operator

A Kubernetes operator to manage Talos Linux–based clusters declaratively.

<div style="text-align:center;">
  <img src="docs/images/logo.png" alt="Logo" width="250" height="250">
</div>

## Description

`talos-operator` enables to bootstrap Talos Kubernetes clusters using custom controllers. It allows you to create and manage Talos clusters in different environments, such as bare metal, virtual machines or even in Kubernetes-in-Kubernetes method by using Talos' container support.

## Documentation

Documentation is also available at [https://alperencelik.github.io/talos-operator/](https://alperencelik.github.io/talos-operator/).

## Motivation

Talos Linux is a great choice for running Kubernetes clusters due to its security, simplicity, and API driven design. However, as a person who is against CLI tools to install clusters, I wanted to create a way to manage Talos clusters declaratively using Kubernetes operators. This operator allows you to define your Talos cluster configuration in Kubernetes Custom Resource Definitions (CRDs) and manage the lifecycle of Talos clusters using Kubernetes controllers. You don't need to worry about Talosconfigs, secret bundles or any-other operation that needs to be done via Talos CLI. The operator takes care all of those and you don't need to run any Talos CLI commands manually.

## Features

- **Decoupled Design**: The operator is designed to decouple a Kubernetes cluster in two parts: the control plane and the worker nodes. This allows you to manage the control plane and worker nodes independently, which is useful different purposes. You can create control planes without any worker nodes, and vice versa. You can design many scenarious with this decoupled design, such as control plane as a service, or control plane as a pod in Kubernetes, or even more. To learn more please see the `examples/` directory.
- **Installation Modes**: The operator supports two installation modes: `container` and `metal`. The `container` mode allows you to run Talos clusters inside Kubernetes pods(like Kubernetes-in-Kubernetes), while the `metal` mode allows you to run Talos clusters on bare metal or virtual machines. This gives you flexibility in how you want to deploy and manage your Talos clusters. Metal mode requires you to have machines already booted with Talos OS(in maintenance mode) and the operator will configure those machines to cluster based on the configuration you provide in the CRDs.
- **Ease of Usage && Flexibility**: The operator is designed to be easy to use and flexible. You can create Talos clusters using simple Kubernetes Custom Resource Definitions (CRDs) and the operator will take care of the rest. Operator will generate all those configurations and write them in to ConfigMaps and Secrets to provide a way to users to integrate. Even though the operator is managing the all config generations and operations, you can still provide the configuration you want to use in the CRDs. This allows you to customize the Talos clusters according to your needs and make them work with your specific requirements.

- **Integratibility**: The operator is designed to be easily integratable with other Kubernetes operators and tools. Since the operator generates a Kubeconfig for the created Talos clusters, you can use that data to feed into other tools or operators such as ArgoCD, FluxCD or any other custom Kubernetes invocation. This allows you to use the operator in your existing Kubernetes workflows and tools.

- **Observability & Metrics**: The operator exposes comprehensive Prometheus metrics for monitoring reconciliation operations, cluster health, etcd backups, and Talos API calls. Pre-built Grafana dashboards are included for easy visualization. See [metrics documentation](docs/metrics.md) for details.

## Getting Started

You can install the `talos-operator` in your Kubernetes cluster using the helm chart under `deploy/talos-operator` directory.

```bash
cd deploy/talos-operator
helm install talos-operator ./ --namespace talos-operator
```
Alternatively, you can use the `make` commands to build and deploy the operator as described below.

## Creating your own First Talos Cluster

To create your first Talos cluster using the `talos-operator`, you need to define a `TalosCluster` custom resource. You can find an examples of `TalosCluster` in the `examples/` directory. 

### Prerequisites
- go version v1.24.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

### To Deploy on the cluster

**Run against the current cluster:**

```sh
make run
```

**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=<some-registry>/talos-operator:tag
```

**NOTE:** This image ought to be published in the personal registry you specified.
And it is required to have access to pull the image from the working environment.
Make sure you have the proper permission to the registry if the above commands don’t work.

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/talos-operator:tag
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

## Project Distribution

Following are the steps to build the installer and distribute this project to users.

1. Build the installer for the image built and published in the registry:

```sh
make build-installer IMG=<some-registry>/talos-operator:tag
```

NOTE: The makefile target mentioned above generates an 'install.yaml'
file in the dist directory. This file contains all the resources built
with Kustomize, which are necessary to install this project without
its dependencies.

2. Using the installer

Users can just run kubectl apply -f <URL for YAML BUNDLE> to install the project, i.e.:

```sh
kubectl apply -f https://raw.githubusercontent.com/<org>/talos-operator/<tag or branch>/dist/install.yaml
```

## Contributing
// TODO(user): Add detailed information on how you would like others to contribute to this project

**NOTE:** Run `make help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

