# Configuration reference

[Documentation index](README.md) · [الشرح العملي بالعربية](NETWORK_GUIDE_AR.md#الإعداد-الدائم)

This reference follows [NodeConfig](../src/config/config.go) in the current
source. Generate defaults with your installed binary; fields omitted from a
file receive defaults when it is read. HJSON and JSON are accepted. Do not
replace a working identity with a generated example configuration.

## Generate and validate safely

For a NEW node on Unix-like systems, in a private directory where `uqda.conf`
does not already exist:

```sh
umask 077
uqda -genconf > uqda.conf
uqda -useconffile uqda.conf -address
```

For a NEW node in PowerShell, avoid accidentally overwriting an existing file:

```powershell
if (Test-Path -LiteralPath .\uqda.conf) { throw 'uqda.conf already exists' }
$configLines = & uqda.exe -genconf
if ($LASTEXITCODE -ne 0) { throw 'Configuration generation failed' }
[IO.File]::WriteAllLines((Join-Path $PWD 'uqda.conf'), [string[]]$configLines, [Text.UTF8Encoding]::new($false))
uqda.exe -useconffile .\uqda.conf -address
```

Store the file somewhere access-controlled, not a shared working directory.
For an installed service, back up and edit the file used by that service instead.
The `-address` check parses the file and loads the key; it does NOT verify every
peer URI, network reachability, driver, firewall rule or service permission.
`-normaliseconf` prints the normalized configuration, which can contain secrets;
do not paste that output into issues. Never redirect normalization to its input
file, because shell redirection may truncate the input before it is read.

## Configuration fields

| Field | Type | Meaning and caution |
| --- | --- | --- |
| `PrivateKey` | hex string | Ed25519 private identity. Keep secret and backed up; do not reuse one identity on simultaneously running nodes. If no persistent key is supplied, startup can generate a different identity. |
| `PrivateKeyPath` | string | PEM private-key file; takes precedence over `PrivateKey`. Protect and back up this file too. Prefer an absolute path for a service. |
| `Peers` | string array | Persistent outbound peer URIs; all overlay traffic travels via peer links, not a separate bootstrap service. See [Peers](#peers). |
| `InterfacePeers` | map of string arrays | Outbound peers grouped by local source-interface name for multi-homed hosts. Otherwise use `Peers`. |
| `Listen` | string array | Incoming peer listeners. Use local bind addresses, not remote hosts. Not required for outbound-only peering; multicast has its own listeners. |
| `AdminListen` | string | Local administration endpoint, or `none` to disable. This API has no application-level authentication; never expose it publicly. |
| `MulticastInterfaces` | object array | Ordered interface-name matching rules for LAN discovery; see [MulticastInterfaces](#multicastinterfaces). |
| `AllowedPublicKeys` | hex-string array | Allowlist for non-local incoming direct peer links. Empty means no key allowlist. Does not restrict outgoing or multicast-discovered links and is NOT an IPv6 service firewall. |
| `GroupPassword` | string | End-to-end session group secret. A mismatch prevents traffic sessions but does not isolate peering or stop forwarding transit traffic. Different derivation formats may be incompatible even with identical text. |
| `IfName` | string | TUN interface name, `auto` for platform selection, or `none` to disable host TUN. No TUN means ordinary host applications cannot use this node's TUN interface. |
| `IfMTU` | integer | Requested local TUN MTU, minimum 1280; supported maximum is platform-dependent. Inspect effective state with `getTun`. |
| `LogLookups` | boolean | Optional lookup-recording diagnostic via the admin module. Its records include keys, paths and times; not needed for normal operation. Treat output as sensitive network metadata. |
| `NodeInfoPrivacy` | boolean | Suppresses automatically added build/platform/architecture information; does not hide manually supplied `NodeInfo`. Default false. |
| `NodeInfo` | object or null | Optional network-queryable metadata. Never include credentials, private keys or private personal information. |

`Certificate` is internal (`json:"-"`), not a user configuration field. The
node's self-issued transport certificate is unrelated to Windows Authenticode
publisher signing. `PrivateKeyPath` expects an Ed25519 key, not an arbitrary
TLS key/certificate bundle.

## Peers

These are illustrative URI shapes, NOT live public peers. Replace
`peer.example.net`, ports and any key placeholders with operator-approved values.

| Scheme | Outbound | Native listener | Shape |
| --- | --- | --- | --- |
| `tcp` | yes | yes | `tcp://peer.example.net:12345` |
| `tls` | yes | yes | `tls://peer.example.net:12345` |
| `quic` | yes | yes | `quic://peer.example.net:12345` |
| `ws` | yes | yes | `ws://peer.example.net:12345/path` |
| `wss` | yes | no | `wss://peer.example.net:443/path`; use a WS listener behind an appropriately configured TLS reverse proxy |
| `socks` | yes | no | `socks://127.0.0.1:1080/peer.example.net:12345` |
| `sockstls` | yes | no | `sockstls://127.0.0.1:1080/peer.example.net:12345` |
| `unix` | yes | yes | `unix:///path/to/peer.sock`; platform support applies |

SOCKS here is an outbound carrier through an existing proxy, not a SOCKS server
for arbitrary applications. `Listen` and `AdminListen` have different purposes;
do not use the same address/port for both.

Generic URI parsing is in [link.go](../src/core/link.go):

| Query option | Scope | Meaning |
| --- | --- | --- |
| `secure=required` | peer and listener | Require UQDA transcript-bound handshake confirmation; incompatible legacy peers are rejected. |
| `secure=opportunistic` | peer and listener | Negotiate confirmation if supported; this is the default behavior when omitted. |
| `password=...` | peer and listener | Shared direct-link password, maximum 64 bytes; must match the other side. |
| `priority=0` | peer and listener | 0–255 link priority, lower preferred among links to the same identity; not a general routing-cost override. |
| `key=...` | outbound peer | Pin an expected authenticated Ed25519 public key; repeat `key` for multiple acceptable keys. Obtain it over a trusted channel. |
| `maxbackoff=30s` | outbound peer | Reconnect backoff cap, minimum 5 seconds. |
| `sni=...` | outbound peer | Carrier-dependent TLS SNI override; not a replacement for key verification. |

Percent-encode reserved characters in URI values. Avoid placing secrets in shell
history. The current [WS implementation](../src/core/link_ws.go) interprets
listener `origin` values as allowed origin patterns; do not copy upstream's
outbound `?origin=` header behavior into UQDA instructions. An `origin=*`
listener bypasses the origin check and is not recommended as a default.
The [WSS implementation](../src/core/link_wss.go) has no native listener.

## MulticastInterfaces

| Field | Meaning |
| --- | --- |
| `Regex` | Regular expression matching the OS interface name; first matching rule wins. |
| `Beacon` | Advertise this node on matching interfaces. |
| `Listen` | Discover and attempt connections to matching neighboring nodes. |
| `Port` | Discovery-created TLS listener port; 0 selects a port dynamically. |
| `Priority` | Preference among links to the same node; lower values are preferred. |
| `Password` | Shared local-discovery secret; separate from link and group passwords. |

Discovery is local, not an Internet peer directory. Interface names differ
between OSes and computers. Verify them before writing a rule; do not copy
`eth0` or `Ethernet` blindly. An empty array disables configured multicast
discovery, so provision a reachable peer if remote connectivity is needed.

## Platform defaults and applying changes

See the [platform defaults table](PROJECT_GUIDE.md#platform-defaults) and
[OS-specific service commands](NETWORK_GUIDE_AR.md#التثبيت-والتشغيل-حسب-النظام).
Windows source after PR #48 defaults to `tcp://localhost:19001`; published
v0.1.9 packages still use the older default. Explicit existing `AdminListen`
values are preserved. Read [Windows migration](windows-installation.md).

Editing a file does not automatically reconfigure a running daemon. After
backup and validation, restart its actual service and query `getSelf`,
`getPeers` and `doctor`. Runtime `addPeer`/`removePeer` do not write the file.
Back up both configuration and external key files before upgrades or removal.
