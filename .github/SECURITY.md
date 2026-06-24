# Security Policy

## Supported versions

Security fixes are applied to the **latest release** and `main`. Older tags
are not supported unless noted in a security advisory.

| Version | Supported |
| ------- | --------- |
| latest release | yes |
| `main` | yes |
| older tags | no |

## Reporting a vulnerability

**Please do not open a public GitHub issue for security vulnerabilities.**

Report them privately so we can fix them before details are public:

1. Open a **[GitHub Security Advisory](https://github.com/aleksandarknezevic/chainform/security/advisories/new)**
   (Repository → **Security** → **Report a vulnerability**), or
2. Contact the maintainer via GitHub: [@aleksandarknezevic](https://github.com/aleksandarknezevic)

We aim to acknowledge reports within **72 hours** and share a remediation plan or
fix timeline as soon as we understand the issue.

## What to include

- Description of the vulnerability and impact
- Steps to reproduce (commands, config snippets — **redact RPC URLs and API keys**)
- ChainForm version (`chainform version`) and how it was installed
- Any proof-of-concept you are comfortable sharing

## In scope

Examples of issues we care about:

- Incorrect or unsafe calldata in `plan` / `export` that could lead to
  unintended contract calls when a batch is executed
- Crashes, panics, or memory issues triggered by config or ABI input
- Dependency vulnerabilities in released binaries or the Docker image
- Leakage of secrets (RPC URLs, env vars) in logs or exported artifacts

## Out of scope

- Bugs in **third-party contracts** you point ChainForm at (report to the protocol)
- **Social engineering** or phishing using the ChainForm name
- Issues that require you to execute a malicious Safe batch without reviewing it
  (ChainForm is designed for human/multisig review before execution)
- Missing features (use [feature requests](.github/ISSUE_TEMPLATE/feature_request.yml) instead)

## Safe disclosure

We follow coordinated disclosure: we will work with you on a fix, credit you in
the advisory if you wish (unless you prefer anonymity), and publish details
after a patch is available.

## Security practices for users

ChainForm does **not** hold private keys or broadcast transactions. You remain
responsible for:

- Keeping `RPC_URL` and API keys out of git (`env("RPC_URL")`, `.env` gitignored)
- Reviewing every `plan` and Safe export before multisig execution
- Pinning release binaries or Docker image tags in production
