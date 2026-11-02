# Security Policy

The NimmaSaarthi project takes platform integrity, user trust, and operational safety seriously.

---

## Scope

This policy covers the application source code, platform infrastructure, deployment configuration, data handling, and related operational workflows.

---

## Security principles

- protect rider, driver, and operator identity data
- minimize unnecessary exposure of personal or location data
- maintain clear audit trails for safety incidents and payments
- ensure critical operational decisions are governed by policy and review
- treat safety and fraud prevention as first-class concerns, not afterthoughts

---

## Supported environments

The project should be operated in secure, controlled environments corresponding to the deployment configuration under `deploy/` and `infrastructure/`.

---

## Reporting a vulnerability

If you believe you have found a security issue, please report it privately and responsibly.

Recommended process:

1. Do not disclose details publicly.
2. Open a private security advisory or contact the repository maintainers through the official supported channel.
3. Include a concise description, affected component, reproduction steps, and risk level.
4. Allow time for triage, remediation, and coordinated disclosure.

The maintainers will review the report and work toward a fix with clear communication and disclosure timing.

---

## Response expectations

Security reports are handled with urgency, and high-impact issues should be prioritized as a platform safety concern. We expect contributors to avoid accidental data exposure, abuse vectors, or unsafe operational behavior.

---

## Safe development practices

- validate authentication and authorization boundaries
- check route and payment flows for abuse conditions
- avoid storing sensitive data in logs or unguarded configuration files
- treat safety incidents and audit trails as operationally sensitive records
- keep secrets and credentials out of source control and deployment templates