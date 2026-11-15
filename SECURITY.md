# Security policy

## Report vulnerabilities privately

Use **Report a vulnerability** in this repository's [GitHub Security tab](https://github.com/geobattles/geobattles-backend/security). A GitHub account is required; do not create a public issue or pull request containing an exploit, private logs or credentials.

Include the affected commit/image version, a concise impact description and the smallest safe reproduction. Remove API keys, JWTs, cookies, passwords, personal player data and credential-bearing URLs. Do not test against production or spend provider quota without prior authorization. Coordinate disclosure privately; no response-time or bounty promise is made.

If a credential may have leaked, notify the maintainers privately and revoke/rotate it through its provider. Deleting it from the latest source does not invalidate it or remove earlier Git history.

## Supported scope

Security fixes target the current maintained default branch and reviewed releases. Historical development snapshots and old container tags have no separate support commitment. A build, dependency audit or secret scan is not a comprehensive security certification.

## Maintainer safeguards

- Keep secret scanning and push protection enabled; do not bypass a finding without privately validating it.
- Review changes through pull requests and required checks. Configure branch protection to require review and resolved conversations, dismiss stale reviews and prevent ordinary force pushes/deletion. An available administrator bypass is not automatic permission to bypass review.
- Validate with no production secrets/provider calls. Image publication remains manual/master-only and separately approved. Ensure required checks match the validation workflow available on the branch.
- Do not attach raw build/browser logs, resolved Compose configuration, environment files or private evidence. Restrict browser Maps keys by API/referrer rather than treating them as secrets.
