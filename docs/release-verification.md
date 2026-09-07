# Verifying UQDA releases

Authenticate the release manifest and verify each selected UQDA asset against it before installation. A checksum by itself only detects accidental corruption or a mismatch with the published manifest; it does not prove who published that manifest.

## Trust signals

Each stable release produced by the current workflow publishes:

- `SHA256SUMS` — SHA-256 digests for release assets;
- `SHA256SUMS.sigstore.json` — a Sigstore bundle for the checksum manifest; and
- GitHub artifact attestations for the artifacts listed in `SHA256SUMS`.

The Sigstore signature is keyless. GitHub Actions obtains a short-lived OIDC identity, Sigstore issues a short-lived signing certificate, and the bundle records the signature and transparency-log material. There is no long-lived release private key stored in this repository.

## Verify before running the installer

Set the release tag and download the installer plus its verification metadata:

```bash
TAG=v0.1.9 # replace with the release you are installing
BASE="https://github.com/Uqda/Core/releases/download/$TAG"

curl -fSLO "$BASE/SHA256SUMS"
curl -fSLO "$BASE/SHA256SUMS.sigstore.json"
curl -fSLO "$BASE/install.sh"
```

Verify that the checksum manifest was signed by the release workflow on `main`:

```bash
cosign verify-blob SHA256SUMS \
  --bundle SHA256SUMS.sigstore.json \
  --certificate-identity "https://github.com/Uqda/Core/.github/workflows/release-beta.yml@refs/heads/main" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com"
```

Only after that succeeds, authenticate the installer against the signed manifest:

```bash
grep -E '  (\./)?install\.sh$' SHA256SUMS > install.sha256
sha256sum -c install.sha256
sudo sh install.sh --version "$TAG"
```

On macOS, use `shasum -a 256` to compare the downloaded file with the digest in `SHA256SUMS` if `sha256sum` is unavailable.

## Verify build provenance

GitHub CLI can independently verify the provenance attestation associated with a downloaded artifact:

```bash
gh attestation verify ./uqda-v0.1.9-linux-amd64.tar.gz -R Uqda/Core
```

This checks that GitHub has a valid signed attestation for the artifact digest and that it belongs to `Uqda/Core`.

## What this protects against

The signed manifest lets users detect a replaced asset or rewritten `SHA256SUMS` unless the attacker can also produce a valid workflow identity. The GitHub attestation provides a separate provenance record for the artifact digest.

This does **not** make repository governance irrelevant. If an attacker can directly modify `main` and the release workflow, they may be able to execute a workflow under the repository's own GitHub OIDC identity. For that reason, protect `main`, require successful CI before merge, and tightly control changes to `.github/workflows/release-beta.yml`.

## Mirrors and forks

A mirror can redistribute the release assets and Sigstore bundle without changing them; verification still binds the manifest to the original `Uqda/Core` release workflow. A fork that wants to publish independently should use its own workflow identity and document that identity explicitly rather than weakening the verifier with a broad regular expression.
