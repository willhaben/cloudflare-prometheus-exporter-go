# Project conventions

## Writing
- Write prose in ASD-STE100 Simplified Technical English.
- Keep sentences to 20 words or fewer. Use one idea for each sentence.
- Use the active voice and the imperative for instructions.
- Use one term for each thing. Do not change the term in the same document.

## Comments
- Explain why. Do not explain what. The code already states what it does. Write no comment when the code is clear without one.
- Use two lines at most. Put longer explanations in the pull request description or the commit message.
- Do not write history. Do not state what the code used to do, which incident caused a change, or when code was added. Use `git log` and `git blame` for this.

## Metrics
- Name Prometheus metrics according to https://prometheus.io/docs/practices/naming/#metric-names. Use a namespace prefix and base units. Use the suffixes `_total`, `_info`, and `_ratio` where they apply. Do not put label values in a metric name.
- The `*Adaptive*` GraphQL datasets from Cloudflare are sampled. Always request `avg { sampleInterval }`.
- Cloudflare already extrapolates `count` and `sum.*` in `httpRequestsAdaptiveGroups`. Do not multiply them by `sampleInterval`. Compare any other dataset with an exact source before you decide how to use it. See `internal/cloudflareapi/sampling.go`.
- Before you use a new dataset or field name, confirm it with live schema introspection.

## Releasing
A release is a `v*` git tag. GoReleaser builds the linux/amd64 binary and publishes the GitHub Release. It also pushes the container image to `ghcr.io/willhaben/cloudflare-prometheus-exporter-go`. The `dockers:` block in `.goreleaser.yaml` controls this push. No other manual step is needed.

The Helm chart `appVersion` is the default `image.tag`. It must match the tag. `.github/workflows/release.yml` fails if it does not. Follow these steps in order:

1. Run `make bump VERSION=x.y.z`. This sets `version` and `appVersion` in `charts/cloudflare-exporter/Chart.yaml`. Do not add a leading `v`.
2. Commit that change with the release content.
3. Run `git tag vx.y.z && git push origin main vx.y.z`.

Do not tag before step 1. Do not reuse a chart version. Unless the user explicitly asks otherwise, the user commits, tags, and pushes.
