# Project security best practices

## Snyk MCP availability check

- First, check whether Snyk MCP tools are available in this environment.
- Consider Snyk MCP available only if tools exist whose names match `snyk_*` (for example: `snyk_code_scan`, `snyk_sca_scan`, `snyk_iac_scan`).
- If Snyk MCP tools are available:
  - Inform the user that Snyk MCP is available, running, and will be used to scan AI-generated changes.
  - Apply all rules in the sections below.
- If Snyk MCP tools are NOT available:
  - Proceed normally without prompting, installing, or blocking any action.
  - Do not error, degrade UX, or attempt to install/configure Snyk.

## Rules to enforce when Snyk MCP is available

### When to scan
- Always run the Snyk Code scanning tool for any new or modified first-party application code.
- Always run the Snyk SCA scanning tool when adding, removing, or updating dependencies.
- Always run the Snyk IaC scanning tool for any new or modified Infrastructure-as-Code files (Terraform, OpenTofu, AWS CloudFormation, or Kubernetes manifests).

### How to remediate findings
- If any security issues are found in newly introduced or modified code, dependencies, or IaC:
  - Attempt to fix the issues using the results context from Snyk.
  - Prefer targeted fixes over large refactors or ecosystem-wide upgrades.
  - Do not introduce new dependencies solely to remediate findings unless explicitly requested.
- After applying fixes, re-run the relevant Snyk scan(s) to confirm remediation.
- Repeat scan → fix → rescan until either:
  - No issues remain for the newly introduced or modified artifacts, OR
  - 3 remediation iterations have been performed.

### Stopping conditions
- If issues remain after 3 remediation iterations:
  - Stop remediation.
  - Summarize the remaining issues clearly.
  - Explain why they were not fixed and what the recommended next step is.

### Reporting requirements
- Always inform the user about any changes made while fixing vulnerabilities.
- Always show a diff of the final changes.
