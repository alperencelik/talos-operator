# TalosEtcdBackup

| Field | Value |
|-------|-------|
| **API Group** | `talos.alperen.cloud` |
| **API Version** | `v1alpha1` |
| **Kind** | `TalosEtcdBackup` |
| **Short Names** | `teb` |
| **Scope** | Namespaced |
| **Subresources** | `status` |

`TalosEtcdBackup` creates a one-time etcd backup from a Talos control plane, streaming the snapshot directly to S3-compatible storage with zero local disk I/O.

## Print Columns

| Name | JSON Path |
|------|-----------|
| Filename | `.status.filename` |
| Age | `.metadata.creationTimestamp` |

---

## Example

```yaml
apiVersion: talos.alperen.cloud/v1alpha1
kind: TalosEtcdBackup
metadata:
  name: my-backup
spec:
  talosControlPlaneRef:
    name: my-controlplane
  backupStorage:
    s3:
      bucket: my-etcd-backups
      region: us-west-2
      accessKeyID:
        name: my-s3-credentials
        key: accessKeyID
      secretAccessKey:
        name: my-s3-credentials
        key: secretAccessKey
```

### With Custom S3 Endpoint (MinIO, Ceph, etc.)

```yaml
spec:
  backupStorage:
    s3:
      bucket: my-backups
      region: us-east-1
      endpoint: https://minio.example.com
      insecureSkipTLSVerify: false
      accessKeyID:
        name: my-s3-credentials
        key: accessKeyID
      secretAccessKey:
        name: my-s3-credentials
        key: secretAccessKey
```

---

## Spec Fields

### `spec` (TalosEtcdBackupSpec)

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `talosControlPlaneRef` | [LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#localobjectreference-v1-core) | Yes | - | Reference to the `TalosControlPlane` to back up (by name). |
| `backupStorage` | [BackupStorage](#backupstorage) | Yes | - | Storage configuration for the backup. |

### BackupStorage

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `s3` | *[S3Storage](#s3storage) | No | - | S3-compatible storage configuration. |

### S3Storage

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `bucket` | string | Yes | - | S3 bucket name. |
| `region` | string | Yes | - | AWS region where the bucket is located. |
| `endpoint` | string | No | - | Custom S3 endpoint URL for S3-compatible services (MinIO, Ceph, etc.). |
| `accessKeyID` | [SecretKeySelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#secretkeyselector-v1-core) | Yes | - | Reference to a Secret key containing the S3 access key ID. |
| `secretAccessKey` | [SecretKeySelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#secretkeyselector-v1-core) | Yes | - | Reference to a Secret key containing the S3 secret access key. |
| `insecureSkipTLSVerify` | bool | No | `false` | Skip TLS certificate verification for the S3 endpoint. |

---

## Status Fields

### `status` (TalosEtcdBackupStatus)

| Field | Type | Description |
|-------|------|-------------|
| `filename` | string | Name of the etcd snapshot file in the storage backend. Pattern: `talos-operator-etcd-backups/<controlplane-name>/etcd-snapshot-<timestamp>.db` |
| `stateFilename` | string | Name of the paired state-secret backup file in the storage backend. |
| `conditions` | [][Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) | List of conditions. Map-list keyed by `type`. |

#### Condition Types

| Type | Status | Reason | Description |
|------|--------|--------|-------------|
| `Progressing` | `True` | `BackupInProgress` | Backup is currently running. |
| `Progressing` | `False` | `BackupCompleted` | Backup finished. |
| `Ready` | `True` | `BackupSucceeded` | Backup completed successfully. |
| `Failed` | `True` | `BackupFailed` | Backup encountered an error. |
