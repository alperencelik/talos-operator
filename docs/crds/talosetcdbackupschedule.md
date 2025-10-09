# TalosEtcdBackupSchedule

The `TalosEtcdBackupSchedule` custom resource enables automated, periodic etcd backups for Talos control planes on a schedule. It creates `TalosEtcdBackup` resources based on a cron schedule and manages retention of old backups.

## Overview

`TalosEtcdBackupSchedule` automates the process of creating regular etcd backups by:
- Creating backups on a cron schedule
- Managing backup retention (automatically deleting old backups)
- Providing status information about the schedule
- Supporting pause/resume functionality

## Features

- **Cron-based Scheduling**: Define backup schedules using standard cron expressions
- **Automatic Backup Creation**: Creates `TalosEtcdBackup` resources automatically based on schedule
- **Retention Management**: Automatically removes old backups based on retention policy
- **Pause/Resume**: Ability to pause and resume backup schedules
- **Status Tracking**: View last backup time, next scheduled backup, and active backups

## Specification

### Required Fields

- `spec.schedule`: Cron expression defining when to run backups (e.g., "0 2 * * *" for daily at 2 AM)
- `spec.backupTemplate.spec.talosControlPlaneRef`: Reference to the TalosControlPlane to backup
- `spec.backupTemplate.spec.backupStorage.s3.bucket`: S3 bucket name
- `spec.backupTemplate.spec.backupStorage.s3.region`: AWS region
- `spec.backupTemplate.spec.backupStorage.s3.accessKeyID`: Secret reference for access key ID
- `spec.backupTemplate.spec.backupStorage.s3.secretAccessKey`: Secret reference for secret access key

### Optional Fields

- `spec.retention`: Number of successful backups to keep (default: 5)
- `spec.paused`: Set to true to pause the backup schedule (default: false)
- `spec.backupTemplate.spec.backupStorage.s3.endpoint`: Custom S3 endpoint
- `spec.backupTemplate.spec.backupStorage.s3.insecureSkipTLSVerify`: Skip TLS verification (default: false)

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

### 2. Create TalosEtcdBackupSchedule Resource

```yaml
apiVersion: talos.alperen.cloud/v1alpha1
kind: TalosEtcdBackupSchedule
metadata:
  name: daily-backup-schedule
spec:
  # Daily backups at 2 AM
  schedule: "0 2 * * *"
  
  # Keep 7 daily backups
  retention: 7
  
  backupTemplate:
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

### 3. Check Schedule Status

```bash
kubectl get talosetcdbackupschedule daily-backup-schedule -o yaml
```

Look for the `status` fields:
- `lastScheduleTime`: When the last backup was created
- `lastSuccessfulBackupTime`: When the last backup completed successfully
- `nextScheduleTime`: When the next backup will be created
- `activeBackups`: List of currently running backups

## Cron Schedule Examples

| Schedule | Description |
|----------|-------------|
| `0 2 * * *` | Daily at 2:00 AM |
| `0 */6 * * *` | Every 6 hours |
| `0 0 * * 0` | Weekly on Sunday at midnight |
| `0 0 1 * *` | Monthly on the 1st at midnight |
| `*/15 * * * *` | Every 15 minutes |

For more cron syntax help, see [crontab.guru](https://crontab.guru/).

## Pausing and Resuming Schedules

To pause a backup schedule:

```bash
kubectl patch talosetcdbackupschedule daily-backup-schedule --type merge -p '{"spec":{"paused":true}}'
```

To resume:

```bash
kubectl patch talosetcdbackupschedule daily-backup-schedule --type merge -p '{"spec":{"paused":false}}'
```

## Retention Policy

The `retention` field specifies how many successful backups to keep. When this number is exceeded, the oldest backups are automatically deleted. This applies only to successful backups; failed backups are not counted against the retention limit.

For example, with `retention: 5`:
- The 5 most recent successful backups are kept
- Older successful backups are automatically deleted
- Failed or in-progress backups are not deleted automatically

## Backup Naming

Backups created by the schedule are automatically named using the pattern:
```
<schedule-name>-<timestamp>
```

For example: `daily-backup-schedule-1704153600`

## Status Conditions

The schedule maintains status conditions:

- `Ready`: Schedule is active and functioning normally
- `Failed`: Schedule encountered an error (e.g., invalid cron expression)

## Best Practices

1. **Schedule During Low Traffic**: Schedule backups during periods of lower cluster activity to minimize impact
2. **Set Appropriate Retention**: Balance storage costs with recovery point objectives (RPO)
3. **Monitor Backup Status**: Regularly check that backups are completing successfully
4. **Test Restores**: Periodically test restoring from backups to ensure they are valid
5. **Use Separate Schedules**: Consider separate schedules for different retention policies (e.g., hourly, daily, weekly)

## Troubleshooting

### Schedule Not Creating Backups

1. Check if the schedule is paused:
   ```bash
   kubectl get talosetcdbackupschedule <name> -o jsonpath='{.spec.paused}'
   ```

2. Verify the cron expression is valid:
   ```bash
   kubectl get talosetcdbackupschedule <name> -o jsonpath='{.status.conditions}'
   ```

3. Check controller logs:
   ```bash
   kubectl logs -n talos-operator-system deployment/talos-operator-controller-manager
   ```

### Backups Not Being Deleted

Check the retention policy:
```bash
kubectl get talosetcdbackupschedule <name> -o jsonpath='{.spec.retention}'
```

Only successful backups count toward retention. Failed or in-progress backups are not automatically cleaned up.

### Invalid Cron Expression

If you see a `Failed` condition with reason `InvalidSchedule`, verify your cron expression:
- Use standard cron format: `minute hour day month weekday`
- Test your expression at [crontab.guru](https://crontab.guru/)

## Relationship with TalosEtcdBackup

`TalosEtcdBackupSchedule` creates `TalosEtcdBackup` resources automatically. You can view all backups created by a schedule:

```bash
kubectl get talosetcdbackup -l talos.alperen.cloud/backup-schedule=<schedule-name>
```

Each backup is owned by the schedule, so deleting the schedule will also delete all its backups.

## Using S3-Compatible Storage

The schedule supports any S3-compatible storage service (MinIO, Ceph, etc.):

```yaml
spec:
  backupTemplate:
    spec:
      backupStorage:
        s3:
          bucket: etcd-backups
          region: us-east-1
          endpoint: https://minio.example.com
          insecureSkipTLSVerify: false
          accessKeyID:
            name: minio-credentials
            key: accessKeyID
          secretAccessKey:
            name: minio-credentials
            key: secretAccessKey
```
