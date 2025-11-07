# Import Existing Resources

The Talos Operator provides the capability to import and manage existing Talos Linux clusters. This is particularly useful when you have manually set up a Talos cluster and want to bring it under the management of the Talos Operator without recreating the resources.

Talos Operator does allow you to import existing TalosControlPlane and TalosWorker resources using the `talos.alperen.cloud/reconcile-mode: import` annotation in the resource metadata. When this annotation is set, the operator will recognize the existing resources and will try to take the control of them. Once imported, the operator will set the status of the resources to `Imported` and will start managing them as if they were created by the operator itself.

## How to Import Existing Resources

To import existing TalosControlPlane and TalosWorker resources, follow these steps:

1. **Grab your existing TalosControlPlane and TalosWorker configurations.** Ensure you have the correct controlplane and worker configurations that match your existing Talos cluster setup.

2. **Create configMaps for Talos configurations.** If you haven't already, create ConfigMaps in your Kubernetes cluster that contain the Talos configuration files for both control plane and worker nodes. These configMaps will be referenced in the TalosControlPlane and TalosWorker resources further down. 

Here is an example command to create a ConfigMap for the control plane configuration:

```bash
kubectl create configmap talos-test-cluster-controlplane --from-file=controlplane.yaml
```

And for the worker configuration:

```bash
kubectl create configmap talos-test-cluster-worker --from-file=worker.yaml
```

3. **Retrieve the Talos version and Kubernetes version.** Make sure you know the Talos version and Kubernetes version that your existing cluster is running.

You can simply achieve this by running the following commands:

```bash
    talosctl get version | awk '{print $6}' | tail -n +2
```

```bash
    talosctl get staticpods.kubernetes.talos.dev kube-apiserver -o yaml | grep "image:" | cut -d':' -f3
```

4. **Define TalosControlPlane and TalosWorker resources with import annotation.** Create YAML manifests for TalosControlPlane and TalosWorker resources, including the `talos.alperen.cloud/reconcile-mode: import` annotation in the metadata section. Ensure that the specifications match your existing cluster setup.

Here is an example of a TalosControlPlane resource definition for importing:

```yaml
apiVersion: talos.alperen.cloud/v1alpha1
kind: TalosControlPlane
metadata:
  name: talos-controlplane-import-test
  # This annotation indicates that the existing TalosControlPlane should be imported
  annotations:
    talos.alperen.cloud/reconcile-mode: import 
spec:
  version: v1.11.3 # The Talos version you retrieved earlier 
  mode: metal # Deployment mode -- can be 'metal' or 'container'
  kubeVersion: v1.34.1 # The Kubernetes version you retrieved earlier
  configRef: 
    name: talos-test-cluster-controlplane 
    key: controlplane.yaml
  metalSpec:
    # List of Talos machines(IP Addresses) in the control plane 
    machines:
      - "MACHINE_IP_ADDRESS_1"
      - "MACHINE_IP_ADDRESS_2"
  # Kubernetes API server endpoint
  endpoint: "https://API_SERVER_ENDPOINT:6443"
```

!!!warning
    The import functionality currently supports only metal mode.

!!!warning
    Ensure that the Talos versions and Kubernetes versions specified in the resource definitions match those of your existing cluster to avoid compatibility issues.

Once you have created the TalosControlPlane nor TalosWorker resources with the import annotation, the Talos Operator will detect these resources and import them into its management. You can verify the import status by checking the status of the resources using:

```bash
kubectl get taloscontrolplane talos-controlplane-import-test -o json | jq '.status.imported'
```

If the import is successful, the status will show `true`, indicating that the Talos Operator is now managing the existing TalosControlPlane resource. You can follow similar steps for TalosWorker resources.

Importing a resource allows you to still control fields such as version and Kubernetes version. However, be cautious when changing fields that directly affect the existing cluster's configuration, as this may lead to inconsistencies or disruptions in the cluster's operation, for more specific configuration you might need to update the Talos configuration configMaps directly.
