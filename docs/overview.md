# Overview

## What is talos-operator?

**talos-operator** is a Kubernetes operator for managing Talos clusters. It can help you to bootstrap/create Talos clusters in different environments.

## Why use talos-operator?

Talos Linux is great choice for running Kubernetes clusters due to it's minimalistic design and being an API-driven OS. However, creating and managing the lifecycle of Talos clusters can be challenging because it's only available to Talos CLI. Maintaiing the artifacts, secrets and configuration files for Talos clusters can be cumbersome. talos-operator aims to solve these problems by providing a Kubernetes-native way to manage Talos clusters. For further information, please refer to the [motivation](##motivation) section of the documentation.

## Motivation

As a person who is against CLI tools to install clusters, I wanted to create a way to manage Talos clusters declaratively using Kubernetes operators. The problem with the managing cluster lifecycles within a CLI tool has some quite challenges such as

- Managing the secrets: Talos API requires mTLS secrets to be passed in order to interact with the Talos cluster. You have to manage these secrets manually and ensure they are properly configured for each Talos cluster. This can be error-prone and time-consuming.
- Managing the configuration files: Talos clusters require various configuration files to be created and managed. This includes controlplane configuration, worker node configuration, and or other patch configurations. Managing these files and versioning them can be difficult, especially when you have multiple Talos clusters with different configurations.
- Managing the lifecycle of Talos clusters: You need to follow some specific steps to create, update, and delete Talos clusters with some specific verifications.

The **talos-operator** allows you to define your Talos cluster configuration in Kubernetes Custom Resource Definitions (CRDs) and manage the lifecycle of Talos clusters using Kubernetes controllers. Since we offload state management part to the Kubernetes we don't need to worry about Talosconfigs, secret bundles or any-other operation that needs to be done via Talos CLI. The operator takes care all of those and you don't need to run any Talos CLI commands manually.
