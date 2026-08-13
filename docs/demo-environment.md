# Cloud Efficiency Engine — Demo Environment

This guide deploys Cloud Efficiency Engine locally on a Kubernetes cluster using Kind, Prometheus, Grafana and Helm.

The goal of the demo is to reproduce the complete FinOps optimization workflow:

```text
Kubernetes
    |
    v
Prometheus
    |
    v
Cloud Efficiency Engine
    |
    +--> Workload Analysis
    |
    +--> Optimization Rules
    |
    +--> Cost Calculation
    |
    +--> Recommendations
    |
    +--> Potential Savings
    |
    v
Prometheus Metrics
    |
    v
Grafana
```

---

## 1. Architecture

The demo environment contains two Kubernetes namespaces.

```text
Kind Cluster
|
+-- cloud-efficiency-engine
|   |
|   +-- cloud-efficiency-engine
|   |
|   +-- payments-api
|   |
|   +-- orders-api
|
+-- monitoring
    |
    +-- Prometheus
    |
    +-- Grafana
    |
    +-- kube-state-metrics
    |
    +-- Alertmanager
```

### Cloud Efficiency Engine

```text
+------------------------------------------------+
|              Cloud Efficiency Engine           |
+------------------------------------------------+
|                                                |
|  REST API                                      |
|  /health                                       |
|  /ready                                        |
|  /metrics                                      |
|  /metrics/finops                               |
|  /api/v1/analyze                               |
|                                                |
|  Analysis Scheduler                            |
|       |                                        |
|       v                                        |
|  Analysis Engine                               |
|       |                                        |
|       +--> Optimization Rules                  |
|       +--> Historical Analysis                 |
|       +--> Recommendation Resolution            |
|       +--> Pricing                             |
|       +--> Cost Calculation                    |
|                                                |
+------------------------------------------------+
```

### Monitoring

```text
+----------------------+
| Kubernetes            |
|                      |
| Pods / Deployments   |
+----------+-----------+
           |
           v
+----------------------+
| kube-state-metrics   |
+----------+-----------+
           |
           v
+----------------------+
| Prometheus           |
|                      |
| Raw K8s metrics      |
| CEE recording rules |
+----------+-----------+
           |
           +----------------------+
           |                      |
           v                      v
+----------------------+  +----------------------+
| Cloud Efficiency     |  | Grafana              |
| Engine               |  |                      |
|                      |  | FinOps Dashboard     |
| Analysis             |  | Recommendations      |
| Recommendations      |  | Scheduler            |
| Cost estimation      |  | Workloads            |
+----------------------+  +----------------------+
```

---

## 2. Prerequisites

Install and verify:

- Docker
- Kind
- kubectl
- Helm

Verify:

```bash
docker version
kind version
kubectl version --client
helm version
```

---

## 3. Create the Kind cluster

Create the cluster:

```bash
kind create cluster --name cloud-efficiency-engine
```

Verify:

```bash
kubectl cluster-info
```

```bash
kubectl get nodes
```

Expected:

```text
NAME                         STATUS   ROLES           AGE
cloud-efficiency-engine-control-plane   Ready    control-plane   ...
cloud-efficiency-engine-worker          Ready    <none>          ...
cloud-efficiency-engine-worker2         Ready    <none>          ...
```

---

## 4. Verify the monitoring stack

The demo expects the monitoring stack to exist in the `monitoring` namespace.

Check:

```bash
kubectl get pods -n monitoring
```

You should see components similar to:

```text
Prometheus
Grafana
kube-state-metrics
Alertmanager
node-exporter
```

Verify services:

```bash
kubectl get svc -n monitoring
```

The Prometheus service used by Cloud Efficiency Engine is:

```text
monitoring-kube-prometheus-prometheus
```

---

## 5. Build the Cloud Efficiency Engine image

From the repository root:

```bash
docker build -t cloud-efficiency-engine:dev .
```

Verify:

```bash
docker images cloud-efficiency-engine
```

---

## 6. Load the image into Kind

Because the demo uses a local Docker image, load it into the Kind cluster:

```bash
kind load docker-image cloud-efficiency-engine:dev --name cloud-efficiency-engine
```

Verify that the cluster can use the image:

```bash
docker exec \
  cloud-efficiency-engine-control-plane \
  crictl images | grep cloud-efficiency-engine
```

---

## 7. Create the application namespace

```bash
kubectl apply \
  -f deployments/kubernetes/namespace.yml
```

Verify:

```bash
kubectl get ns cloud-efficiency-engine
```

Expected:

```text
cloud-efficiency-engine   Active
```

---

## 8. Deploy the demo workloads

Apply:

```bash
kubectl apply \
  -f deployments/kubernetes/workloads.yml
```

Verify:

```bash
kubectl get deployments \
  -n cloud-efficiency-engine
```

Expected:

```text
NAME           READY   UP-TO-DATE   AVAILABLE
orders-api     2/2     2            2
payments-api   3/3     3            3
```

Verify Pods:

```bash
kubectl get pods \
  -n cloud-efficiency-engine \
  -o wide
```

---

## 9. Deploy Cloud Efficiency Engine with Helm

The recommended deployment method is Helm.

Run:

```bash
helm upgrade --install cloud-efficiency-engine \
  deployments/helm/cloud-efficiency-engine \
  --namespace cloud-efficiency-engine \
  --create-namespace \
  --set image.repository=cloud-efficiency-engine \
  --set image.tag=dev \
  --set image.pullPolicy=IfNotPresent
```

Verify the release:

```bash
helm list \
  -n cloud-efficiency-engine
```

Expected:

```text
NAME                NAMESPACE
cloud-efficiency    cloud-efficiency-engine
```

---

## 10. Validate the Helm release

Check the Deployment:

```bash
kubectl get deployments \
  -n cloud-efficiency-engine
```

Check Pods:

```bash
kubectl get pods \
  -n cloud-efficiency-engine
```

Check Service:

```bash
kubectl get svc \
  -n cloud-efficiency-engine
```

Check endpoints:

```bash
kubectl get endpoints \
  -n cloud-efficiency-engine
```

Check the release:

```bash
helm status \
  cloud-efficiency-engine \
  -n cloud-efficiency-engine
```

---

## 11. Run Helm tests

Run:

```bash
helm test \
  cloud-efficiency-engine \
  -n cloud-efficiency-engine
```

The test verifies that the service can reach the health endpoint.

Expected result:

```text
TEST SUITE: ...
PASSED
```

---

## 12. Validate health and readiness

Port-forward the service:

```bash
kubectl port-forward \
  -n cloud-efficiency-engine \
  svc/cloud-efficiency-engine \
  8080:8080
```

Depending on the Helm release name, the Service name may differ.

Verify the actual Service first:

```bash
kubectl get svc \
  -n cloud-efficiency-engine
```

Then:

```bash
curl http://localhost:8080/health
```

Expected:

```json
{
  "status": "UP"
}
```

Readiness:

```bash
curl http://localhost:8080/ready
```

Expected:

```json
{
  "status": "UP"
}
```

---

## 13. Verify the application metrics endpoint

Run:

```bash
curl http://localhost:8080/metrics
```

You should see metrics such as:

```text
cee_http_requests_total
cee_analysis_total
cee_analysis_errors_total
cee_workloads_analyzed_total
cee_optimizable_workloads_total
```

---

## 14. Verify FinOps metrics

Run:

```bash
curl http://localhost:8080/metrics/finops
```

The endpoint exposes metrics such as:

```text
cee_current_monthly_cost_usd
cee_optimized_monthly_cost_usd
cee_potential_savings_usd
cee_savings_percentage

cee_workload_count
cee_optimizable_workload_count

cee_workload_current_monthly_cost_usd
cee_workload_optimized_monthly_cost_usd
cee_workload_potential_savings_usd
cee_workload_savings_percentage
cee_workload_optimizable

cee_recommendation_count
cee_recommendation_workload_count
cee_recommendation_rule_count
cee_recommendation_severity_count
cee_recommendation_confidence_count

cee_scheduler_runs_total
cee_scheduler_success_total
cee_scheduler_failure_total
cee_scheduler_last_success_timestamp
cee_scheduler_last_failure_timestamp
cee_scheduler_last_duration_seconds
```

---

## 15. Verify ServiceMonitor

Check:

```bash
kubectl get servicemonitor \
  -n monitoring
```

Inspect:

```bash
kubectl get servicemonitor \
  -n monitoring \
  -o yaml
```

The Cloud Efficiency Engine ServiceMonitor should scrape:

```text
/metrics
/metrics/finops
```

---

## 16. Verify Prometheus targets

Port-forward Prometheus:

```bash
kubectl port-forward \
  -n monitoring \
  svc/monitoring-kube-prometheus-prometheus \
  9090:9090
```

Open:

```text
http://localhost:9090/targets
```

Find the Cloud Efficiency Engine target.

The target should be:

```text
UP
```

---

## 17. Verify raw Kubernetes metrics

Before testing CEE recording rules, verify that Prometheus can see Kubernetes metrics.

### Deployment replicas

```promql
kube_deployment_spec_replicas{
  namespace="cloud-efficiency-engine"
}
```

### CPU requests

```promql
kube_pod_container_resource_requests{
  namespace="cloud-efficiency-engine",
  resource="cpu"
}
```

### Memory requests

```promql
kube_pod_container_resource_requests{
  namespace="cloud-efficiency-engine",
  resource="memory"
}
```

### CPU usage

```promql
container_cpu_usage_seconds_total{
  namespace="cloud-efficiency-engine"
}
```

### Memory usage

```promql
container_memory_working_set_bytes{
  namespace="cloud-efficiency-engine"
}
```

---

## 18. Verify Cloud Efficiency recording rules

The platform transforms Kubernetes metrics into the CEE workload contract.

### Workload ownership

```promql
cee_workload_pod_owner{
  namespace="cloud-efficiency-engine"
}
```

### CPU requests

```promql
cee_workload_cpu_request_millicores{
  namespace="cloud-efficiency-engine"
}
```

Expected workloads:

```text
payments-api
orders-api
```

### CPU usage

```promql
cee_workload_cpu_usage_millicores{
  namespace="cloud-efficiency-engine"
}
```

### Memory requests

```promql
cee_workload_memory_request_bytes{
  namespace="cloud-efficiency-engine"
}
```

### Memory usage

```promql
cee_workload_memory_usage_bytes{
  namespace="cloud-efficiency-engine"
}
```

### Replicas

```promql
cee_workload_replicas{
  namespace="cloud-efficiency-engine"
}
```

---

## 19. Verify the analysis engine against real Kubernetes data

The Cloud Efficiency Engine should be configured to use Prometheus:

```text
METRICS_PROVIDER=prometheus
```

and:

```text
PROMETHEUS_URL=http://monitoring-kube-prometheus-prometheus.monitoring.svc:9090
```

The scheduler should use:

```text
ANALYSIS_NAMESPACE=cloud-efficiency-engine
```

and periodically execute analysis according to:

```text
ANALYSIS_INTERVAL=1m
```

Check the environment inside the Pod:

```bash
kubectl exec \
  -n cloud-efficiency-engine \
  deployment/cloud-efficiency-engine \
  -- env
```

---

## 20. Execute a manual analysis

Port-forward the application:

```bash
kubectl port-forward \
  -n cloud-efficiency-engine \
  svc/cloud-efficiency-engine \
  8080:8080
```

Run:

```bash
curl \
  -X POST \
  http://localhost:8080/api/v1/analyze \
  -H "Content-Type: application/json" \
  -d '{
    "namespace": "cloud-efficiency-engine",
    "lookbackHours": 24,
    "stepSeconds": 300
  }'
```

The response should contain:

```text
generatedAt
summary
workloads
```

The summary contains:

```text
totalWorkloads
optimizableWorkloads
currentMonthlyCostUsd
optimizedMonthlyCostUsd
potentialSavingsUsd
savingsPercentage
```

---

## 21. Verify workload recommendations

Each workload can contain:

```text
workload
status
history
recommendations
cost
```

A recommendation can contain:

```text
rule
workload
description
severity
confidence
currentCpuRequestMillicores
suggestedCpuRequestMillicores
currentMemoryRequestBytes
suggestedMemoryRequestBytes
```

This allows the demo to answer:

```text
What is expensive?
What is overprovisioned?
What should be changed?
How confident is the recommendation?
How much could be saved?
```

---

## 22. Verify FinOps metrics in Prometheus

### Current cost

```promql
sum(
  cee_current_monthly_cost_usd
)
```

### Optimized cost

```promql
sum(
  cee_optimized_monthly_cost_usd
)
```

### Potential savings

```promql
sum(
  cee_potential_savings_usd
)
```

### Savings percentage

```promql
cee:monthly_savings_percentage
```

### Optimization coverage

```promql
cee:optimization_coverage_percentage
```

---

## 23. Top savings opportunities

Find the workloads with the highest potential savings:

```promql
topk(
  10,
  cee_workload_potential_savings_usd
)
```

Find the most expensive workloads:

```promql
topk(
  10,
  cee_workload_current_monthly_cost_usd
)
```

Find the workloads with the highest savings percentage:

```promql
topk(
  10,
  cee_workload_savings_percentage
)
```

---

## 24. Recommendation metrics

Total recommendations:

```promql
cee_recommendation_count
```

Workloads with recommendations:

```promql
cee_recommendation_workload_count
```

Recommendations by rule:

```promql
sort_desc(
  cee_recommendation_rule_count
)
```

Recommendations by severity:

```promql
sort_desc(
  cee_recommendation_severity_count
)
```

Recommendations by confidence:

```promql
sort_desc(
  cee_recommendation_confidence_count
)
```

---

## 25. Scheduler metrics

Scheduler executions:

```promql
cee_scheduler_runs_total
```

Successful executions:

```promql
cee_scheduler_success_total
```

Failed executions:

```promql
cee_scheduler_failure_total
```

Last successful analysis:

```promql
cee_scheduler_last_success_timestamp
```

Last failed analysis:

```promql
cee_scheduler_last_failure_timestamp
```

Last analysis duration:

```promql
cee_scheduler_last_duration_seconds
```

---

## 26. Scheduler freshness

A stale analysis is detected when more than one hour has passed since the last successful scheduled analysis.

```promql
cee:analysis_stale
```

Expected:

```text
0 = fresh
1 = stale
```

---

## 27. Prometheus alerts

Cloud Efficiency Engine exposes alerts for scenarios such as:

```text
CloudEfficiencyAnalysisStale
CloudEfficiencySchedulerFailures
CloudEfficiencyCriticalRecommendations
CloudEfficiencyHighSavingsOpportunity
CloudEfficiencyHighHTTPErrorRate
```

Verify the rules:

```bash
kubectl get prometheusrules \
  -n monitoring
```

Open:

```text
http://localhost:9090/alerts
```

---

## 28. Grafana

Port-forward Grafana:

```bash
kubectl port-forward \
  -n monitoring \
  svc/monitoring-grafana \
  3000:80
```

Open:

```text
http://localhost:3000
```

Dashboard:

```text
Cloud Efficiency Engine
```

The dashboard should expose:

```text
Current Monthly Cost
Optimized Monthly Cost
Potential Monthly Savings
Savings Opportunity
Workloads
Optimizable Workloads
Recommendations
Scheduler Success Rate
HTTP Error Rate
Potential Savings by Namespace
Savings Percentage by Namespace
Top Savings Opportunities
Most Expensive Workloads
Scheduler Runs
Scheduler Duration
```

---

## 29. Expected business flow

At the end of the demo:

```text
Kubernetes
    |
    | requests
    | usage
    | replicas
    v
Prometheus
    |
    | CEE recording rules
    v
Cloud Efficiency Engine
    |
    +--> Workload analysis
    |
    +--> Historical analysis
    |
    +--> Optimization rules
    |
    +--> Recommendation resolution
    |
    +--> Pricing
    |
    +--> Cost calculation
    |
    v
Potential Savings
    |
    v
Prometheus FinOps Metrics
    |
    v
Grafana
```

---

## 30. Production-oriented properties demonstrated

The demo also validates:

```text
Health checks
Readiness checks
Startup checks
Graceful shutdown
Prometheus scraping
ServiceMonitor
PrometheusRule
Scheduler
FinOps metrics
Workload attribution
Recommendation metrics
Cost attribution
Kubernetes integration
Helm deployment
CI validation
Docker image build
```

---

## 31. Cleanup

Remove the Helm release:

```bash
helm uninstall \
  cloud-efficiency-engine \
  -n cloud-efficiency-engine
```

Delete the namespace:

```bash
kubectl delete namespace cloud-efficiency-engine
```

Delete the Kind cluster:

```bash
kind delete cluster \
  --name cloud-efficiency-engine
```

---

## 32. Demo result

The final demo proves that Cloud Efficiency Engine can:

```text
1. Discover workload metrics through Prometheus.
2. Associate Pods with Kubernetes workloads.
3. Analyze CPU and memory utilization.
4. Detect optimization opportunities.
5. Generate recommendations.
6. Estimate current infrastructure cost.
7. Estimate optimized infrastructure cost.
8. Calculate potential savings.
9. Expose FinOps metrics.
10. Visualize the results in Grafana.
11. Run the analysis automatically.
12. Detect operational failures and stale analysis.
13. Package the application through Helm.
14. Validate the project through CI.
```

The objective is not merely to demonstrate a Go application.

The objective is to demonstrate a complete Kubernetes FinOps optimization workflow.