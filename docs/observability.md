# Observability

The observability stack is intentionally split by signal and keeps the application vendor-neutral.

## Signals

- Metrics: Prometheus, exposed by the API and FinOps metrics endpoints.
- Traces: OpenTelemetry SDK -> OTLP -> OpenTelemetry Collector -> Tempo.
- Logs: structured JSON through `slog`; HTTP logs include request ID, trace ID and span ID so they can be correlated by a log collector.
- Visualization: Grafana.

## Why these components

### Prometheus

The project already exposes domain-level FinOps metrics. This remains the source for alerting, recording rules and Grafana metrics panels.

### OpenTelemetry

OpenTelemetry is the instrumentation API and protocol boundary. The application does not depend on Tempo, Jaeger or Elastic-specific SDKs.

### OpenTelemetry Collector

The Collector is the routing boundary. Trace exporters can be changed without recompiling the engine. The default local stack routes traces to Tempo.

### Tempo

Tempo is the default trace backend because it integrates directly with Grafana, Prometheus and Loki and is lightweight for a self-hosted FinOps control plane.

### Loki

Loki is the default log backend. The application keeps JSON logs on stdout/stderr; a deployment-level log collector can ship those logs to Loki while preserving the trace ID emitted by the API middleware.

### Grafana

Grafana provides the operator view across metrics, logs and traces. The provisioned dashboard focuses on execution reliability, provider latency and savings realization.

## Why not Elastic Stack by default

Elastic is a valid enterprise backend and can ingest OpenTelemetry data. It is intentionally not part of the default stack because it would duplicate the Prometheus + Loki + Tempo + Grafana platform and add another storage/search operating model.

For organizations that already standardize on Elastic, configure the Collector to export OTLP to the Elastic endpoint instead of running Tempo/Loki.

## Why not Jaeger by default

Jaeger is a valid OTLP trace backend, but running it together with Tempo would duplicate trace storage and querying. If a team standardizes on Jaeger, the Collector can export OTLP to Jaeger instead and Tempo can be disabled.

## Local stack

Start the existing application dependencies plus observability with:

```bash
docker compose -f docker-compose.yml -f docker-compose.observability.yml up --build
```

Endpoints:

- API: `http://localhost:8080`
- Prometheus: `http://localhost:9090`
- Grafana: `http://localhost:3000`
- Tempo: `http://localhost:3200`
- Loki: `http://localhost:3100`
- OTLP HTTP: `http://localhost:4318`
- OTLP gRPC: `http://localhost:4317`

Grafana is provisioned with Prometheus, Loki and Tempo data sources and a FinOps overview dashboard.

## Configuration

- `OTEL_ENABLED=true` enables tracing explicitly.
- `OTEL_EXPORTER_OTLP_ENDPOINT` configures the OTLP HTTP endpoint.
- `OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf` is used by the local stack.
- `OTEL_SERVICE_NAME` identifies the service in telemetry backends.
- `OTEL_TRACES_SAMPLER_ARG` controls the trace ID ratio sampler; default is 0.10.

## FinOps metrics

The runtime telemetry adds metrics for:

- action execution outcomes and duration
- provider operation outcomes and duration
- verification outcomes and duration
- current potential savings
- realized savings from successful, newly created executions

The existing analysis metrics remain available for namespace, workload and recommendation analysis.

## SLO alerts

Prometheus rules cover:

- execution success below 99%
- verification failures above 1%
- repeated scheduler failures

Tune thresholds only after measuring the normal operating baseline.
