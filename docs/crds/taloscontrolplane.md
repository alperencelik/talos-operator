# TalosControlPlane

`TalosControlPlane` is a custom resource definition (CRD) used to define the control plane of a Talos Linux cluster in a Kubernetes environment. It allows users to specify the configuration and desired state of the control plane, including the number of control plane nodes, networking settings, and other parameters specific to the control plane. `TalosControlPlane` might be owned by a `TalosCluster` resource, which manages the overall cluster configuration but also can be defined independently to allow users flexibility in their deployment methods. The `TalosControlPlane` CRD is essential for managing the control plane nodes, ensuring they are correctly configured and maintained according to the specified parameters. It's also place that Talos API secrets are stored, which are used to authenticate and manage the operations via Talos API.

## Modes

The `TalosControlPlane` can be defined in two modes:

- **Container**: This mode allows the control plane to run as a containers within a Kubernetes pod. Thankfully, Talos is compatible to run as Kubernetes in Kubernetes mode so you can run TalosControlPlane as a pod in a Kubernetes cluster. 

- **Metal**: This mode allows the control plane to run directly on bare metal or virtual machines. In this mode, the TalosControlPlane is responsible for managing the lifecycle of the control plane nodes, including installation, configuration, and updates. This is useful for scenarios where you want to run Talos on dedicated hardware or virtual machines outside of Kubernetes.

!!!tip
    For more examples please refer to [examples](https://github.com/alperencelik/talos-operator/tree/main/examples) directory in the repository.
