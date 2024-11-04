# Keel - automated Kubernetes deployments for the rest of us

* Website [https://keel.sh](https://keel.sh)
* User Guide [https://keel.sh/docs/](https://keel.sh/docs/)

Keel is a tool for automating [Kubernetes](https://kubernetes.io/) deployment updates. Keel is stateless, robust and lightweight.

Keel provides several key features:

* __[Kubernetes](https://kubernetes.io/) and [Helm](https://helm.sh) providers__ - Keel has direct integrations with Kubernetes and Helm.

* __No CLI/API__ - tired of `***ctl` for everything? Keel doesn't have one. Gets job done through labels, annotations, charts.

* __Semver policies__ - specify update policy for each deployment/Helm release individually.

* __Automatic [Google Container Registry](https://cloud.google.com/container-registry/) configuration__ - Keel automatically sets up topic and subscriptions for your deployment images by periodically scanning your environment.

* __[Native, DockerHub and Quay webhooks](https://keel.sh/user-guide/triggers/#webhooks) support__ -  once webhook is received impacted deployments will be identified and updated.

*  __[Polling](https://keel.sh/user-guide/#polling-deployment-example)__ - when webhooks and pubsub aren't available - Keel can still be useful by checking Docker Registry for new tags (if current tag is semver) or same tag SHA digest change (ie: `latest`).

* __Notifications__ - out of the box Keel has Slack, HipChat, Mattermost, Teams and standard webhook notifications, more info [here](https://keel.sh/user-guide/#notifications)


## Installing

Docker image _polling_, _Kubernetes provider_ and _Helm provider_ support are set by default, then Kubernetes _deployments_ can be upgraded when new Docker image is available:

```console
$ helm upgrade --install keel --namespace keel keel/keel
```

### Setting up Helm release to be automatically updated by Keel

Add the following to your app's `values.yaml` file and do `helm upgrade ...`:

```
keel:
  # keel policy (all/major/minor/patch/force)
  policy: all
  # trigger type, defaults to events such as pubsub, webhooks
  trigger: poll
  # polling schedule
  pollSchedule: "@every 3m"
  # images to track and update
  images:
    - repository: image.repository # it must be the same names as your app's values
      tag: image.tag # it must be the same names as your app's values
```

The same can be applied with `--set` flag without using `values.yaml` file:

```console
$ helm upgrade --install whd webhookdemo --namespace keel --reuse-values \
  --set keel.policy="all",keel.trigger="poll",keel.pollSchedule="@every 3m" \
  --set keel.images[0].repository="image.repository" \
  --set keel.images[0].tag="image.tag"
```

You can read in more details about supported policies, triggers and etc in the [User Guide](https://keel.sh/user-guide/).

Also you should check the [Webhooh demo app](https://github.com/webhookrelay/webhook-demo) and it's chart to have more clear
idea how to set automatic updates.


## Uninstalling the Chart

To uninstall/delete the `keel` deployment:

```console
$ helm delete --purge keel
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

## Configuration

The following table lists has the main configurable parameters (polling, triggers, notifications, service) of the _Keel_ chart and they apply to both Kubernetes and Helm providers:

| Parameter                                   | Description                            | Default                                                   |
| ------------------------------------------- | -------------------------------------- | --------------------------------------------------------- |
| `polling.enabled`                           | Docker registries polling              | `true`                                                    |
| `helmProvider.enabled`                      | Enable/disable Helm provider           | `true`                                                    |
| `helmProvider.helmDriver`                   | Set driver for Helm3                   | ``                                                        |
| `helmProvider.helmDriverSqlConnectionString`| Set SQL connection string for Helm3    | ``                                                        |
| `gcr.enabled`                               | Enable/disable GCR Registry            | `false`                                                   |
| `gcr.projectId`                             | GCP Project ID GCR belongs to          |                                                           |
| `gcr.pubsub.enabled`                        | Enable/disable GCP Pub/Sub trigger     | `false`                                                   |
| `ecr.enabled`                               | Enable/disable AWS ECR Registry        | `false`                                                   |
| `ecr.roleArn`                               | Service Account IAM Role ARN for EKS   |                                                           |
| `ecr.accessKeyId`                           | AWS_ACCESS_KEY_ID for ECR Registry     |                                                           |
| `ecr.secretAccessKey`                       | AWS_SECRET_ACCESS_KEY for ECR Registry |                                                           |
| `ecr.region`                                | AWS_REGION for ECR Registry            |                                                           |
| `insecureRegistry`                          | Enable/disable insecure registries     | `false`                                                   |
| `webhook.enabled`                           | Enable/disable Webhook Notification    | `false`                                                   |
| `webhook.endpoint`                          | Remote webhook endpoint                |                                                           |
| `slack.enabled`                             | Enable/disable Slack Notification      | `false`                                                   |
| `slack.botName`                             | Name of the Slack bot                  |                                                           |
| `slack.token`                               | Slack token                            |                                                           |
| `slack.channel`                             | Slack channel                          |                                                           |
| `slack.approvalsChannel`                    | Slack channel for approvals            |                                                           |
| `teams.enabled`                             | Enable/disable MS Teams Notification   | `false`                                                   |
| `teams.webhookUrl`                          | MS Teams Connector's webhook url       |                                                           |
| `service.enabled`                           | Enable/disable Keel service            | `false`                                                   |
| `service.type`                              | Keel service type                      | `LoadBalancer`                                            |
| `service.externalIP`                        | Keel static IP                         |                                                           |
| `service.externalPort`                      | Keel service port                      | `9300`                                                    |
| `service.clusterIP`                         | Keel service clusterIP                 |                                                           |
| `webhookRelay.enabled`                      | Enable/disable WebhookRelay integration| `false`                                                   |
| `webhookRelay.key`                          | WebhookRelay key                       |                                                           |
| `webhookRelay.secret`                       | WebhookRelay secret                    |                                                           |
| `webhookRelay.bucket`                       | WebhookRelay bucket                    |                                                           |
| `rbac.enabled`                              | Enable/disable RBAC installation       | `true`                                                    |
| `hipchat.enabled`                           | Enable/disable Hipchat integration     | `false`                                                   |
| `hipchat.token`                             | Hipchat token                          |                                                           |
| `hipchat.channel`                           | Hipchat channel                        |                                                           |
| `hipchat.approvalsChannel`                  | Hipchat channel for approvals          |                                                           |
| `hipchat.botName`                           | Name of the Hipchat bot                |                                                           |
| `hipchat.userName`                          | Hipchat username in Jabber format      |                                                           |
| `hipchat.password`                          | Hipchat password for approvals user    |                                                           |
| `mail.enabled`                              | Enable/disable mail notifications      | `false`                                                   |
| `mail.from`                                 | Mail sender address                    |                                                           |
| `mail.to`                                   | Mail destination address               |                                                           |
| `mail.smtp.server`                          | Mail SMTP server address               |                                                           |
| `mail.smtp.port`                            | Mail SMTP server port                  | `25`                                                      |
| `mail.smtp.user`                            | Mail SMTP server user (optional)       |                                                           |
| `mail.smtp.pass`                            | Mail SMTP server password (optional)   |                                                           |
| `mattermost.enabled`                        | Enable/disable Mattermost integration  | `false`                                                   |
| `mattermost.endpoint`                       | Mattermost API endpoint                |                                                           |
| `googleApplicationCredentials`              | GCP Service account key configurable   |                                                           |
| `hipchat.password`                          | Hipchat password for approvals user    |                                                           |
| `gcloud.managedCertificates.enabled`        | Enable/Disable managed ssl on Gcloud   | `false`                                                   |
| `gcloud.managedCertificates.domains`        | List of managed certificate domains    | `[]`                                                      |
| `ingress.enabled`                           | Enables Ingress                        | `false`                                                   |
| `ingress.annotations`                       | Ingress annotations                    | `{}`                                                      |
| `ingress.labels`                            | Ingress labels                         | `{}`                                                      |
| `ingress.hosts`                             | Ingress accepted hosts                 | `[]`                                                      |
| `ingress.tls`                               | Ingress TLS configuration              | `[]`                                                      |
| `basicauth.enabled`                         | Enable/disable Basic Auth on approvals | `false`                                                   |
| `basicauth.user`                            | Basic Auth username                    |                                                           |
| `basicauth.password`                        | Basic Auth password                    |                                                           |
| `dockerRegistry.enabled`                    | Docker registry secret enabled.        | `false`                                                   |
| `dockerRegistry.name`                       | Docker registry secret name            |                                                           |
| `dockerRegistry.key`                        | Docker registry secret key             |                                                           |
| `secret.name`                               | Secret name                            |                                                           |
| `secret.create`                             | Create secret                          | `true`                                                    |
| `persistence.enabled`                       | Enable/disable audit log persistence   | `true`                                                    |
| `persistence.storageClass`                  | Storage Class for the Persistent Volume| `-`                                                       |
| `persistence.size`                          | Persistent Volume size                 | `1Gi`                                                     |
| `imagePullSecrets`                          | Image pull secrets                     | `[]`                                                      |

Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`.

Alternatively, a YAML file that specifies the values for the above parameters can be provided while installing the chart. For example,

```console
$ helm install --name keel --namespace keel -f values.yaml keel/keel
```
> **Tip**: You can use the default [values.yaml](values.yaml)
