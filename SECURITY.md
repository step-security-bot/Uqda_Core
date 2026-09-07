# Security policy

## Reporting a vulnerability

Please report suspected vulnerabilities through GitHub's **Private vulnerability reporting** feature on the repository's Security tab. Do not open a public issue or disclose exploit details publicly before a fix is available.

Include, when possible:

- the affected version or commit;
- the operating system and architecture;
- steps to reproduce the issue;
- the expected security impact; and
- a minimal proof of concept with secrets and personal data removed.

The maintainers will acknowledge the report, investigate it, and coordinate disclosure and remediation with the reporter. Please allow a reasonable period for a fix before publishing details.

## Supported versions

Security fixes are made on the current `main` branch and included in the next release. Users should run the latest available release.

New nodes remain compatible with protocol 0.5 peers. Secure transcript confirmation is negotiated automatically when both sides support it. For sensitive or controlled overlays, append `?secure=required` to both peering and listener URIs to reject legacy handshakes and prevent downgrade.

Example: `tls://peer.example:9001?secure=required`.

## Release integrity and provenance

Stable releases produced by the current workflow publish three complementary integrity signals:

- `SHA256SUMS`, containing the SHA-256 digest of every release asset;
- `SHA256SUMS.sigstore.json`, a Sigstore bundle containing the keyless signature, signing certificate, and transparency-log proof for the checksum manifest; and
- GitHub artifact attestations that bind the published artifact digests to the repository's release workflow.

The checksum manifest is signed in GitHub Actions with a short-lived OIDC identity. Verification must require both the exact workflow identity and GitHub Actions OIDC issuer:

```bash
cosign verify-blob SHA256SUMS \
  --bundle SHA256SUMS.sigstore.json \
  --certificate-identity "https://github.com/Uqda/Core/.github/workflows/release-beta.yml@refs/heads/main" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com"
```

After the manifest is authenticated, verify a downloaded asset against `SHA256SUMS` before installing it. Users with GitHub CLI can additionally verify build provenance for a downloaded artifact:

```bash
gh attestation verify ./uqda-RELEASE-ASSET -R Uqda/Core
```

The release workflow refuses to publish from any ref other than `refs/heads/main` and refuses to overwrite an existing release. Repository administrators can still edit release metadata or assets manually, so users must verify the signed manifest rather than relying on the release page alone. Repository branch protection is also part of this trust model: direct or insufficiently reviewed modification of `main` or the release workflow would weaken the value of workload-identity signing. Keep `main` protected, require CI before merge, and restrict who can modify release workflows.
