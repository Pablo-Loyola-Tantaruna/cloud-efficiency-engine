# Multi-cloud Provider Architecture

Cloud Efficiency Engine keeps the analysis core independent from individual cloud SDKs.
Provider-specific implementations are registered through `internal/providers.Registry`.

## Capabilities

Every cloud provider can expose these capabilities:

- Metrics
- Historical metrics
- Pricing
- Billing
- Capacity

The Registry resolves the capabilities for a `domain.AnalysisContext` and returns a single bundle to the analysis engine.

## Providers

| Provider | Metrics | Historical | Pricing | Billing | Capacity |
| --- | --- | --- | --- | --- | --- |
| AWS | supported | supported | supported | supported | supported |
| Kubernetes | supported | supported | supported | optional | supported |
| Azure | contract/adapter layer | contract/adapter layer | contract/adapter layer | contract/adapter layer | adapter layer |
| GCP | contract/adapter layer | contract/adapter layer | contract/adapter layer | contract/adapter layer | adapter layer |

AWS and Kubernetes are the currently exercised integrations. Azure and GCP provide the same provider contracts and registration model; their cloud-specific API clients can be wired independently without changing the analysis engine.

## Adding a provider

A provider should expose a small adapter layer and one registration function:

```text
internal/providers/<provider>/
  provider.go
  registry.go
  metrics.go
  pricing.go
  billing.go
  capacity.go
```

The adapter translates cloud-specific APIs into the internal contracts. The analysis engine must not contain provider-specific branching.

## Cost attribution

Billing returns the actual cost for the requested period. Cost attribution combines that cost with provider capacity and workload metrics. Attribution uses normalized CPU and memory allocations and can report allocated and unallocated cost.

Monthly cost used for attribution is derived from the requested billing period; the raw billing total remains the actual period cost in the analysis report.
