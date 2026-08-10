# Local Demo Environment

Cloud Efficiency Engine provides a reproducible local Kubernetes
environment for demonstrating infrastructure efficiency analysis.

## Architecture

```text
Kind
  |
  +-- Kubernetes workloads
  |
  +-- kube-state-metrics
  |
  +-- Prometheus
  |
  +-- Grafana
  |
  +-- Alertmanager
          |
          v
Cloud Efficiency Engine
          |
          v
Optimization Analysis
```

## Purpose

The local demo provides a controlled Kubernetes environment containing
intentionally overprovisioned workloads.

The objective is to demonstrate how Cloud Efficiency Engine can:

1. Collect Kubernetes infrastructure metrics.
2. Compare requested resources against observed usage.
3. Identify inefficient workloads.
4. Generate optimization recommendations.
5. Estimate infrastructure cost.
6. Calculate potential monthly savings.
7. Calculate annualized savings.

## Prerequisites

The local demo requires:

- Docker
- Kubernetes CLI (`kubectl`)
- Kind
- Helm

Verify the required tools:

```bash
docker version
kubectl version --client
kind version
helm version
```

## Kubernetes Cluster

The demo uses Kind to create a local Kubernetes cluster.

Cluster name:

```text
cloud-efficiency
```

The cluster contains:

- 1 control-plane node
- 2 worker nodes

Check the cluster:

```bash
kubectl get nodes
```

Expected topology:

```text
cloud-efficiency-control-plane
cloud-efficiency-worker
cloud-efficiency-worker2
```

## Kubernetes Namespace

All application workloads used by the demo run inside:

```text
cloud-efficiency
```

Check the namespace:

```bash
kubectl get namespace cloud-efficiency
```

## Demo Workloads

The demo intentionally deploys workloads with resource requests
that are higher than their expected utilization.

This allows the optimization engine to identify resource waste.

### payments-api

Replicas:

```text
3
```

CPU request:

```text
1000m
```

CPU limit:

```text
2000m
```

Memory request:

```text
1Gi
```

Memory limit:

```text
2Gi
```

### orders-api

Replicas:

```text
2
```

CPU request:

```text
500m
```

CPU limit:

```text
1000m
```

Memory request:

```text
1Gi
```

Memory limit:

```text
2Gi
```

## Observability Stack

The demo uses `kube-prometheus-stack`.

The stack provides:

- Prometheus
- Grafana
- Alertmanager
- Prometheus Operator
- kube-state-metrics
- node-exporter

The architecture is:

```text
Kubernetes
     |
     +----------------------+
     |                      |
     v                      v
kube-state-metrics       node-exporter
     |                      |
     +----------+-----------+
                |
                v
           Prometheus
                |
       +--------+--------+
       |                 |
       v                 v
    Grafana          Cloud Efficiency
                         Engine
```

## Metrics

Cloud Efficiency Engine analyzes metrics related to:

- CPU requests
- CPU utilization
- Memory requests
- Memory utilization
- Kubernetes workload metadata
- Replica information

The initial analysis focuses on identifying resource
overprovisioning.

## Optimization Flow

The analysis pipeline is:

```text
Kubernetes
    |
    v
Metrics Collection
    |
    v
Prometheus
    |
    v
Metrics Provider
    |
    v
Workload Resolution
    |
    v
Analysis Engine
    |
    v
Optimization Rules
    |
    v
Cost Calculator
    |
    v
Potential Savings
```

## Cost Analysis

The engine estimates:

```text
Current Monthly Cost
        |
        v
Optimized Monthly Cost
        |
        v
Potential Monthly Savings
        |
        v
Annualized Savings
```

Example:

```text
Current monthly cost:       $500
Optimized monthly cost:     $350
Potential monthly savings:  $150
Annualized savings:         $1,800
```

The values above are illustrative.

Actual results depend on:

- Workload resource requests
- Observed utilization
- Optimization rules
- Cloud pricing configuration
- Analysis window

## Running the Demo

Create the cluster:

```bash
kind create cluster --config deployments/kind/cluster.yaml
```

Create the namespace:

```bash
kubectl apply -f deployments/kubernetes/namespace.yml
```

Deploy the demo workloads:

```bash
kubectl apply -f deployments/kubernetes/workloads.yml
```

Verify workloads:

```bash
kubectl get deployments -n cloud-efficiency
```

Verify Pods:

```bash
kubectl get pods -n cloud-efficiency
```

Verify the monitoring stack:

```bash
kubectl get pods -n monitoring
```

## Accessing Prometheus

Find the Prometheus service:

```bash
kubectl get svc -n monitoring
```

Port-forward Prometheus:

```bash
kubectl port-forward -n monitoring svc/monitoring-kube-prometheus-prometheus 9090:9090
```

Prometheus will then be available at:

```text
http://localhost:9090
```

## Accessing Grafana

Find the Grafana service:

```bash
kubectl get svc -n monitoring
```

Port-forward Grafana:

```bash
kubectl port-forward -n monitoring svc/monitoring-grafana 3000:80
```

Grafana will then be available at:

```text
http://localhost:3000
```

The default administrator credentials are managed by the Kubernetes Secret
created by the Helm chart.

Retrieve the password with:

```bash
kubectl get secret monitoring-grafana \
  -n monitoring \
  -o jsonpath="{.data.admin-password}" | base64 --decode
```

## Validation

Before running Cloud Efficiency Engine, validate that Prometheus
contains Kubernetes metrics.

Example queries:

```promql
kube_pod_info
```

```promql
kube_pod_container_resource_requests
```

```promql
kube_pod_container_resource_limits
```

```promql
container_cpu_usage_seconds_total
```

```promql
container_memory_working_set_bytes
```

## Future Demo

The final demo will provide a single command:

```bash
make demo
```

The command will eventually:

1. Create the Kind cluster.
2. Deploy the Kubernetes workloads.
3. Install the observability stack.
4. Deploy Cloud Efficiency Engine.
5. Configure Prometheus.
6. Execute an analysis.
7. Generate an optimization report.
8. Display potential monthly and annual savings.

## Reproducibility

The local environment is intentionally designed to be reproducible.

Infrastructure configuration is version-controlled in the repository.

Future versions will package the complete environment using:

- Helm
- Terraform where appropriate
- Kubernetes manifests
- Docker
- GitHub Actions

The goal is to allow a developer or potential client to reproduce
the demonstration environment with minimal manual configuration.