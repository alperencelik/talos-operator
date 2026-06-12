# Custom Resource Definitions (CRDs)

The Talos Operator defines the following CRDs in the `talos.alperen.cloud/v1alpha1` API group. All resources are namespaced.

## Core Resources

| CRD | Short Name | Description |
|-----|-----------|-------------|
| [TalosCluster](./taloscluster.md) | `tc` | Top-level resource that composes a Talos cluster from a control plane and worker nodes. |
| [TalosControlPlane](./taloscontrolplane.md) | `tcp` | Defines and manages the control plane of a Talos cluster. |
| [TalosWorker](./talosworker.md) | `tw` | Defines and manages the worker nodes of a Talos cluster. |
| [TalosMachine](./talosmachine.md) | `tm` | Represents a single Talos machine. Auto-managed by the operator in `metal` mode. |

## Backup Resources

| CRD | Short Name | Description |
|-----|-----------|-------------|
| [TalosEtcdBackup](./talosetcdbackup.md) | `teb` | One-time etcd backup streamed to S3-compatible storage. |
| [TalosEtcdBackupSchedule](./talosetcdbackupschedule.md) | `tebs` | Cron-based scheduled etcd backups with retention management. |

## Addon Resources

| CRD | Short Name | Description |
|-----|-----------|-------------|
| [TalosClusterAddon](./talosclusteraddon.md) | `tca` | Helm chart addon applied to clusters matching a label selector. |
| [TalosClusterAddonRelease](./talosclusteraddonrelease.md) | `tcar` | Per-cluster Helm release instance (auto-managed by `TalosClusterAddon`). |

## Resource Relationships

```
TalosCluster
 ├── TalosControlPlane (inline or ref)
 │    ├── TalosMachine (metal mode, auto-created)
 │    ├── TalosEtcdBackupSchedule
 │    │    └── TalosEtcdBackup (auto-created per schedule)
 │    └── TalosEtcdBackup (manual)
 ├── TalosWorker (inline or ref)
 │    └── TalosMachine (metal mode, auto-created)
 └── TalosClusterAddon
      └── TalosClusterAddonRelease (auto-created per matched cluster)
```
