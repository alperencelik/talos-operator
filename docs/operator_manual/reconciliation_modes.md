# Reconciliation Mode

Talos Operator is also supports different type of reconciliation modes to manage your Talos Linux resources. You can define the annotation `talos.alperen.cloud/reconcile-mode` in objects to set the reconciliation mode. The values are case-insensitive. The default value is `reconcile` which means the operator will reconcile the Custom Resource object(s) and handle the operations. You can find the all reconciliation modes in the table below.

| Reconciliation Mode | Description |
|---------------------|-------------|
| Reconcile           | The operator will reconcile the object and handle the operations based on the Custom Resource. |
| Disable              | The operator will not reconcile the object and do not handle the operations. This is useful if you would like to debug something or you would like to disable the operator for the specific object for a while. |
| DryRun     | The operator will run the reconciliation but will not perform any mutating operations. Kubernetes writes (child resources, ConfigMaps, Secrets, Jobs, status updates) are sent with server-side dry-run (`dryRun=All`) so they are validated by the API server without being persisted. Talos config applies use Talos' native dry-run, so the change details (diff) reported by the node are logged and surfaced as Kubernetes Events. Operations without dry-run support (bootstrap, upgrade, reset, meta keys, PXE boot stack file operations) and readiness waits are skipped and reported as "Would do X" events instead. Supported on `TalosCluster`, `TalosMachine`, and the **metal mode** of `TalosControlPlane` and `TalosWorker` — container mode is not supported yet and skips reconciliation. |

> **Notes on DryRun mode:**
>
> - No status is persisted on the object, so each reconciliation re-runs the full simulation (about every 5 minutes). Use `kubectl describe <resource> <name>` to see the `DryRun` events describing the operations that would be performed.
> - Finalizer management still runs normally so objects can be created and deleted cleanly. When you delete a DryRun-annotated object, the operator skips its cleanup operations (e.g. Talos reset, explicit child deletion) but the deletion itself proceeds — and Kubernetes owner-reference garbage collection will still cascade to child resources.
