# TalosEtcdBackup

The `TalosEtcdBackup` custom resource enables automated etcd backups for Talos control planes, streaming snapshots directly to S3-compatible storage without local disk usage.

## Overview

TalosEtcdBackup provides a declarative way to backup etcd data from Talos control planes. The operator:

1. Connects to the Talos API to stream an etcd snapshot
2. Uploads the snapshot directly to S3 (or S3-compatible storage)
3. Uses zero local disk I/O for minimal footprint
4. Updates status conditions to track backup progress

## Features

- **Zero Local I/O**: Snapshots stream directly from Talos API to S3
- **Memory Efficient**: Uses streaming to minimize memory footprint
- **S3 Compatible**: Works with AWS S3, MinIO, Ceph, and other S3-compatible storage
- **Secure**: Credentials stored in Kubernetes secrets
- **Status Tracking**: Updates conditions for easy monitoring

## Specification

### Required Fields

- `spec.talosControlPlaneRef`: Reference to the TalosControlPlane to backup
- `spec.backupStorage.s3.bucket`: S3 bucket name
- `spec.backupStorage.s3.region`: AWS region
- `spec.backupStorage.s3.accessKeyID`: Secret reference for access key ID
- `spec.backupStorage.s3.secretAccessKey`: Secret reference for secret access key

### Optional Fields

- `spec.backupStorage.s3.endpoint`: Custom S3 endpoint (for S3-compatible storage)
- `spec.backupStorage.s3.insecureSkipTLSVerify`: Skip TLS verification (default: false)

## Example

### 1. Create S3 Credentials Secret

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: my-s3-credentials
  namespace: default
type: Opaque
stringData:
  accessKeyID: "YOUR_ACCESS_KEY_ID"
  secretAccessKey: "YOUR_SECRET_ACCESS_KEY"
```

### 2. Create TalosEtcdBackup Resource

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

### 3. Check Backup Status

```bash
kubectl get talosetcdbackup my-backup -o yaml
```

Look for the `status.conditions` field to check backup progress:

- `Progressing`: Backup is in progress
- `Ready`: Backup completed successfully
- `Failed`: Backup encountered an error

## Using S3-Compatible Storage

For MinIO, Ceph, or other S3-compatible storage, add the `endpoint` field:

```yaml
spec:
  backupStorage:
    s3:
      bucket: my-backups
      region: us-east-1
      endpoint: https://minio.example.com
      insecureSkipTLSVerify: false  # Set to true for self-signed certs
      accessKeyID:
        name: my-s3-credentials
        key: accessKeyID
      secretAccessKey:
        name: my-s3-credentials
        key: secretAccessKey
```

## Backup Location

Backups are stored in S3 with the following key pattern:

```
talos-operator-etcd-backups/<controlplane-name>/etcd-snapshot-<timestamp>.db
```

Example:
```
talos-operator-etcd-backups/my-controlplane/etcd-snapshot-2025-10-04T11-30-00Z.db
```

## Status Conditions

The operator updates status conditions throughout the backup lifecycle:

| Condition | Status | Reason | Description |
|-----------|--------|--------|-------------|
| Progressing | True | BackupInProgress | Backup is currently running |
| Progressing | False | BackupCompleted | Backup finished |
| Ready | True | BackupSucceeded | Backup completed successfully |
| Failed | True | BackupFailed | Backup encountered an error |

## Best Practices

1. **Credentials Management**: Always use Kubernetes secrets for S3 credentials
2. **Bucket Lifecycle**: Configure S3 lifecycle policies for automatic cleanup
3. **Monitoring**: Watch status conditions for backup health
4. **Testing**: Test restores regularly to ensure backups are valid
5. **Network**: Ensure operator has network access to S3 endpoint

## Troubleshooting

### Backup Fails with "Failed to get TalosControlPlane"

Ensure:
- The referenced TalosControlPlane exists in the same namespace
- The TalosControlPlane is in a ready state with valid bundle config

### Backup Fails with "Failed to get secret"

Ensure:
- The secret exists in the same namespace as the TalosEtcdBackup
- The secret contains the correct keys (`accessKeyID` and `secretAccessKey`)

### Backup Fails with "Failed to upload snapshot to S3"

Check:
- S3 credentials have write permissions to the bucket
- Network connectivity to S3 endpoint
- Bucket exists and is in the correct region
- For custom endpoints, verify the endpoint URL is correct

### Backup Hangs in Progressing State

- Check operator logs for detailed error messages
- Verify Talos API is accessible from the operator
- Ensure sufficient network bandwidth for snapshot transfer

## Performance Considerations

The backup process streams data directly from Talos to S3:

- **Memory Usage**: Minimal, as data is streamed (not buffered)
- **Disk Usage**: Zero, no local storage is used
- **Network**: Bandwidth dependent on snapshot size
- **Duration**: Depends on snapshot size and network speed

For large etcd databases, ensure:
- Adequate network bandwidth between operator and S3
- Sufficient Talos API timeout settings
- Monitoring of backup completion times
