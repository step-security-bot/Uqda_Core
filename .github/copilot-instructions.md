# UQDA Core repository instructions

- UQDA Core is a Go implementation of an encrypted, self-organizing IPv6
  overlay. Preserve compatibility with protocol 0.5 peers unless the change is
  explicitly a coordinated protocol migration.
- Use Go 1.25.13 or newer. Run `gofmt`, `go vet ./...`, `go test ./...`, and
  `go build ./...` for Go changes.
- Add focused tests for behavior changes. Keep operating-system code behind the
  correct Go build constraints and exercise affected package/install workflows.
- Treat node private keys, peer credentials, public IP addresses, and live
  configurations as secrets. Never add them to code, examples, tests, logs, or
  pull-request text.
- Keep the Windows service configuration under `%ProgramData%\UQDA`, macOS and
  generic Unix configuration under `/etc`, and packaged platform overrides
  consistent with `docs/PROJECT_GUIDE.md`.
- Stable release artifacts are built only by `.github/workflows/release-beta.yml`.
  Do not weaken its checksums, Sigstore identity constraints, attestations,
  Authenticode gates, or main-branch restriction.
- Update English and Arabic user documentation for user-visible command,
  installation, or operational changes.
