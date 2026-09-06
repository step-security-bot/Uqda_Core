# Windows release signing

UQDA stable Windows releases must be Authenticode-signed with a publicly trusted
certificate. Signing only the MSI is not sufficient: `uqda.exe` and
`uqdactl.exe` are signed first, those exact binaries are embedded in the MSI,
and the completed MSI is signed afterwards. The stable-release workflow refuses
to publish Windows assets when signing is unavailable, rejected, or invalid.

Checksums, GitHub artifact attestations, and Sigstore prove release provenance,
but Windows application-control policies evaluate the embedded Authenticode
signature on the executable or installer they launch.

## Selected service: SignPath Foundation

UQDA uses the free SignPath Foundation program for qualifying open-source
projects. Microsoft lists SignPath Foundation as an open-source code-signing
option. The certificate and private key are managed by SignPath in an HSM, so
UQDA does not need Microsoft Artifact Signing, a PFX file, a USB token, or a
Microsoft Azure account.

The Windows publisher shown to users is **SignPath Foundation**, which vouches
that the signed files came from UQDA's public repository and trusted GitHub
Actions build. Acceptance is subject to SignPath Foundation review and is not
automatic.

- Apply: https://signpath.org/apply
- Program conditions: https://signpath.org/terms.html
- GitHub integration: https://docs.signpath.io/trusted-build-systems/github
- Public policy: [UQDA code signing policy](CODE_SIGNING_POLICY.md)

## One-time enrollment

Only a UQDA repository owner can complete the external enrollment and approve
signing requests.

1. Apply to SignPath Foundation with `https://github.com/Uqda/Core`.
2. After approval, enable multi-factor authentication for the SignPath account
   and install the SignPath GitHub App for `Uqda/Core`.
3. In SignPath, add the predefined **GitHub.com** trusted build system and link
   it to the UQDA project.
4. Create two artifact configurations:
   - one ZIP-root configuration that signs `uqda.exe` and `uqdactl.exe`;
   - one ZIP-root configuration that signs the generated `.msi`.
   GitHub's upload-artifact action stores each submission as a ZIP, so both
   configurations must use a `zip-file` root. Require SHA-256 Authenticode
   signing, RFC 3161 timestamping, product name **UQDA Core**, and one consistent
   product version.
5. Create a production signing policy restricted to this repository, the stable
   release workflow, the `main` branch, and GitHub-hosted runners. Foundation
   releases require manual approval by an authorized UQDA approver.
6. Create an API token for a SignPath user that has submitter access to that
   policy.

Do not store a certificate or private key in GitHub.

## GitHub Actions settings

Add one repository **Actions secret**:

- `SIGNPATH_API_TOKEN`

Add these repository **Actions variables**, using the exact values configured
by SignPath:

- `SIGNPATH_ORGANIZATION_ID`
- `SIGNPATH_PROJECT_SLUG`
- `SIGNPATH_SIGNING_POLICY_SLUG`
- `SIGNPATH_EXECUTABLES_ARTIFACT_CONFIGURATION_SLUG`
- `SIGNPATH_MSI_ARTIFACT_CONFIGURATION_SLUG`

The workflow intentionally fails before building release assets if any value is
missing. The unsigned intermediate GitHub artifacts are clearly named
`internal-unsigned-*`, retained for one day, and are never published in the
stable release.

## Release gate

For every x64, x86, and ARM64 Windows build, the stable-release workflow:

1. builds `uqda.exe` and `uqdactl.exe` on a GitHub-hosted Windows runner;
2. uploads those executables to the linked SignPath trusted build;
3. waits for SignPath approval/signing and downloads the signed files;
4. rejects either executable unless PowerShell reports a valid, timestamped
   Authenticode signature;
5. embeds those exact signed executables in the MSI without rebuilding;
6. submits the MSI to SignPath and validates its returned signature/timestamp;
7. silently installs the signed x64 MSI, checks files, configuration, machine
   `PATH`, service startup, version output, and `uqdactl getSelf`; and
8. uninstalls it and verifies removal of the service, binaries, and `PATH`
   entry while preserving the node identity.

The publish job cannot run unless all three signed Windows jobs and every other
platform job succeed. Never bypass this gate.

## Independent verification

After downloading a release:

```powershell
$msi = Get-Item .\uqda-*-x64.msi
Get-AuthenticodeSignature $msi.FullName |
  Format-List Status, StatusMessage, SignerCertificate, TimeStamperCertificate
```

`Status` must be `Valid`, the signer must be SignPath Foundation, and a
timestamp certificate must be present.
