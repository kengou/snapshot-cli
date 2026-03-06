# Security Policy

## Supported versions

| Version | Supported |
|---------|-----------|
| latest (`main`) | Yes |
| older tags | No |

Only the latest release on `main` receives security fixes.

## Reporting a vulnerability

**Please do not open a public GitHub issue for security vulnerabilities.**

Report vulnerabilities privately via [GitHub Security Advisories](https://github.com/kengou/snapshot-cli/security/advisories/new).

Include in your report:
- A description of the vulnerability and its potential impact
- Steps to reproduce or a proof-of-concept
- Affected versions
- Any suggested fix, if available

You will receive an acknowledgement within **5 business days**.
If the vulnerability is confirmed, a fix will be prioritised and a CVE requested where appropriate.
We aim to publish a patched release within **30 days** of confirmation.

## Security considerations

- Credentials are read exclusively from `OS_*` environment variables — never passed on the command line or stored on disk.
- The container image uses `distroless/static` and runs as non-root (UID 65532) to minimise the attack surface.
- All GitHub Actions dependencies are pinned to exact commit SHAs.
- Container images are signed with [cosign](https://github.com/sigstore/cosign) on every release.
