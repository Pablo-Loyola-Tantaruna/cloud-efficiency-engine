# FinOps Security

## Runtime security

The FinOps control plane is secure by default. Configure `FINOPS_JWT_SECRET` with at least 32 characters.

Bearer tokens must use HS256 and contain:

- `sub`: authenticated actor identifier
- `tenant`: tenant identifier
- `roles`: one or more of `viewer`, `operator`, `approver`, `admin`
- `exp`: expiration timestamp

The API rejects expired tokens, missing subjects, unsupported roles and unexpected signing algorithms.

Local development can explicitly disable authentication with:

```text
FINOPS_AUTH_MODE=disabled
```

Do not use the disabled mode for shared or production environments.

### Authorization

| Operation | Minimum role |
|---|---|
| Analyze | viewer |
| Create action plan | operator |
| Submit action plan | operator |
| Dry run | viewer |
| Approve action plan | approver |
| Execute action | operator |
| Read execution/history/verification/audit | viewer |
| Read recovery | viewer |

`admin` bypasses role checks.

The approval actor is taken from the authenticated `sub` claim. A client-supplied `approvedBy` value is ignored for authenticated requests.

### Tenant isolation

Action plans are stamped with the authenticated `tenant` claim at creation time.

Execution, verification, audit and recovery reads resolve ownership through the parent action plan before returning data.

A resource belonging to a different tenant is returned as not found to avoid cross-tenant existence disclosure.

### Cloud credentials

Cloud credentials are not accepted through FinOps API payloads. Provider SDKs use their configured credential chains. Configure least-privilege permissions for the execution actions supported by each provider.

### HTTP hardening

Authenticated requests receive:

- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `Referrer-Policy: no-referrer`
- `Cache-Control: no-store`

Health, readiness and metrics endpoints remain outside the authenticated control plane so infrastructure probes continue to work.

## CI/CD security gates

Security is treated as a pipeline concern in addition to runtime authorization.

### SAST

**CodeQL** analyzes Go code and publishes findings to GitHub Code Scanning. CodeQL supports Go and can emit SARIF results for code scanning.

Reference: https://docs.github.com/en/code-security/concepts/code-scanning/codeql/codeql-cli

**zizmor** audits GitHub Actions workflows for CI/CD security issues such as excessive permissions, credential persistence and unsafe references.

Reference: https://github.com/zizmorcore/zizmor

### Dependency and filesystem security

**govulncheck** checks Go dependencies against the Go vulnerability database and prioritizes vulnerabilities that affect code paths actually used by the project.

Reference: https://go.dev/doc/tutorial/govulncheck

**Trivy** scans the repository filesystem for vulnerabilities and secrets and scans IaC for misconfigurations. Trivy supports SARIF output for GitHub Code Scanning.

References:
- https://trivy.dev/docs/latest/target/filesystem/
- https://trivy.dev/docs/latest/scanner/misconfiguration/

### Secret scanning

**Gitleaks** scans repository content and history for hardcoded credentials, API keys and other secrets.

Reference: https://github.com/gitleaks/gitleaks

### SBOM

**Syft** generates a CycloneDX SBOM as a workflow artifact. The SBOM gives us a concrete dependency inventory that can be used for vulnerability response, compliance and downstream security tooling.

Reference: https://github.com/anchore/sbom-action

### Container security

The security workflow builds the application image and runs Trivy vulnerability, secret and misconfiguration scanning against the exact image built from the commit. High and critical findings are blocking unless explicitly handled through the scanner policy.

### JFrog Xray

JFrog Xray is supported as an optional enterprise integration, not as a required dependency of this repository. When `JF_URL` and `JF_XRAY_ENABLED=true` are configured, CI can use the official JFrog CLI setup action and run an Xray source scan.

Xray provides artifact/container security analysis, SBOM capabilities and policy-based vulnerability/compliance scanning.

References:
- https://docs.jfrog.com/security/docs/sbom
- https://docs.jfrog.com/security/docs/xray-docker-containers
- https://github.com/jfrog/setup-jfrog-cli

The integration is intentionally conditional because Xray requires a configured JFrog Platform/Artifactory environment. The repository does not fake or vendor a JFrog service just to make CI green.

### Dependency automation

Dependabot is configured for:

- Go modules
- GitHub Actions
- Docker

This keeps the security tooling and application dependencies continuously maintainable rather than relying on periodic manual upgrades.

## Deferred controls

Rate limiting is intentionally not part of this feature yet. It will be added when runtime traffic and usage characteristics justify a concrete policy. Redis remains focused on cache and execution coordination.
