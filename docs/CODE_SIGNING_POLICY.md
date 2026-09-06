# Code signing policy

Free code signing is provided by [SignPath.io](https://signpath.io/);
the certificate is provided by
[SignPath Foundation](https://signpath.org/).

## Scope

Only Windows release artifacts built from the public
[Uqda/Core](https://github.com/Uqda/Core) source repository by the protected
stable-release GitHub Actions workflow may be signed. The signed scope is:

- `uqda.exe`;
- `uqdactl.exe`; and
- the UQDA MSI installers that contain those signed executables.

Every release must pass the repository's required tests and supply-chain gates.
SignPath independently verifies the GitHub build origin. Each production signing
request requires manual approval. Unsigned intermediate artifacts are never
published as stable release downloads.

## Team roles

- Committer and reviewer: [maher-xs](https://github.com/maher-xs)
- Signing approver: [Uqda organization owners](https://github.com/orgs/Uqda/people?query=role%3Aowner)

Repository access and SignPath access require multi-factor authentication.
Changes to the signing workflow or this policy must use a protected pull
request.

## Privacy and network behavior

UQDA does not collect or transmit telemetry. It communicates with networked
systems only when the operator requests that behavior through configured peers,
listeners, or local multicast discovery. UQDA routes operator traffic through
the selected encrypted overlay; it does not send that traffic to a UQDA-owned
central service.

The project's security boundaries and operational behavior are documented in
the [project guide](PROJECT_GUIDE.md) and [security policy](../SECURITY.md).

## Verification and incident response

Users must verify that a downloaded MSI has a valid Authenticode signature from
SignPath Foundation and a timestamp, and should also verify the published
SHA-256 checksum and GitHub artifact attestation.

Suspected compromised releases or signing-policy violations must be reported
privately as described in [SECURITY.md](../SECURITY.md). Maintainers will pause
releases, investigate the originating build and SignPath request, and request
certificate revocation when required.
