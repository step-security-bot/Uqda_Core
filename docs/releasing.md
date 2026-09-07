# Publishing a stable release

The release version is selected by a versioned release-notes file. Package metadata and documentation must be updated in the same pull request.
No Apple Developer account is required for an explicitly unsigned macOS release.

## Normal release path

1. Choose a stable SemVer tag such as `v0.1.2`.
2. Finish the code, tests, documentation, and `CHANGELOG.md` changes in a pull
   request.
3. Add `.github/releases/v0.1.2.md` with the user-facing release notes. The filename is the release workflow's version source.
4. Update `Casks/uqda.rb` to the same bare version and set its SHA-256 to the current `install.sh` digest.
5. Confirm README, verification examples, and package metadata do not claim support or signatures that the workflow does not validate.
6. Merge only after all required checks pass.
7. The `Stable Release` workflow selects the newest stable release-notes file,
   reruns the release quality gate, builds every platform package, creates and
   signs `SHA256SUMS`, creates provenance attestations, and publishes the GitHub
   release and tag.
8. Verify that the workflow and published release succeeded before announcing
   the version.

The publisher refuses to overwrite an existing release. Editing old notes will
therefore never silently replace published assets.

## Manual run or retry

Open **Actions → Stable Release → Run workflow**. Enter the exact tag, or leave
the tag blank to select the newest stable release-notes file. A retry uses the
same full quality gate and still refuses to overwrite an existing release.

## macOS without a paid Apple account

When Apple credentials are absent, the workflow publishes explicitly named
`*-unsigned.pkg` files and all normal checksum, Sigstore, and provenance
verification still applies. If paid Apple credentials are added later, the
same workflow automatically signs and notarizes the macOS binaries and
installer. Never disable Gatekeeper globally.

## Homebrew Cask

`Casks/uqda.rb` pins the stable version and the SHA-256 of that version's
`install.sh`. The installer selects the native macOS architecture and verifies
the downloaded package against `SHA256SUMS`. Every new stable release therefore
requires the Cask version to be updated; update its checksum whenever
`install.sh` changes. The Cask CI job checks this metadata as well as syntax and style.
