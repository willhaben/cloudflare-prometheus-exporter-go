# cloudflare-exporter

Helm chart for [cloudflare-prometheus-exporter-go](https://github.com/willhaben/cloudflare-prometheus-exporter-go).

## Install

```sh
kubectl create secret generic cloudflare-exporter-token \
  --from-literal=CLOUDFLARE_API_TOKEN=<your-token>

helm install cloudflare-exporter charts/cloudflare-exporter \
  --set apiToken.existingSecret=cloudflare-exporter-token
```

Build and push the `Containerfile` image first if you use your own registry. Then set `image.repository` and `image.tag`. The chart does not build or push images.

## Values

| Key | Default | Description |
|-----|---------|--------------|
| `replicaCount` | `2` | Number of replicas. More than one turns on leader election and adds the Role, RoleBinding and PodDisruptionBudget. |
| `topologySpreadConstraints` | `[]` | Pod topology spread constraints. When empty and `replicaCount` is greater than 1, the chart spreads replicas across nodes with a soft constraint. |
| `image.repository` | `ghcr.io/willhaben/cloudflare-prometheus-exporter-go` | Image repository, without the tag. |
| `image.tag` | `""` | An empty value uses `Chart.AppVersion`. |
| `image.pullPolicy` | `IfNotPresent` | |
| `apiToken.existingSecret` | `""` | Name of an existing Secret that holds the token. Use this value in production. |
| `apiToken.existingSecretKey` | `CLOUDFLARE_API_TOKEN` | Key in that Secret. |
| `apiToken.value` | `""` | Plaintext token. The chart creates the Secret from it. Use it only for a local test. Never commit a values file that sets it. The chart ignores it when `apiToken.existingSecret` is set. |
| `exporter.configPath` | `/etc/cloudflare-exporter/config.yaml` | The chart passes it as `-config`. It also sets the mount path and the key of the ConfigMap. |
| `exporter.analyticsWindow` | `1m` | `-analytics-window` |
| `exporter.analyticsLag` | `5m` | `-analytics-lag` |
| `exporter.zoneAnalyticsLag` | `0s` | `-zone-analytics-lag`. The value `0s` means the same as `analyticsLag` |
| `exporter.analyticsTimestamps` | `false` | `-analytics-timestamps` |
| `exporter.queryLimit` | `10000` | `-query-limit` |
| `exporter.metricsPath` | `/metrics` | Sets `server.metricsPath` in `config.yaml`. |
| `exporter.scrapeTimeout` | `120s` | Sets `server.scrapeTimeout` in `config.yaml`. Keep the `/scrape-timeout` annotation and `serviceMonitor.scrapeTimeout` at or above this value. |
| `exporter.accountConcurrency` | `4` | Sets `server.accountConcurrency` in `config.yaml`. This is the number of accounts that one job scrapes in parallel. |
| `exporter.requestTimeout` | `30s` | Sets `server.requestTimeout` in `config.yaml`. It limits one HTTP, WAF or errors analytics API call. It does not depend on `scrapeTimeout`. |
| `exporter.dnsRequestTimeout` | `120s` | Sets `server.dnsRequestTimeout` in `config.yaml`. It limits the DNS analytics call. That call queries a larger dataset for the full account, so it needs more time than `requestTimeout`. |
| `exporter.discovery.jobs` | one `production-zones` job, `env=production` | The chart copies it to `discovery.jobs` in `config.yaml`. The main [README](../../README.md#configuration) shows the format of `searchTags` and `accounts`. |
| `service.type` | `ClusterIP` | |
| `service.port` | `9199` | Also sets the container port and `listenAddress` in `config.yaml`. |
| `service.annotations` | `prometheus.io/scrape`, `/port`, `/path`, `/interval` (`150s`), `/scrape-timeout` (`120s`) | For Prometheus discovery without the Operator. Prometheus applies `interval` and `scrape-timeout` only if your scrape config maps them to `__scrape_interval__` and `__scrape_timeout__`. |
| `serviceAccount.create` | `true` | Create a ServiceAccount for the Deployment. |
| `serviceAccount.annotations` | `{}` | Annotations for the ServiceAccount, for example for IRSA or Workload Identity. |
| `serviceAccount.name` | `""` | An empty value uses the fullname of the chart. If `create` is `false`, an empty value uses `default`. |
| `serviceAccount.automountServiceAccountToken` | `false` | The chart ignores it when `replicaCount` is greater than 1. Leader election needs the token, so the pod always mounts it. |
| `serviceMonitor.enabled` | `false` | Needs the Prometheus Operator CRDs. |
| `serviceMonitor.interval` / `.scrapeTimeout` | `150s` / `120s` | |
| `serviceMonitor.honorLabels` | `true` | Keeps the `job` label of the exporter metrics, which holds the discovery job name. With `false`, Prometheus renames it to `exported_job`. |
| `resources` | `50m`/`64Mi` requests, `256Mi` limit | |
| `podSecurityContext`, `securityContext` | non-root, read-only root file system, no capabilities | |
| `nodeSelector`, `tolerations`, `affinity` | `{}` / `[]` / `{}` | |
| `podAnnotations`, `podLabels` | `{}` | |

## High availability

With more than one replica, the chart turns on leader election. It adds a Role, a RoleBinding and a PodDisruptionBudget. Only the leader queries Cloudflare. A standby replica serves only the `cloudflare_exporter_*` metrics. With `replicaCount: 1`, the chart renders none of these objects. A node drain then causes a gap in the metrics. Aggregate across replicas with `max without (pod, instance)`. Alert on `sum(cloudflare_exporter_leader) != 1`.

## Notes

The exporter has no separate poll loop. Each scrape queries Cloudflare directly. The scrape interval therefore sets the Cloudflare API call rate. To lower the call rate, raise the interval that your Prometheus uses. For annotation discovery, that is `prometheus.io/interval` in `service.annotations`. For Operator discovery, that is `serviceMonitor.interval`. Then set `exporter.analyticsWindow` to match. Otherwise the exporter under-counts, as the main README describes.

The Deployment uses its own ServiceAccount. With one replica, the pod does not mount the token, because the exporter does not call the Kubernetes API. Set `serviceAccount.create: false` to use an existing ServiceAccount.

The exporter cannot scrape Cloudflare without a token. Set `apiToken.existingSecret` before you install the chart. For a local test, you can set `apiToken.value`.

If you change `exporter.configPath`, the ConfigMap key and the mount path change with it. Keep the file name extension `.yaml`. The chart mounts the directory read-only.
