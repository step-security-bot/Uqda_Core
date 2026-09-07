# Code signing policy

The stable-release workflow supports optional Windows Authenticode signing
through [SignPath.io](https://signpath.io/) and the
[SignPath Foundation](https://signpath.org/) program. Enrollment and signing
credentials are external prerequisites; their presence must not be inferred
from this repository.

The release filename is authoritative:

- `*-unsigned.msi` has no Authenticode publisher signature;
- an MSI without the `-unsigned` suffix must pass the workflow's signature and
  timestamp checks before publication.

## Scope

If signing is enabled, only Windows artifacts built from the public
[Uqda/Core](https://github.com/Uqda/Core) repository by the protected stable
release workflow may be submitted:

- `uqda.exe`;
- `uqdactl.exe`; and
- the UQDA MSI containing those exact executables.

The workflow validates returned signatures before packaging or publication.
Unsigned SignPath submission artifacts are short-lived workflow artifacts and
are not stable release downloads.

## Roles and approval

The repository documents these intended roles:

- committer and reviewer: [maher-xs](https://github.com/maher-xs);
- signing approver: [Uqda organization owners](https://github.com/orgs/Uqda/people?query=role%3Aowner).

Actual SignPath enrollment, access controls, and production approval state are
managed outside this repository. A release must not claim to be signed until
the published files pass independent Authenticode verification.

## Privacy and network behavior

UQDA has no telemetry client or UQDA-owned coordination service in the current
codebase. It communicates with configured peers and listeners, optionally
discovers local peers through multicast, and carries operator traffic through
the selected encrypted overlay.

The security boundaries and operational behavior are documented in the
[project guide](PROJECT_GUIDE.md) and [security policy](../SECURITY.md).

## Verification and incident response

For an explicitly unsigned MSI, authenticate the release manifest with
Sigstore and compare the MSI with its SHA-256 entry before installation. Local
Windows application-control policy may still reject it.

For a normally named signed MSI, additionally require
`Get-AuthenticodeSignature` to report `Valid`, inspect the signer, and confirm
a timestamp certificate is present.

Report suspected compromised releases or signing-policy violations privately
as described in [SECURITY.md](../SECURITY.md). Maintainers should pause
publication, investigate the originating workflow and any signing request, and
request certificate revocation when a signing key or certificate is affected.
