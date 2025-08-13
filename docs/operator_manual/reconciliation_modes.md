# Reconciliation Mode

Talos Operator is also supports different type of reconciliation modes to manage your Talos Linux resources. You can define the annotation `talos.alperen.cloud/reconciliation-mode` in objects to set the reconciliation mode. The default value is `Reconcile` which means the operator will reconcile the Custom Resource object(s) and handle the operations. You can find the all reconciliation modes in the table below.

| Reconciliation Mode | Description |
|---------------------|-------------|
| Reconcile           | The operator will reconcile the object and handle the operations based on the Custom Resource. |
| Disable              | The operator will not reconcile the object and do not handle the operations. This is useful if you would like to debug something or you would like to disable the operator for the specific object for a while. |
| DryRun (TODO)     | The operator will not apply the changes to the object but will log the operations that would be performed. This is useful for testing purposes to see what would happen without actually making changes. |
