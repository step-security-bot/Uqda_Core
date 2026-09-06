# Windows release signing

UQDA stable Windows releases must be Authenticode-signed with a publicly trusted
certificate. Signing only the MSI is not sufficient: `uqda.exe` and
`uqdactl.exe` are signed first, those exact binaries are embedded in the MSI,
and the finished MSI is signed afterwards. The stable-release workflow refuses
to publish Windows assets when signing is unavailable or signature validation
fails.

This is required for Windows Smart App Control. Checksums and Sigstore attest
that a release came from the expected GitHub workflow, but Windows application
control evaluates the Authenticode signature on the executable it launches.

## Obtain a Microsoft Artifact Signing certificate

Microsoft renamed Trusted Signing to **Artifact Signing**. Use a **Public
Trust** production certificate profile; Private Trust and Public Trust Test are
not trusted by ordinary Windows computers.

1. Create or select an Azure subscription and Microsoft Entra tenant. The legal
   name and address in Azure billing must match the identity that will appear on
   the certificate.
2. Register the `Microsoft.CodeSigning` resource provider.
3. Create an Artifact Signing account in a supported region. A Basic account is
   sufficient for this release pipeline unless the project needs Premium
   capacity.
4. In the account's **Access control (IAM)**, give the maintainer performing
   identity verification the `Artifact Signing Identity Verifier` role.
5. Under **Identity validations**, create a **Public** identity validation and
   complete the legal-identity and email checks. Microsoft states that an
   organization validation can take 1–20 business days. Individual Public
   Trust enrollment is currently limited to developers in the United States
   and Canada; organizations are supported in the countries listed in
   Microsoft's current quickstart.
6. After the validation status is `Completed`, create a certificate profile of
   type **Public Trust**, for example `UqdaPublicRelease`.

See Microsoft's official
[Artifact Signing setup guide](https://learn.microsoft.com/azure/artifact-signing/quickstart).

Never create or upload a PFX for this path. Microsoft stores and rotates the
signing key in its managed HSM; GitHub receives only a short-lived OIDC token.

## Connect GitHub Actions without a client secret

1. In Microsoft Entra ID, create an application registration and its service
   principal.
2. Add a federated credential for GitHub Actions. Restrict it to this repository
   and the `main` branch. The subject is:

   ```text
   repo:Uqda/Core:ref:refs/heads/main
   ```

3. On the Artifact Signing account (or, more narrowly, its certificate
   profile), assign the service principal the
   `Artifact Signing Certificate Profile Signer` role.
4. Add these GitHub **Actions secrets** under
   `Uqda/Core > Settings > Secrets and variables > Actions`:

   - `AZURE_CLIENT_ID`
   - `AZURE_TENANT_ID`
   - `AZURE_SUBSCRIPTION_ID`

5. Add these GitHub **Actions variables**:

   - `ARTIFACT_SIGNING_ENDPOINT` — the endpoint for the selected Azure region,
     such as `https://weu.codesigning.azure.net/` for West Europe.
   - `ARTIFACT_SIGNING_ACCOUNT_NAME`
   - `ARTIFACT_SIGNING_CERTIFICATE_PROFILE_NAME`

The workflow has only `contents: read` and `id-token: write` while signing. Do
not add `AZURE_CLIENT_SECRET`; OIDC makes a stored client secret unnecessary.

See Microsoft's official
[Artifact Signing GitHub Action and OIDC guide](https://github.com/Azure/artifact-signing-action).

## Release gate and installation test

For each x64, x86, and ARM64 Windows build, the stable-release workflow:

1. builds `uqda.exe` and `uqdactl.exe`;
2. signs and timestamps both executables;
3. rejects either executable if PowerShell does not report `Valid`;
4. packages the already-signed files without rebuilding them;
5. signs and timestamps the MSI;
6. rejects the MSI if its signature or timestamp is invalid; and
7. on x64, silently installs the MSI, checks files, configuration, machine
   `PATH`, service startup, version output, and `uqdactl getSelf`, then
   uninstalls it and verifies that the service, binaries, and PATH entry were
   removed while the node identity was preserved.

Do not publish a Windows package by bypassing this job. A successful build of
an unsigned MSI is useful only for pull-request testing, not for end users.

After downloading a published release, a user can independently verify it:

```powershell
$msi = Get-Item .\uqda-*-x64.msi
Get-AuthenticodeSignature $msi.FullName | Format-List Status,StatusMessage,SignerCertificate
```

`Status` must be `Valid`, and the publisher displayed by Windows must match the
validated identity chosen for the UQDA certificate profile.
