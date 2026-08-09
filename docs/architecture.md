# Architecture

## 1. Overview

Cloud Efficiency Engine is a cloud-native platform designed to analyze
Kubernetes workloads and identify resource inefficiencies, optimization
opportunities and potential cost impact.

The initial version focuses on detecting resource overprovisioning by
comparing requested resources against actual resource utilization.

The architecture is intentionally designed to support multiple metrics
providers and cloud providers without coupling the analysis engine to
a specific infrastructure platform.

---

## 2. Goals

The platform aims to:

- Analyze Kubernetes workload resource utilization.
- Detect CPU and memory overprovisioning.
- Identify inefficient resource allocation.
- Generate actionable optimization recommendations.
- Estimate potential infrastructure impact.
- Support multiple metrics providers.
- Remain independently testable without requiring a Kubernetes cluster.
- Provide a foundation for future cloud cost analysis.

---

## 3. MVP Scope

The first MVP focuses on:

- CPU requests.
- CPU utilization.
- Memory requests.
- Memory utilization.
- Resource utilization ratios.
- Overprovisioning detection.
- Optimization recommendations.
- REST API exposure.

The MVP does not require a real Kubernetes cluster.

Synthetic and mock metrics can be used during development and testing.

---

## 4. High-Level Architecture

```text
                         ┌─────────────────────┐
                         │ Kubernetes Cluster  │
                         └──────────┬──────────┘
                                    │
                                    ▼
                         ┌─────────────────────┐
                         │  Metrics Provider   │
                         │                     │
                         │ Prometheus / Mock   │
                         └──────────┬──────────┘
                                    │
                                    ▼
                         ┌─────────────────────┐
                         │   Analysis Engine   │
                         │                     │
                         │ CPU                 │
                         │ Memory              │
                         │ Utilization         │
                         └──────────┬──────────┘
                                    │
                                    ▼
                         ┌─────────────────────┐
                         │     Rule Engine     │
                         │                     │
                         │ CPU Overprovision   │
                         │ Memory Overprovision│
                         │ Low Utilization     │
                         └──────────┬──────────┘
                                    │
                                    ▼
                         ┌─────────────────────┐
                         │ Recommendation      │
                         │ Engine              │
                         └──────────┬──────────┘
                                    │
                                    ▼
                         ┌─────────────────────┐
                         │      REST API       │
                         │                     │
                         │ /health             │
                         │ /analyze            │
                         │ /recommendations    │
                         └─────────────────────┘
```

---

## 5. Architectural Principles

### Separation of Concerns

Each component has a single responsibility.

Metrics collection, analysis, rules, recommendations and HTTP delivery
must remain independent.

### Dependency Inversion

The analysis engine must not depend directly on Prometheus,
Kubernetes APIs or a specific cloud provider.

Instead, infrastructure integrations implement abstractions defined
by the application.

### Provider Abstraction

Metrics must be obtained through a provider abstraction.

```text
MetricsProvider
      │
      ├── MockMetricsProvider
      │
      └── PrometheusMetricsProvider
```

This allows the core application to be tested without external
infrastructure.

### Testability

Business rules must be executable using in-memory data.

A developer should be able to run the complete unit test suite without:

- Kubernetes
- AWS
- Azure
- Prometheus
- Docker

### Extensibility

Optimization rules must be independently replaceable and extendable.

Adding a new optimization rule should not require modifying the
existing rules.

---

## 6. Component Responsibilities

### Metrics Provider

Responsible for obtaining workload metrics.

Examples:

- Prometheus.
- Kubernetes Metrics API.
- Mock provider.

It does not perform optimization analysis.

---

### Analysis Engine

Responsible for orchestrating the analysis process.

It receives workload metrics and evaluates the configured rules.

---

### Rule Engine

Contains individual optimization rules.

Examples:

- CPU overprovisioning.
- Memory overprovisioning.
- Low utilization.
- Excessive replicas.
- Missing resource limits.

Each rule evaluates a workload and optionally produces a recommendation.

---

### Recommendation Engine

Transforms analysis results into actionable recommendations.

A recommendation should explain:

- What was detected.
- Why it matters.
- What could be changed.
- Estimated impact.
- Confidence level.

---

### REST API

Exposes the analysis capabilities to external clients.

The API must not contain business rules.

HTTP handlers should delegate to application services.

---

## 7. Domain Model

The initial domain revolves around Kubernetes workload metrics.

```text
Workload
   │
   ├── Namespace
   ├── Name
   ├── Replicas
   │
   ├── CPU
   │    ├── Requested
   │    └── Usage
   │
   └── Memory
        ├── Requested
        └── Usage
```

The domain model must remain independent of Prometheus,
Kubernetes SDKs and cloud provider SDKs.

---

## 8. Analysis Flow

```text
1. Obtain workload metrics
            ↓
2. Validate metrics
            ↓
3. Execute optimization rules
            ↓
4. Generate recommendations
            ↓
5. Aggregate analysis result
            ↓
6. Return result through API
```

---

## 9. Example

Given:

```text
Workload: payments-api

CPU request:       1000m
CPU usage:          180m

Memory request:    2048Mi
Memory usage:       640Mi
```

The engine may produce:

```text
Status: WARNING

CPU utilization: 18%
Memory utilization: 31%

Recommendation:

CPU appears to be overprovisioned.

Suggested request:
1000m → 300m

Confidence:
HIGH
```

The values shown above are illustrative.

---

## 10. Future Architecture

The architecture is expected to evolve toward:

```text
                         ┌──────────────────────┐
                         │ Kubernetes Clusters  │
                         └──────────┬───────────┘
                                    │
                         ┌──────────▼───────────┐
                         │ Metrics Collection   │
                         └──────────┬───────────┘
                                    │
                                    ▼
                         ┌──────────────────────┐
                         │ Analysis Platform    │
                         └──────────┬───────────┘
                                    │
                  ┌─────────────────┼─────────────────┐
                  ▼                 ▼                 ▼
             Efficiency          Cost            Reliability
               Engine            Engine             Engine
                  │                 │                 │
                  └─────────────────┼─────────────────┘
                                    ▼
                         ┌──────────────────────┐
                         │ Recommendations      │
                         └──────────┬───────────┘
                                    ▼
                         ┌──────────────────────┐
                         │ Dashboard / API      │
                         └──────────────────────┘
```

Future capabilities may include:

- AWS cost integration.
- Azure cost integration.
- Multi-cluster analysis.
- Historical analysis.
- Cost allocation.
- Cloud pricing providers.
- Automated recommendations.
- Dashboard.
- Alerts.
- Multi-tenant SaaS architecture.

---

## 11. Non-Goals

The MVP will not attempt to:

- Automatically modify production workloads.
- Automatically resize Kubernetes resources.
- Manage cloud infrastructure.
- Replace Kubernetes monitoring systems.
- Replace cloud billing platforms.

The system initially provides analysis and recommendations.

Automation may be considered in future versions.

---

## 12. Security Considerations

The system should follow least-privilege principles when accessing
Kubernetes or cloud infrastructure.

The MVP should not require cluster-admin permissions.

Future Kubernetes integrations should use read-only permissions
whenever possible.

---

## 13. Observability

The application itself should eventually expose:

- Application metrics.
- Request latency.
- Analysis execution time.
- Rule execution time.
- Error rates.

Future integrations may include:

- Prometheus.
- OpenTelemetry.
- Distributed tracing.

---

## 14. Deployment

The application is designed to run as a containerized workload.

Target deployment environments:

- Local Docker.
- Kubernetes.
- AWS.
- Azure.

Infrastructure provisioning will be managed through Terraform
where applicable.

---

## 15. Design Decision

The core analysis engine is intentionally independent from the
infrastructure layer.

This allows the project to evolve from:

```text
Kubernetes optimization tool
```

into:

```text
Cloud efficiency and infrastructure intelligence platform
```

without requiring a complete rewrite of the domain logic.