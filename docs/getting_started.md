# Getting Started

This guide will walk you through the process of installing and configuring the talos-operator.

## Installation

You can install the talos-operator using Helm:

```bash
helm repo add talos-operator https://alperencelik.github.io/talos-operator
helm install talos-operator talos-operator/talos-operator
```

## Creating your first Talos Cluster

To create your first Talos cluster using the `talos-operator`, you need to define a `TalosCluster` custom resource. You can find an examples of `TalosCluster` in the `examples/` directory. 

TO be updated

## Features

- **Decoupled Design**: The operator is designed to decouple a Kubernetes cluster in two parts: the control plane and the worker nodes. This allows you to manage the control plane and worker nodes independently, which is useful different purposes. You can create control planes without any worker nodes, and vice versa. You can design many scenarious with this decoupled design, such as control plane as a service, or control plane as a pod in Kubernetes, or even more. To learn more please see the `examples/` directory.
- **Installation Modes**: The operator supports two installation modes: `container` and `metal`. The `container` mode allows you to run Talos clusters inside Kubernetes pods(like Kubernetes-in-Kubernetes), while the `metal` mode allows you to run Talos clusters on bare metal or virtual machines. This gives you flexibility in how you want to deploy and manage your Talos clusters. Metal mode requires you to have machines already booted with Talos OS(in maintenance mode) and the operator will configure those machines to cluster based on the configuration you provide in the CRDs.
- **Ease of Usage && Flexibility**: The operator is designed to be easy to use and flexible. You can create Talos clusters using simple Kubernetes Custom Resource Definitions (CRDs) and the operator will take care of the rest. Operator will generate all those configurations and write them in to ConfigMaps and Secrets to provide a way to users to integrate. Even though the operator is managing the all config generations and operations, you can still provide the configuration you want to use in the CRDs. This allows you to customize the Talos clusters according to your needs and make them work with your specific requirements.

- **Integratibility**: The operator is designed to be easily integratable with other Kubernetes operators and tools. Since the operator generates a Kubeconfig for the created Talos clusters, you can use that data to feed into other tools or operators such as ArgoCD, FluxCD or any other custom Kubernetes invocation. This allows you to use the operator in your existing Kubernetes workflows and tools.