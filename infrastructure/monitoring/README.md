# Monitoring

Prometheus scrapes the API's public `/metrics` exposition endpoint and evaluates the rules in `../prometheus/alerts.yml`. Grafana provisions a Prometheus datasource and an operational overview dashboard from `../grafana/provisioning/`.

The currently exposed trip counters are low-cardinality operational signals. They do not yet include request latency histograms, database exporter metrics, worker lag, or incident/wallet integrity metrics; do not infer those values from the current dashboard. Add exporters/instrumentation before defining alerts for those domains.