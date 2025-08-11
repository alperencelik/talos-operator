# TalosWorker

`TalosWorker` is a Kubernetes custom resource definition (CRD) that represents worker nodes in a Talos cluster. It is used to manage the lifecycle of worker nodes. The `TalosWorker` resource allows users to define the desired state of worker nodes, including their configuration, networking settings, and other parameters. It is typically used in conjunction with the `TalosControlPlane` resource to manage the entire Talos cluster. Unlike the `TalosControlPlane`, which is responsible for the control plane nodes, you can't define `TalosWorker` resource indiviually, because it requires a reference to a valid `TalosControlPlane` object. I don't find any use case where you would want to define `TalosWorker` resource without a `TalosControlPlane` object, so it's not allowed.

!!!tip
    For more examples please refer to [examples](https://github.com/alperencelik/talos-operator/tree/main/examples) directory in the repository.
