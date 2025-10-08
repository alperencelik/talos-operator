# Examples

Below are some examples to help you get started with the `talos-operator`. Files with "container" in their names can be run in a containerized environment, while those with "metal" are intended for either bare-metal or virtualized environments. 

You can either apply the `TalosCluster` resource, or both `TalosControlPlane` and `TalosWorker` resources together. The `TalosCluster` object encapsulates both `TalosControlPlane` and `TalosWorker`, so you can choose whichever resource(s) best suit your needs. You can also use the `TalosControlPlane` and `TalosWorker` resources independently if you prefer to manage them separately.

## Available Examples

### Cluster Resources
- `talos-cluster-container.yaml` - Container-based Talos cluster
- `talos-cluster-metal.yaml` - Bare-metal/VM-based Talos cluster
- `talos-controlplane-container.yaml` - Container-based control plane
- `talos-controlplane-metal.yaml` - Bare-metal/VM-based control plane
- `talos-worker-container.yaml` - Container-based worker nodes
- `talos-worker-metal.yaml` - Bare-metal/VM-based worker nodes

### Backup Resources
- `talos-etcd-backup.yaml` - One-time etcd backup
- `talos-etcd-backup-schedule.yaml` - Scheduled periodic etcd backups (daily, hourly, weekly examples)