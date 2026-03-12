---
template: home.html
hide:
  - toc
---

**talos-operator** is a Kubernetes operator for managing the full lifecycle of [Talos Linux](https://www.talos.dev/) clusters using native Custom Resources. It replaces manual `talosctl` workflows with a declarative, controller-driven approach — secrets, configs, upgrades, and backups all reconciled continuously.

---

## Features

- **Declarative cluster management** — define control planes, workers, and cluster topology as CRDs
- **Automatic secret management** — mTLS bundles, Talos secrets, and kubeconfigs stored as Kubernetes Secrets
- **Metal & container modes** — run on bare metal machines in maintenance mode or as pods inside an existing cluster
- **Etcd backup & restore** — scheduled snapshots with S3 storage via `TalosEtcdBackup` and `TalosEtcdBackupSchedule`
- **Helm addon management** — deploy and lifecycle-manage Helm charts into Talos clusters
- **Declarative upgrades** — upgrade Talos OS and Kubernetes versions across control plane and worker nodes

---

## Motivation

Talos Linux is a minimal, API-driven OS purpose-built for Kubernetes. It strips away everything unnecessary — no SSH, no shell, no package manager — leaving a secure and immutable system controlled entirely through an API.

But operating Talos at scale still requires significant manual effort. Generating machine configs, bootstrapping nodes, rotating secrets, and managing upgrades across a fleet of clusters means running sequences of `talosctl` commands, maintaining shell scripts, and tracking state outside of Kubernetes.

This creates a gap: Talos gives you a great foundation, but the operational layer on top of it is left to you.

---

## Why talos-operator?

**talos-operator** closes that gap by bringing Talos cluster lifecycle management into Kubernetes itself. Instead of imperative CLI workflows, you describe the desired state of your clusters as Custom Resources — and the operator continuously reconciles reality to match.

- **No more `talosctl` for day-to-day operations.** Bootstrapping, secret generation, and upgrades are handled by the controller.
- **GitOps-native.** Your cluster definitions live in Git alongside everything else, and changes are applied through the standard Kubernetes reconciliation loop.
- **Works with your existing tooling.** The operator generates kubeconfigs as Kubernetes Secrets, making it straightforward to integrate with ArgoCD, FluxCD, or any other tool that consumes kubeconfigs.
- **Flexible deployment models.** Run Talos clusters on bare metal in maintenance mode, or spin them up as pods inside an existing cluster for testing and development.

---

## Next Steps

- [Getting Started](getting_started.md) — installation and first cluster walkthrough
- [CRD Reference](crds/index.md) — full reference for all custom resources
- [Platform Modes](operator_manual/modes.md) — metal vs container mode explained
- [Upgrade Guide](operator_manual/upgrade_versions.md) — upgrading Talos OS and Kubernetes versions
