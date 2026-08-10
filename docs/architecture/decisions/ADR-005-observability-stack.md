# ADR-005: Kubernetes Observability Stack

## Status

Accepted

## Context

Cloud Efficiency Engine requires Kubernetes infrastructure and workload
metrics to identify resource inefficiencies.

The platform needs metrics describing:

- Kubernetes workload metadata.
- Resource requests.
- Resource limits.
- Pod information.
- Node information.
- Resource utilization.

Manually maintaining individual Prometheus, Grafana and Kubernetes
monitoring manifests would increase operational complexity and make the
local demonstration harder to reproduce.

## Decision

Use `kube-prometheus-stack` as the primary Kubernetes observability
stack for the local demonstration environment.

The stack provides:

- Prometheus
- Grafana
- Alertmanager
- Prometheus Operator
- kube-state-metrics
- node-exporter

Helm is used as the package manager for the observability stack.

## Architecture

```text
Kubernetes
    |
    +---------------------+
    |                     |
    v                     v
kube-state-metrics   node-exporter
    |                     |
    +----------+----------+
               |
               v
          Prometheus
               |
        +------+------+
        |             |
        v             v
     Grafana        CEE
```

## Rationale

The decision provides:

- Reproducible installation.
- Standard Kubernetes observability components.
- Prometheus Operator integration.
- ServiceMonitor and PrometheusRule support.
- Easier local development.
- A deployment model that can later be adapted to cloud Kubernetes
  environments.

## Alternatives Considered

### Manually managed Prometheus manifests

Rejected because they increase configuration and lifecycle complexity.

### Standalone kube-state-metrics

Rejected as the primary approach because the selected observability
stack already manages kube-state-metrics.

### Custom metrics collector

Deferred.

A custom collector may be introduced later if Cloud Efficiency Engine
requires metrics that are not provided by the standard Kubernetes
observability stack.

## Consequences

### Positive

The monitoring environment becomes easier to reproduce and maintain.

The project can use standard Prometheus and Kubernetes observability
mechanisms.

### Negative

The Helm stack introduces additional components and operational
complexity.

The project must understand Prometheus Operator concepts such as:

- ServiceMonitor
- PodMonitor
- PrometheusRule

This complexity is considered acceptable because observability is a
core capability of the platform.