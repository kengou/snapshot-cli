# snapshot-cli Helm Chart

A Helm chart for deploying the snapshot-cli tool as CronJobs in Kubernetes.

## Installation

To install the chart from the OCI registry with the release name `snapshot-cli`:

```bash
helm install snapshot-cli oci://ghcr.io/kengou/snapshot-cli/charts/snapshot-cli --version 0.2.0
```

## Credentials

The CronJobs always read the `OS_*` environment variables from a Kubernetes
Secret (via `envFrom`), so credentials never appear inline in the rendered
manifests. Either:

- bring your own Secret (recommended — e.g. managed by external-secrets or
  sealed-secrets) and set `openstack.secretName` to its name, or
- set `openstack.createSecret=true` and provide the values under
  `openstack.credentials` to have the chart create the Secret. The chart
  refuses to render if `createSecret` is true but `credentials` is empty.

## CronJobs

- **cleanup** runs by default (`0 0 * * *`) and deletes snapshots older than
  `--older-than`.
- **create** is disabled by default because its arguments are
  installation-specific (share ID, name, description). Set `create.schedule`
  AND replace the example placeholders in `create.args` to enable it — the
  chart refuses to render placeholder args.

## Configuration

The following table lists the configurable parameters of the snapshot-cli chart and their default values.

| Parameter | Description | Default |
| --- | --- | --- |
| `image.repository` | The image repository to pull from. | `ghcr.io/kengou/snapshot-cli` |
| `image.pullPolicy` | The image pull policy. | `IfNotPresent` |
| `image.tag` | The image tag to use. `main` is mutable — pin a release tag for production. | `main` |
| `imagePullSecrets` | A list of secrets to use for pulling images. | `[]` |
| `nameOverride` | A string to override the chart name. | `""` |
| `fullnameOverride` | A string to override the fully qualified app name. | `""` |
| `securityContext` | The container security context. | non-root, read-only rootfs, all capabilities dropped |
| `resources` | The resources to allocate for the pod. | `{}` |
| `timeZone` | Time zone for both CronJob schedules. | `"Etc/UTC"` |
| `cronJob.concurrencyPolicy` | Whether overlapping runs are allowed. | `Forbid` |
| `cronJob.startingDeadlineSeconds` | Skip a run that could not start within this window. | `300` |
| `cronJob.successfulJobsHistoryLimit` | Finished jobs kept for inspection. | `3` |
| `cronJob.failedJobsHistoryLimit` | Failed jobs kept for inspection. | `3` |
| `cronJob.backoffLimit` | Retries per scheduled run. | `3` |
| `cleanup.schedule` | The schedule for the cleanup cronjob (empty disables it). | `"0 0 * * *"` |
| `cleanup.args` | The arguments for the cleanup cronjob. | `["cleanup", "--share", "--older-than", "168h"]` |
| `create.schedule` | The schedule for the create cronjob (empty disables it). | `""` (disabled) |
| `create.args` | The arguments for the create cronjob. Replace the placeholders before enabling. | see `values.yaml` |
| `openstack.secretName` | The name of the Secret providing the `OS_*` variables. | `"openstack-creds"` |
| `openstack.createSecret` | Whether the chart creates the Secret from `openstack.credentials`. | `false` |
| `openstack.credentials` | Key/value pairs for the created Secret (only used with `createSecret=true`). | `{}` |

Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`. For example:

```bash
helm install snapshot-cli oci://ghcr.io/kengou/snapshot-cli/charts/snapshot-cli --version 0.2.0 \
  --set openstack.createSecret=true \
  --set openstack.credentials.OS_AUTH_URL=https://my-openstack.com
```
