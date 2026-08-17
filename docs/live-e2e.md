# Live FinOps E2E

The live E2E suite validates the complete FinOps execution lifecycle against a real EKS, AKS, or GKE node group.

## What it validates

For one real node group / node pool, the test:

1. Reads the current desired capacity.
2. Builds a `READY_TO_APPLY` action plan with explicit approval metadata and savings attribution.
3. Executes the reduction through the real provider SDK adapter.
4. Verifies the real provider state at the desired capacity.
5. Replays the same action and verifies idempotency: the same execution record is reused instead of issuing another mutation.
6. Restores the original capacity in a cleanup step.

The execution path records runtime metrics for execution, provider latency, verification, and realized savings.

## Safety

Live mutation is never enabled by the normal CI pipeline. The tests require:

```text
FINOPS_E2E_ALLOW_MUTATION=true
```

The test also refuses to reduce a node group / node pool below one node.

Run only against a disposable or explicitly approved environment. The cleanup restore runs from a test `defer`, but cloud API failures can still leave infrastructure at the reduced size, so the target must be operationally safe to mutate.

## Local execution

AWS requires the provider build tag already used by the repository's tagged AWS build:

```bash
git fetch origin feature/finops-observability
git checkout feature/finops-observability

FINOPS_E2E_ALLOW_MUTATION=true \
AWS_E2E_REGION=us-east-1 \
AWS_E2E_CLUSTER=my-cluster \
AWS_E2E_NODE_GROUP=my-node-group \
go test -tags "live_e2e aws_sdk_v2" ./test/e2e \
  -run TestLiveAWSEKSFinOpsLifecycle -count=1 -v
```

Azure:

```bash
FINOPS_E2E_ALLOW_MUTATION=true \
AZURE_E2E_SUBSCRIPTION_ID=... \
AZURE_E2E_RESOURCE_GROUP=... \
AZURE_E2E_CLUSTER=my-cluster \
AZURE_E2E_NODE_POOL=my-node-pool \
go test -tags live_e2e ./test/e2e \
  -run TestLiveAzureAKSFinOpsLifecycle -count=1 -v
```

GCP:

```bash
FINOPS_E2E_ALLOW_MUTATION=true \
GCP_E2E_PROJECT_ID=my-project \
GCP_E2E_LOCATION=us-central1 \
GCP_E2E_CLUSTER=my-cluster \
GCP_E2E_NODE_POOL=my-node-pool \
go test -tags live_e2e ./test/e2e \
  -run TestLiveGCPGKEFinOpsLifecycle -count=1 -v
```

Credentials are resolved through the standard SDK credential chains used by the provider adapters.

## GitHub Actions

The repository contains a manual workflow:

```text
Actions → Live FinOps E2E → Run workflow → provider
```

Required AWS configuration:

- secret `FINOPS_E2E_AWS_ROLE_ARN`
- variable `FINOPS_E2E_AWS_REGION`
- variable `FINOPS_E2E_AWS_CLUSTER`
- variable `FINOPS_E2E_AWS_NODE_GROUP`

Required Azure configuration:

- secret `FINOPS_E2E_AZURE_CLIENT_ID`
- secret `FINOPS_E2E_AZURE_TENANT_ID`
- secret `FINOPS_E2E_AZURE_SUBSCRIPTION_ID`
- variable `FINOPS_E2E_AZURE_RESOURCE_GROUP`
- variable `FINOPS_E2E_AZURE_CLUSTER`
- variable `FINOPS_E2E_AZURE_NODE_POOL`

Required GCP configuration:

- secret `FINOPS_E2E_GCP_WORKLOAD_IDENTITY_PROVIDER`
- secret `FINOPS_E2E_GCP_SERVICE_ACCOUNT`
- variable `FINOPS_E2E_GCP_PROJECT_ID`
- variable `FINOPS_E2E_GCP_LOCATION`
- variable `FINOPS_E2E_GCP_CLUSTER`
- variable `FINOPS_E2E_GCP_NODE_POOL`

The workflow is intentionally `workflow_dispatch` only. It never runs from ordinary pushes or pull requests.
