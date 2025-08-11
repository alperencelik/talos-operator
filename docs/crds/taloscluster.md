# TalosCluster

`TalosCluster` is a custom resource definition (CRD) used to define a Talos Linux cluster in a Kubernetes environment. It allows users to specify the configuration and desired state of a Talos cluster, including the number of control plane and worker nodes, networking settings, and other cluster parameters. `TalosCluster` is breaking the cluster into two parts: the `TalosControlPlane` and the `TalosWorker`. You can either define them inline or you can refer to them by name.

## Creating your first TalosCluster

To create your first `TalosCluster` as container mode, you can use the following example YAML manifest:

```yaml
apiVersion: talos.alperen.cloud/v1alpha1
kind: TalosCluster
metadata:
  name: taloscluster-sample
spec:
  controlPlane:
    version: v1.10.3
    mode: container
    replicas: 2
    kubeVersion: v1.33.0
  worker:
    version: v1.10.3
    mode: container 
    replicas: 2
    kubeVersion: v1.33.0 
```

This manifest defines a `TalosCluster` named `taloscluster-sample` with the following specifications:

- **Control Plane**:
  - Version: `v1.10.3` (Talos version)
  - Mode: `container` (running Talos as a container in Kubernetes)
  - Replicas: `2` (two control plane nodes)
  - Kubernetes Version: `v1.33.0`
- **Worker Nodes**:
  - Version: `v1.10.3` (Talos version)
  - Mode: `container` (running Talos as a container in Kubernetes)
  - Replicas: `2` (two worker nodes)
  - Kubernetes Version: `v1.33.0`

!!!tip
    For more examples please refer to [examples](https://github.com/alperencelik/talos-operator/tree/main/examples) directory in the repository.

!!!warning
    Mixing modes is highly discouraged. You should either use `container` mode for both control plane and worker nodes or `metal` mode for both. Mixing modes can lead to unexpected behavior and it's not fully supported. If you want to run Talos on bare metal or virtual machines, you should use the `metal` mode for both control plane and worker nodes.
