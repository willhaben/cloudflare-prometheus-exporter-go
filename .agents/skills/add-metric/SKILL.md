---
name: add-metric
description: Add a new Prometheus metric to the Cloudflare exporter, from the GraphQL query to the collector, tests and README. Use when a task adds or changes a `cloudflare_*` metric or a Cloudflare dataset field.
---

# Add a metric

Read `AGENTS.md` first. Its Metrics rules apply to every step below. All paths are relative to the repository root.

## 1. Decide the name

Follow the [Prometheus naming guide](https://prometheus.io/docs/practices/naming/). These rules matter here:

- Start with the `cloudflare_` prefix. The `namespace` constant and the `desc(...)` helper in `internal/collector/collector.go` add it. Pass the name without the prefix.
- Use base units in the name, for example `_seconds` or `_bytes`. Use one unit for each metric.
- Use `_ratio` for a value from 0 to 1. Use `_info` for a metric whose value is always 1.
- Never put a label value in the name. Use a label for a dimension, and only for a dimension with bounded cardinality.
- Zone metrics carry the labels `zone_id` and `zone`. Add them with `zoneLabels` and `withLabel(zoneLabels, "host")`.

Every `cloudflare_zone_*` analytics metric is a gauge. Each gauge holds the sum over the analytics window of one scrape. It is not a counter. So these metrics have no `_total` suffix. Existing examples are `cloudflare_zone_requests` and `cloudflare_zone_bandwidth_bytes`. Use `_total` only for a real counter, as the `cloudflare_exporter_*_errors_total` self metrics do.

Write the help text with `windowHelp("...")` so it names the analytics window. Self metrics of the exporter use the `exporter_` name part and do not use `windowHelp`.

## 2. Confirm the dataset and field names

Do not trust memory or documentation for a dataset or field name. Run a GraphQL introspection query against `https://api.cloudflare.com/client/v4/graphql`. Send an API token in the header `Authorization: Bearer <token>`. List the fields of the dataset type with `__type(name: "...")`. Find the type name with `__schema { types { name } }`. Confirm the field exists and note its type. Check the values of a dimension on a real zone before you use it as a label.

## 3. Apply the sampling rules

The `*Adaptive*` datasets are sampled. Always request `avg { sampleInterval }` for them.

`httpRequestsAdaptiveGroups` returns `count` and `sum.*` already extrapolated. Use them as returned. Do not multiply them by `sampleInterval`.

For any other dataset, compare its numbers with an exact source before you decide how to use them. For HTTP, the exact dataset is `httpRequests1mGroups`. It backs `cloudflare_zone_requests`. Read `internal/cloudflareapi/sampling.go` for `estimatedCount` and `lowConfidence`. The `Sampling` section of `README.md` explains the current choices.

## 4. Change the API layer

Work in `internal/cloudflareapi`. The files are `analytics.go`, `waf.go`, `dns.go` and `errorrate.go`.

1. Add the field to the query constant. If the metric needs a new dimension, add it under `dimensions`. If it needs a new value, add it under `sum`, `avg` or `count`.
2. Add the matching `json` field to the response struct of that file.
3. Add the field to the exported group struct, for example `ErrorGroup`. Copy it in the `Fetch...` function.
4. Reuse an existing query when the dataset is the same. A new dimension on the query adds rows, so mention `-query-limit` truncation in the README row.
5. Update the JSON fixture and the assertions in `internal/cloudflareapi/graphql_test.go`. `TestClient_FetchErrorMetrics` is the model.

A new dataset needs a new query, a `Fetch...` method on `Client`, and a matching method on the `cloudflareClient` interface in `internal/collector/collector.go`. Add the same method to `fakeClient` in `internal/collector/collector_test.go`. Add a `dataset...` constant next to `datasetHTTPRequestsAdaptive` if the dataset needs low-confidence accounting.

## 5. Change the collector

Work in `internal/collector/collector.go`.

1. Add a `*prometheus.Desc` field to the `Collector` struct.
2. Create it in `New` with `desc("zone_...", windowHelp("..."), withLabel(zoneLabels, "..."))`.
3. Add it to `allDescs()`. `Describe` reads this list. A Desc missing here makes the registry reject the metric.
4. Emit it in the emit function of its dataset. The functions are `emitZoneMetrics`, `emitWAFMetrics`, `emitErrorMetrics` and `emitDNSMetrics`. Sum rows in a map first. Then call `emitByZone` or `emitByLabel`. Use `emitNonZero` for per-label values that must not appear as zero.
5. Send the metric to the channel that the emit function receives. This channel comes from `c.stamped(...)` in `collectAccount`. It adds the window timestamp when `Options.Timestamps` is set. Never send to the raw channel.
6. Sum rows that share a label value. Two rows with equal labels make a duplicate series, and the registry rejects the whole scrape.
7. Keep zones that overlap between jobs safe. `claimZones` already makes each zone emit once per scrape, so no extra code is needed.

Gauges for a new dataset go in a new `wg.Go` block in `collectAccount`. That block needs its own `groups.Disable...` switch and its own `fetchTimed` call. Prefer a field on an existing query instead.

## 6. Add tests

- Collector test: add a case to `internal/collector/collector_test.go`. Put the input rows in `newFake()` or append them in the test, as `TestCollector_CachedBandwidthByHost` does.
- Call `gather(t, c)`. It uses a pedantic registry, so a duplicate series or a wrong Desc fails the test.
- Assert values with `metricValue(out, name, labels)`. The labels must match exactly. Use `hasSeries` for a series check without a value.
- Cover the edge cases: a zero value, a row that must be excluded, and rows that share a label value.
- `TestCollector_AllDescsInitialised` catches a Desc missing from `New`. No change is needed there.
- API test: see step 4, item 5.

## 7. Update the docs

Add one row to the metrics table in the `Metrics` section of `README.md`. Use the format `| `cloudflare_zone_name` | `label` | Description |`. Self metrics go in the `Exporter self-metrics` table. State when a value is a sampled estimate, and name the exact metric to compare. Mark high-cardinality labels as **High cardinality**. If the metric uses an adaptive dataset, check the `-analytics-lag` row and the `Sampling` section. Change the chart README `charts/cloudflare-exporter/README.md` only when a chart value changes.

## 8. Verify

Run these commands. They are the same checks as `.github/workflows/ci.yml`.

```sh
gofmt -l .
go vet ./...
golangci-lint run ./...
go build ./...
go test ./... -race
```

`make lint` runs the first three. `make test` runs `go test ./...` without `-race`.

Checklist:

- The name follows step 1, with no label value and no `_total` on a gauge.
- Dataset and field names are confirmed by introspection.
- Adaptive queries request `avg { sampleInterval }`, and `count` and `sum.*` of `httpRequestsAdaptiveGroups` are not scaled.
- The Desc is in the struct, in `New` and in `allDescs()`.
- The emit code uses the stamped channel and sums rows with equal labels.
- Collector and API tests pass with `-race`.
- The README row exists and matches the table format.
- Do not tag or bump the chart. The user releases.
