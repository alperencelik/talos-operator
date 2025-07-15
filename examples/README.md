# Examples

Below are some examples to help you get started with the `talos-operator`. Files with "container" in their names can be run in a containerized environment, while those with "metal" are intended for either bare-metal or virtualized environments. 

You can either apply the `TalosCluster` resource, or both `TalosControlPlane` and `TalosWorker` resources together. The `TalosCluster` object encapsulates both `TalosControlPlane` and `TalosWorker`, so you can choose whichever resource(s) best suit your needs. You can also use the `TalosControlPlane` and `TalosWorker` resources independently if you prefer to manage them separately.