# Windows installation, coexistence and trust

## Development changes (not yet a stable release)

Configuration normalization preserves explicit `AdminListen`, including `none`.
Existing persistent configurations without a usable identity are rejected; they
must not acquire a new random identity on restart. TCP administration now uses
mutual TLS with the local node identity: update `uqda.exe` and `uqdactl.exe`
together. Keep private keys local and protected; no Yggdrasil peering changes
are required. See [administration authentication](admin-api.md).

New MSI authoring reserves 100 build numbers per SemVer patch: beta.1..49 map
to 1..49, rc.1..49 to 50..98 and stable to 99. For example, 0.1.11-beta.1,
0.1.11-rc.1 and 0.1.11 map to 0.1.1101, 0.1.1150 and 0.1.1199.
Major/minor are limited to 255 and the SemVer patch to 654. Binary and filename
versions retain SemVer. Never reissue a published stable tag with this scheme.

Historical MSI migration remains a release blocker: the legacy service guard
must not be removed without proving ownership, configuration retention,
rollback and non-interference with Yggdrasil using historical installer fixtures.
The tests below do not constitute that acceptance. Do not delete a user's
configuration as an upgrade strategy.

## Release boundary

The installer isolation changes start with v0.1.10. They do not repair already
downloaded v0.1.9 MSI files. Do not relabel old files as fixed or signed.

UQDA used upstream MSI family/component identifiers and the upstream requested
Wintun adapter GUID. This was a packaging/isolation defect. The new package uses
a UQDA-only UpgradeCode and path-derived component identities, a UQDA-only adapter
GUID, and a Windows administration default of `tcp://localhost:19001`.

Yggdrasil's default local administration port remains 9001. This change separates
local administration; it does not promise that simultaneous overlay IP routing
on the same host is conflict-free. Both applications can install routes covering
overlapping address space. Test your intended routing before production use.

The TUN startup path no longer calls `wintun.Uninstall()` when adapter creation
fails. Wintun is shared driver infrastructure, not disposable UQDA-only state.

## Existing UQDA installations

The old UpgradeCode is shared with another product. The new MSI deliberately
does not perform an automatic upgrade using that old family identifier.

If an existing UQDA service is found without a matching new-family installation,
setup stops with a migration message BEFORE modifying the installation:

1. Back up `%ProgramData%\UQDA` securely, including any externally referenced
   private-key file. Do not publish the configuration or its backups.
2. In Installed apps, select **UQDA**, not Yggdrasil, and uninstall UQDA normally.
   Normal removal preserves the configuration and node identity.
3. Install the new MSI and open a new elevated PowerShell window.
4. Run `uqdactl.exe getSelf` and compare the public key to the previous identity.

If old UQDA and Yggdrasil were already installed over each other, their shared
MSI registrations may already be damaged. Inspect the installed products and
backups before removal; the new installer cannot retrospectively repair those
registrations safely. Do not bulk-delete MSI registry keys or the shared driver.

Updates within the new UQDA family use the normal MSI major-upgrade path and
preserve existing configuration. Downgrades are rejected.

An explicit `AdminListen` in an existing config is preserved. If it still selects
9001 and another application owns that endpoint, change **only** that field to
`tcp://localhost:19001` after backup. Never bind the unauthenticated admin API to
`0.0.0.0` or a public interface. Omitted values use the new default. `uqdactl`
reads an explicitly configured endpoint from the platform config file.

## Setup and error handling

MSI runs configuration preparation as the privileged installer process, using
the system command interpreter with AutoRun disabled and fully quoted paths.
New config is generated to a temporary file, parsed before being moved into
place, and protected by a SYSTEM/Administrators directory ACL. Existing config
is not regenerated, including on repair. A parse/key-loading failure aborts
before StartServices and leaves `install-config-error.log` beside the config.
This is not full semantic validation of every peer URI or runtime resource.

There are different classes of failure:

| Message | Meaning | Evidence to inspect |
| --- | --- | --- |
| Service failed to start / MSI 1920 | Service startup failed; not proof of bad admin credentials | UQDA log, Service Control Manager events and verbose MSI log |
| Windows protected your PC / unknown publisher | SmartScreen/publisher reputation | Authenticode status of the actual MSI and executables |
| Application control policy blocked this file | WDAC, AppLocker or Smart App Control decision | CodeIntegrity/AppLocker events; policy owner's review |

Read-only diagnostics in elevated PowerShell:

```powershell
Get-Service UQDA,Yggdrasil -ErrorAction SilentlyContinue
Get-CimInstance Win32_Service -Filter "Name='UQDA'" |
    Select-Object Name,State,StartName,PathName,ExitCode
Get-NetTCPConnection -State Listen -ErrorAction SilentlyContinue |
    Where-Object LocalPort -in 9001,19001 |
    Select-Object LocalAddress,LocalPort,OwningProcess
Get-Content "$env:ProgramData\UQDA\uqda.log" -Tail 100 -ErrorAction SilentlyContinue
Get-WinEvent -FilterHashtable @{
    LogName='System'; ProviderName='Service Control Manager'
    StartTime=(Get-Date).AddHours(-1)
} -ErrorAction SilentlyContinue | Select-Object TimeCreated,Id,Message
```

To inspect publisher trust, use `Get-AuthenticodeSignature` on the exact downloaded
MSI, installed `uqda.exe`, `uqdactl.exe`, and `wintun.dll`. A signed Wintun DLL
does not sign the other files. A SHA256 digest or GitHub verified commit badge
is not Windows Authenticode. Source ZIPs cannot establish the signature status
of a separately distributed binary.

For the specific upstream `yggdrasil-0.5.14-x64.msi`, our Windows comparison
[run](https://github.com/Uqda/Core/actions/runs/34249860549/job/102141439813)
verified SHA256 `8a871a037040094c793b08447e98815ff90040fa6c4c0d094b64f7f8e3f3a93d`
and `Get-AuthenticodeSignature` reported `NotSigned`. It nevertheless installed
successfully on the hosted runner. This result concerns that MSI, not every
upstream executable or architecture. Absence of a warning on another computer
is therefore not evidence of a special upstream signing technique.

## No guarantee of warning-free installation

[Microsoft documents](https://learn.microsoft.com/en-us/windows/apps/package-and-deploy/smartscreen-reputation)
both file and publisher reputation. Even a correctly signed new binary can show
SmartScreen warnings. Self-signing does not give public publisher trust.
[Smart App Control](https://learn.microsoft.com/en-us/windows/apps/develop/smart-app-control/code-signing-for-smart-app-control)
requires certificates from trusted providers for its signature checks.

Use the existing [optional signing integration](windows-release-signing.md) after
the project has an approved signing identity. No installer fix, filename change,
or CI pass can manufacture that identity or guarantee acceptance by all policies.
Do not disable Windows protection or import a self-issued root as a workaround.

## Test scope

The dedicated Windows CI workflow builds two test-only versions and checks x64
installation alongside the SHA256-pinned Yggdrasil 0.5.14 MSI, separate admin
queries/adapter identities, upgrade and identity preservation, invalid-config
repair, legacy-service refusal, and removal without stopping upstream's service.
Logs are retained without collecting private configurations. A clean hosted
runner does not reproduce every user's Windows policy or driver state. The
legacy guard test uses a simulated service, not every historical MSI version.
No end-to-end simultaneous overlay routing guarantee is implied by this test.
