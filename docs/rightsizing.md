# Workload Rightsizing

## Scope

`RIGHTSIZE_WORKLOAD_CPU` and `RIGHTSIZE_WORKLOAD_MEMORY` change Kubernetes resource requests on Deployments, StatefulSets, and DaemonSets.

Jobs are intentionally excluded from mutation because changing an existing Job template is not a safe in-place rightsizing operation.

## Target identity

Every executable workload recommendation carries:

- `namespace/name` workload reference
- Kubernetes `WorkloadType`
- optional `ContainerName`

A multi-container workload must identify its target container explicitly. A single-container workload can omit the container name.

## Execution safety

Before changing a workload, the executor reads the current request and compares it with the action's `CurrentValue`.

- Current value differs from `CurrentValue` and is not already desired: execution fails with drift.
- Current value already equals `DesiredValue`: execution succeeds idempotently without an update.
- Current value equals `CurrentValue`: only the relevant CPU or memory request is changed.

Limits, replicas, probes, scheduling rules, and other container settings are preserved.

## Multicloud model

The mutation is performed through the Kubernetes API rather than cloud-specific SDKs. EKS, AKS, and GKE all expose the same Kubernetes workload API, so one executor covers all three clouds while the action still records the logical cloud provider and cluster identity.

The API process can use either an explicit `FINOPS_KUBECONFIG` or an in-cluster service account. If neither is available, workload execution remains disabled and the API continues to expose the existing not-configured behavior.

## Local configuration

```text
FINOPS_KUBECONFIG=/path/to/kubeconfig
```

In Kubernetes, leave `FINOPS_KUBECONFIG` unset and provide normal in-cluster credentials through the service account.

The service account used for execution should be limited to `get`/`update` on the supported workload resources in the namespaces the tenant can operate. Read-only analysis credentials should not automatically receive mutation permissions.
