# State Secret

For each `TalosControlPlane` the operator creates and maintains a Secret named `{name}-state` in the same namespace. This Secret holds the durable runtime state the operator needs to keep managing an existing cluster across CR re-applies, operator restarts, and management-cluster moves.

## What it contains

- `secretBundle` — the Talos PKI used to sign and validate machine configs
- `bundleConfig` — the bundle parameters the operator generated for this cluster
- `state` — the most recent reconciliation state of the control plane
- `config` — the last applied control plane configuration
- `observedKubeVersion` — the Kubernetes version the operator last reconciled

## Why it matters

`kubectl apply` does not write the `status` subresource — applying a CR always lands with empty `.status`. Without a side artifact, the operator would generate a fresh PKI on the next reconcile, which would not match the PKI baked into the already-running Talos nodes. The cluster would become unmanageable.

The state secret breaks this dependency on `.status`:

1. The operator writes critical state into the Secret on every status update.
2. On reconcile, if `.status.secretBundle` is empty but the Secret exists, the operator restores `.status` from it and continues from where it left off.
3. The Secret has no owner reference and is not deleted when the CR is deleted, so deleting and re-applying a `TalosControlPlane` recovers the original state automatically.

## Backing it up

Because the operator runs in a Kubernetes cluster that may itself be ephemeral, you should back the state secret up to durable storage outside that cluster. The `TalosEtcdBackup` controller does this automatically — every etcd backup uploads the matching state secret to the same S3 bucket alongside the etcd snapshot. You can list both objects under the `talos-operator-etcd-backups/{cluster-name}/` prefix.

If you are not using `TalosEtcdBackup`, export the secret manually and store it somewhere safe:

```bash
kubectl get secret <tcp-name>-state -o yaml > <tcp-name>-state.yaml
```

## Restoring from a state secret

Whether you are recovering from a lost CR, a wiped management cluster, or moving the operator to a different cluster:

1. Apply the state secret first:

   ```bash
   kubectl apply -f <tcp-name>-state.yaml
   ```

2. Apply the `TalosControlPlane` (and any `TalosMachine`/`TalosWorker`) CRs:

   ```bash
   kubectl apply -f taloscontrolplane.yaml
   ```

The operator picks up the existing Secret, restores `.status` from it on the first reconcile, and continues managing the cluster — no re-provisioning, no PKI mismatch.

!!!note
    `TalosMachine` does not have a state secret. On reconcile the controller probes the node over a secure (mTLS) connection: if the node responds, the machine state is restored to `Available` automatically. This works because the state restored on the parent `TalosControlPlane` already provides the PKI needed for that probe.

!!!warning
    The state secret contains the Talos PKI — treat it like any other root credential. Anyone with this Secret can issue commands to the cluster's Talos API.

## Lifecycle

The state secret has no owner reference and the operator never deletes it — not when the `TalosControlPlane` is deleted, and not when `spec.deletionPolicy: reset` wipes the underlying nodes. This is intentional guard rail to
avoid accidentally losing the state needed to manage an existing cluster. If the operator deleted the Secret, a mistaken `kubectl delete taloscontrolplane` would orphan the nodes and lose the PKI, making recovery much more difficult.

If you genuinely want a clean slate (for example, you have decommissioned the cluster and want to remove all leftover state), delete the Secret yourself:

```bash
kubectl delete secret <tcp-name>-state
```
