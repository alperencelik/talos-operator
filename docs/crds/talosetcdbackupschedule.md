# TalosEtcdBackupSchedule

| Field | Value |
|-------|-------|
| **API Group** | `talos.alperen.cloud` |
| **API Version** | `v1alpha1` |
| **Kind** | `TalosEtcdBackupSchedule` |
| **Short Names** | `tebs` |
| **Scope** | Namespaced |
| **Subresources** | `status` |

`TalosEtcdBackupSchedule` creates `TalosEtcdBackup` resources on a cron schedule and manages retention of old backups.

## Print Columns

| Name | JSON Path |
|------|-----------|
| Schedule | `.spec.schedule` |
| Last Backup | `.status.lastSuccessfulBackupTime` |
| Next Backup | `.status.nextScheduleTime` |
| Paused | `.spec.paused` |

---

## Example

```yaml
apiVersion: talos.alperen.cloud/v1alpha1
kind: TalosEtcdBackupSchedule
metadata:
  name: daily-backup
spec:
  schedule: "0 2 * * *"
  retention: 7
  paused: false
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

---

## Spec Fields

### `spec` (TalosEtcdBackupScheduleSpec)

| Field | Type | Required | Default | Validation | Description |
|-------|------|----------|---------|------------|-------------|
| `schedule` | string | Yes | - | MinLength: 1 | Cron expression defining when to trigger backups. e.g. `"0 2 * * *"` for daily at 2 AM. |
| `backupTemplate` | [TalosEtcdBackupTemplateSpec](#talosetcdbackuptemplatespec) | Yes | - | - | Template for the `TalosEtcdBackup` resources created by this schedule. |
| `retention` | *int32 | No | `5` | Minimum: 1 | Number of successful backups to keep. Older backups are automatically deleted. |
| `paused` | bool | No | `false` | - | Pause the schedule. No new backups will be created while `true`. |

### TalosEtcdBackupTemplateSpec

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `spec` | [TalosEtcdBackupSpec](./talosetcdbackup.md#spec-fields) | Yes | - | The full `TalosEtcdBackupSpec` used as the template for each created backup. |

### Cron Schedule Examples

| Expression | Description |
|------------|-------------|
| `0 2 * * *` | Daily at 2:00 AM |
| `0 */6 * * *` | Every 6 hours |
| `0 0 * * 0` | Weekly on Sunday at midnight |
| `0 0 1 * *` | Monthly on the 1st at midnight |
| `*/15 * * * *` | Every 15 minutes |

---

## Status Fields

### `status` (TalosEtcdBackupScheduleStatus)

| Field | Type | Description |
|-------|------|-------------|
| `lastScheduleTime` | *[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#time-v1-meta) | Timestamp of the last time a backup was scheduled. |
| `lastSuccessfulBackupTime` | *[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#time-v1-meta) | Timestamp of the last successful backup completion. |
| `nextScheduleTime` | *[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#time-v1-meta) | Timestamp of the next scheduled backup. |
| `activeBackups` | []string | Names of currently active (in-progress) backup resources. |
| `conditions` | [][Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) | List of conditions. Map-list keyed by `type`. |

#### Condition Types

| Type | Status | Reason | Description |
|------|--------|--------|-------------|
| `Ready` | `True` | - | Schedule is active and functioning normally. |
| `Failed` | `True` | `InvalidSchedule` | Invalid cron expression or other configuration error. |
