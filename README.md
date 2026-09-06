<div align="center">

<img src="contrib/logo/uqda-logo.png" alt="UQDA" width="720">

# UQDA Core

**An encrypted, self-organizing IPv6 overlay network**

[![CI](https://github.com/Uqda/Core/actions/workflows/ci.yml/badge.svg)](https://github.com/Uqda/Core/actions/workflows/ci.yml)
[![Go version](https://img.shields.io/badge/Go-1.25.13%2B-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Go Report Card](https://goreportcard.com/badge/github.com/Uqda/Core)](https://goreportcard.com/report/github.com/Uqda/Core)
[![License](https://img.shields.io/badge/License-LGPLv3-blue.svg)](LICENSE)

**English** · [العربية](README_AR.md)

</div>

> **Uqda** (Arabic: **عُقَد**, “nodes” or “knots”) is a userspace router for creating encrypted IPv6 networks over existing IPv4 or IPv6 links.

## Project status

**The latest stable UQDA release is recommended for supported platforms.** The project has not been independently security-audited and should not be treated as an anonymity system. Use an IPv6 firewall and avoid exposing services that should not be reachable by other network participants.

Read the [complete project guide](docs/PROJECT_GUIDE.md) for the architecture,
identity and addressing model, routing, cryptography, security boundaries,
configuration, every supported installation path, operation, and development.

Want one box to serve every phone and computer at home? Follow the
[UQDA home gateway guide](docs/HOME_GATEWAY.md) for Raspberry Pi OS,
Debian/Ubuntu appliances, and the experimental OpenWrt profile.
For a public visitor hotspot, use the hardened
[UQDA café gateway profile](docs/CAFE_GATEWAY.md).
For latency targets, peer selection, p95 measurement, and bufferbloat diagnosis,
read the [performance guide](docs/PERFORMANCE.md).

## Features

- End-to-end encrypted traffic between UQDA nodes.
- Self-organizing multi-hop routing without a central routing authority.
- Cryptographically derived IPv6 addresses tied to node identities.
- Peering over TCP, TLS, QUIC, WebSocket, secure WebSocket, SOCKS, and Unix sockets.
- Optional local multicast discovery.
- TUN integration for ordinary IPv6-capable applications.
- Code and integrations for Linux, macOS, Windows, FreeBSD, OpenBSD, OpenWrt, EdgeRouter, and VyOS.
- Local administration through `uqdactl`.

## How it works

Each node generates a cryptographic identity and derives its IPv6 address from the public key. The daemon creates a virtual TUN interface, establishes configured or locally discovered peerings, and forwards encrypted IPv6 packets across the available mesh paths. Peer links may themselves run over either IPv4 or IPv6 networks.

## Build

Requirements: [Go 1.25.13 or newer](https://go.dev/dl/) and Git.

```bash
git clone https://github.com/Uqda/Core.git
cd Core
./build
```

The build produces:

- `uqda` — the network daemon;
- `uqdactl` — the local administration client.

## Verified one-command installation

The release installer detects the operating system and CPU, selects the native
package where available, verifies it against the release `SHA256SUMS`, and then
uses the platform package manager. Stable releases also publish a Sigstore
bundle for the checksum manifest and GitHub artifact attestations. Review the
script before running it:

```bash
curl -fsSLO https://github.com/Uqda/Core/releases/latest/download/install.sh
sudo sh install.sh
```

### Homebrew on macOS

The repository also provides a Homebrew Cask that invokes the same verified
installer. It supports both Apple Silicon and Intel Macs:

```bash
brew tap uqda/core https://github.com/Uqda/Core
brew trust --cask uqda/core/uqda
brew install --cask uqda/core/uqda
```

To ask Homebrew to check and install the newest stable release:

```bash
brew upgrade --cask uqda/core/uqda
```

Homebrew requires explicit trust for third-party taps. Trusting only
`uqda/core/uqda` limits that approval to this Cask instead of every executable
definition in the tap. The Cask delegates package selection and SHA-256
verification to the release installer, so the direct and Homebrew paths have
the same platform detection and checksum protection.

To update through the same verified path while preserving the existing
configuration and node identity:

```bash
curl -fsSLO https://github.com/Uqda/Core/releases/latest/download/updater.sh
sudo sh updater.sh
```

### Windows

Download the matching `.msi` asset from the latest release and run it as an
administrator. The installer creates and starts the `UQDA` Windows service.
New packages add the installation directory to the system `PATH`; open a new
PowerShell window after installation, then use:

```powershell
uqda.exe -version
uqdactl.exe list
uqdactl.exe getSelf
uqdactl.exe getPeers
uqdactl.exe doctor
Get-Service UQDA
Restart-Service UQDA
```

Service commands require an elevated PowerShell window. Do not run `uqda.exe`
without a mode to inspect the installed node: that only prints daemon flags.
Use `uqdactl.exe getSelf` for the running node.

In an older package whose directory is not yet in `PATH`, PowerShell requires
the `.\` prefix for a program in the current directory:

```powershell
cd "C:\Program Files (x86)\UQDA"
.\uqda.exe -version
.\uqdactl.exe getSelf
```

Depending on the installed architecture, the directory can instead be
`C:\Program Files\UQDA`. The configuration and service log are under
`$env:ProgramData\UQDA`.

The stable-release pipeline requires Authenticode signatures on both installed
executables and the MSI. For release-pipeline setup see
[Windows release signing](docs/windows-release-signing.md); for independent
publisher verification see [the release verification guide](docs/release-verification.md).

## Uninstall

Every supported Unix-like package includes the official uninstaller. Normal
removal stops and removes the service and binaries but preserves the
configuration and cryptographic node identity, making a later reinstall use
the same node:

```bash
sudo /usr/local/share/uqda/uninstall.sh
```

Debian/Ubuntu and router packages install it at
`/usr/share/uqda/uninstall.sh`. Homebrew users can use:

```bash
brew uninstall --cask uqda/core/uqda
```

To permanently remove all UQDA configuration, node identity, and backups,
download and review the release script, then explicitly enable purge mode:

```bash
curl -fsSLO https://github.com/Uqda/Core/releases/latest/download/uninstall.sh
sudo sh uninstall.sh --dry-run --purge
sudo sh uninstall.sh --purge
```

The purge command asks for the word `PURGE`; automation must additionally
pass `--yes`. This deletion cannot be undone.

Supported release paths are systemd-based Debian/Ubuntu, Fedora and immutable
Fedora derivatives such as Bazzite, macOS, EdgeOS 2.x, VyOS 1.3, and the listed
portable Linux/FreeBSD/OpenBSD targets. Windows users should download the
matching `.msi` asset from the release. OpenWrt is not yet included in the
one-command installer and remains an explicitly unvalidated release target.

The project does not currently have a paid Apple Developer account. macOS
packages are therefore published with `-unsigned.pkg` in their filename and
will trigger a Gatekeeper warning when opened from Finder. Prefer the verified
command-line installer above, which downloads the declared package and checks
its SHA-256 digest before invoking the system installer. Do not disable
Gatekeeper globally.

## Run

For a temporary node using automatically generated settings:

```bash
sudo ./uqda -autoconf
```

For a persistent configuration:

```bash
./uqda -genconf > uqda.conf
$EDITOR uqda.conf
sudo ./uqda -useconffile ./uqda.conf
```

On Unix-like systems, administration is intentionally root-only. Run all
`uqdactl` commands through `sudo`, for example `sudo uqdactl getSelf`. The
local administration socket is created with mode `0600`; access is not granted
to ordinary users or groups.

Run the read-only health and security diagnostic after installation or when
connectivity is unclear:

```bash
sudo uqdactl doctor
sudo uqdactl -json doctor
```

`doctor` checks the running daemon, identity, administration endpoint, TUN,
direct peers, routing convergence, and multicast bootstrap state without
printing private keys, peer passwords, or other configuration secrets. It
exits with status `0` when healthy, `2` for warnings, and `1` for a security or
identity failure.

Interactive terminal output uses restrained semantic colors: green for healthy
or connected state, yellow for warnings, red for failures, and cyan for labels.
Colors are disabled automatically for JSON, redirected output, and terminals
using `NO_COLOR`. Use `-color=always` or `-color=never` to override automatic
detection for human-readable output. Status words remain present, so color is
never the only indication.

New nodes remain compatible with existing protocol 0.5 peers. When both sides support the hardened handshake, they negotiate it automatically. To require the hardened handshake on a controlled link and reject legacy peers, append `?secure=required` to both the peer URI and listener URI.

Use `-json` with `-genconf` if strict JSON is preferred over commented HJSON. Creating a TUN interface normally requires administrator privileges. On Linux, the binary may instead be granted the required capability:

```bash
sudo setcap CAP_NET_ADMIN=+eip ./uqda
./uqda -useconffile ./uqda.conf
```

## Docker

Pull the public multi-platform package from GitHub Container Registry:

```bash
docker pull ghcr.io/uqda/core:latest
docker run --rm -it \
  --cap-add=NET_ADMIN \
  --device=/dev/net/tun \
  -v uqda-config:/etc/uqda \
  ghcr.io/uqda/core:latest
```

The package supports `linux/amd64`, `linux/arm64`, `linux/arm/v7`, and
`linux/arm/v6`. Stable releases publish `vX.Y.Z`, `X.Y.Z`, `X.Y`, `X`, and
`latest` tags; the current development build is published as `edge`. The
container creates its configuration inside the persistent `uqda-config` volume
on first start.

To build locally instead:

```bash
docker build -t uqda-core -f contrib/docker/Dockerfile .
```

## EdgeOS and VyOS

Integrated router packages are produced by the Packages workflow for
EdgeRouter X (`mipsel`), EdgeRouter Lite (`mips`), and VyOS (`amd64`/`i386`).
After installing the matching `.deb`, create an interface through the native
router CLI:

```text
configure
set interfaces uqda tun0
set interfaces uqda tun0 description UQDA
commit
save
```

Each interface has its own `/config/uqda.tunN.conf`, administration socket,
and `uqda@tunN.service`. Advanced peer settings can be edited in that config,
then applied with `restart uqda tun0`. See [`contrib/vyatta`](contrib/vyatta)
for supported targets and local build commands.

## Repository layout

| Path | Purpose |
| --- | --- |
| `cmd/uqda` | Node daemon entry point |
| `cmd/uqdactl` | Administration CLI |
| `src/core` | Sessions, routing integration, and peer transports |
| `src/config` | Configuration and platform defaults |
| `src/tun` | Platform-specific virtual network interfaces |
| `src/admin` | Local administration API |
| `contrib/` | Containers, service files, packages, and platform integrations |
| `.github/workflows/` | Continuous integration and packaging automation |

## Development

```bash
gofmt -w .
go vet ./...
go test ./...
go build ./...
sh tests/install_test.sh
```

See [CONTRIBUTING.md](CONTRIBUTING.md) before submitting changes. Report suspected vulnerabilities privately as described in [SECURITY.md](SECURITY.md).

## License

UQDA Core is distributed under the GNU Lesser General Public License v3 with the additional exception included in [LICENSE](LICENSE). Third-party components remain subject to their respective licenses.

## Acknowledgements and upstream origin

**UQDA Core is based on and derived from the open-source [Yggdrasil Network](https://github.com/yggdrasil-network/yggdrasil-go) codebase.** UQDA retains substantial concepts and implementation lineage from Yggdrasil while being developed under its own name and repository. Yggdrasil is an independent upstream project; this repository must not imply endorsement by or official affiliation with its maintainers. See [NOTICE.md](NOTICE.md) for attribution details.
