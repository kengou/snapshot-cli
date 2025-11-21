# snapshot-cli Helm Chart

A Helm chart for deploying the snapshot-cli tool as CronJobs in Kubernetes.

## Installation

To install the chart from the OCI registry with the release name `snapshot-cli`:

```bash
helm install snapshot-cli oci://ghcr.io/kengou/snapshot-cli/charts/snapshot-cli --version 0.1.3
```

## Configuration

The following table lists the configurable parameters of the snapshot-cli chart and their default values.

| Parameter | Description | Default |
| --- | --- | --- |
| `image.repository` | The image repository to pull from. | `ghcr.io/kengou/snapshot-cli` |
| `image.pullPolicy` | The image pull policy. | `IfNotPresent` |
| `image.tag` | The image tag to use. | `main` |
| `imagePullSecrets` | A list of secrets to use for pulling images. | `[]` |
| `nameOverride` | A string to override the chart name. | `""` |
| `fullnameOverride` | A string to override the fully qualified app name. | `""` |
| `securityContext` | The security context for the pod. | `{}` |
| `resources` | The resources to allocate for the pod. | `{}` |
| `cleanup.schedule` | The schedule for the cleanup cronjob. | `"0 0 * * *"` |
| `cleanup.args` | The arguments for the cleanup cronjob. | `["cleanup", "--share", "--older-than", "168h"]` |
| `create.schedule` | The schedule for the create cronjob. | `"0 2 * * *"` |
| `create.args` | The arguments for the create cronjob. | `["snapshot", "create", "--share-id", "SHARE_ID_PLACEHOLDER", "--name", "NAME_PLACEHOLDER", "--description", "DESCRIPTION_PLACEHOLDER", "--cleanup", "--older-than", "168h"]` |
| `openstack.enabled` | Whether to create a secret with OpenStack credentials. | `false` |
| `openstack.secretName` | The name of the secret to create. | `"openstack-creds"` |
| `openstack.OS_AUTH_URL` | The OpenStack authentication URL. | `""` |
| `openstack.OS_USERNAME` | The OpenStack username. | `""` |
| `openstack.OS_PASSWORD` | The OpenStack password. | `""` |
| `openstack.OS_USER_DOMAIN_NAME` | The OpenStack user domain name. | `""` |
| `openstack.OS_PROJECT_NAME` | The OpenStack project name. | `""` |
| `openstack.OS_PROJECT_DOMAIN_NAME` | The OpenStack project domain name. | `""` |
| `openstack.OS_REGION_NAME` | The OpenStack region name. | `""` |

Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`. For example:

```bash
helm install snapshot-cli oci://ghcr.io/kengou/snapshot-cli/charts/snapshot-cli --version 0.1.3 --set openstack.enabled=true,openstack.OS_AUTH_URL=https://my-openstack.com
```
