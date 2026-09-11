# Administration CLI and JSON API

Mesh transport hardening (development): WSS uses system CA and hostname
verification. Direct TLS and QUIC validate self-issued Ed25519 certificates and
bind the certificate key to the overlay identity before admitting the peer.
Self-issued mesh TLS still bypasses public CA/DNS verification; its CodeQL alert
must not be treated as closed solely because custom validation was added.

Development security update: local TCP administration uses a separate certificate
derived from the existing node identity, a node-only trust store, and standard
mutual certificate verification. The mesh certificate is not modified.
Listener startup errors omit addresses and raw errors to avoid credential leaks.

## Development hardening: authenticated local administration

In the development branch, TCP administration (including existing `tcp://`
endpoints) negotiates mutual TLS 1.3. Both endpoints pin this node's public key;
the client loads the local protected configuration to prove possession of its
private key. Update daemon and client together. Use `uqdactl -useconffile PATH`
for a non-default local configuration. Never copy the node private key to a
remote administrator. Unix sockets retain OS permissions. Mesh peering,
protocol 0.5, public peer keys and overlay addresses are unchanged.

Administration sessions are limited to 64 concurrent connections, 1 MiB total
JSON input per connection and 30 seconds per request. A CLI elevation check is
not the server's authentication boundary. The historical unauthenticated TCP
description below applies to v0.1.10 and earlier, not this development branch.

[Documentation index](README.md) · [الإدارة بالعربية](NETWORK_GUIDE_AR.md#إدارة-العقدة)

`uqda` is the daemon; `uqdactl` queries the running daemon. Running `uqda`
without a mode prints help rather than starting the installed service.
`uqda -version` reports that executable, while `uqdactl getSelf` reports the
running node. Their versions can differ until a service restart after upgrade.

## Local access boundary

The [admin implementation](../src/admin/admin.go) has no application-level
authentication. Filesystem Unix sockets are mode `0600`: use the authorized
owner/root account, normally `sudo uqdactl`. On Windows the default is a
loopback TCP endpoint; loopback restricts remote access, but is NOT an
administrator-only authentication mechanism. Other local processes may connect.
Keep untrusted users/processes outside the node's host security boundary.

Source after PR #48 uses Windows `tcp://localhost:19001`; older v0.1.9 packages
use 9001. An explicit installed configuration may override the default.
Use the endpoint in the running service's actual configuration; never bind
administration to all interfaces to fix a connection error.

## CLI examples

Unix-like systems:

```sh
sudo uqdactl list
sudo uqdactl getSelf
sudo uqdactl getPeers
sudo uqdactl -json doctor
```

Windows PowerShell, using the normally installed binaries:

```powershell
uqda.exe -version
uqdactl.exe list
uqdactl.exe getSelf
uqdactl.exe getPeers
uqdactl.exe -json doctor
```

Open a new terminal after an installer updates PATH. Within the installation
directory use `.\uqdactl.exe` if needed. If Windows application-control policy
blocks the executable, stop and have the policy owner review the file and
CodeIntegrity/AppLocker evidence. A different client is not a policy fix.

Options go BEFORE the command; named arguments go AFTER it:

```sh
sudo uqdactl -json getPeers sort=uptime
sudo uqdactl -endpoint=unix:///var/run/uqda.sock getSelf
```

Use `list` on the running version as the authoritative command inventory.

| Command | Purpose | Relevant arguments |
| --- | --- | --- |
| `list` | Registered commands and argument names | none |
| `getSelf` | Build, identity-derived address/subnet, routing entries | none |
| `getPeers` | Direct links, state, counters and connection errors | optional `sort`; consult current implementation |
| `getTree` | Known routing-tree entries | none |
| `getPaths` | Established routing paths | none |
| `getSessions` | End-to-end traffic sessions, not just direct peers | none |
| `getTun` | Whether TUN is enabled; name and effective MTU | none |
| `getMulticastInterfaces` | Active discovery interface state | none |
| `getNodeInfo` | Request a remote node's voluntarily published metadata | `key=PUBLIC_KEY_HEX` |
| `doctor` | Read-only health/security summary | none |
| `addPeer` | Add an outbound peer at runtime | `uri=...`, optional `interface=...` |
| `removePeer` | Remove a runtime peer | `uri=...`, optional `interface=...` |

Additional debug handlers can appear. `lookups` depends on `LogLookups`; its
records expose network metadata and it is not a routine health command.
Do not copy historical `getDHT`/`getTunTap` examples: those are not the current
commands above. `doctor` CLI exit codes are 0 healthy, 2 warning, 1 failure;
a warning may mean the daemon is running but no useful peer is connected.

Runtime changes are not persistent configuration edits. To keep a peer after
restart, edit `Peers` or `InterfacePeers` securely in the service configuration.
To remove it permanently, remove it there as well. Avoid secrets in CLI history.

## Wire format for authorized integrations

This is JSON over a local stream socket, NOT HTTP or a REST endpoint.
Send a complete JSON request, conventionally followed by a newline:

```json
{"request":"getSelf","arguments":{},"keepalive":false}
```

Arguments belong inside `arguments`, not alongside `request`. The
[request/response structures and framing](../src/admin/admin.go) are the source
of truth. A response has `status` (`success` or `error`), the echoed `request`,
`response` data, and optional `error` text. The echoed request is an object,
not a string. The JSON is pretty-printed across multiple lines.

Read one complete JSON value with a streaming decoder. With `keepalive:false`
(the default), the server closes after the response, so a bounded read to EOF
also works. With `keepalive:true`, parse successive JSON values; do not wait
for EOF between requests. Reading only one line can produce just `{`, which
is not a complete JSON document. Integrations should impose connection/read
timeouts and response-size limits and close sockets on failure.

`uqdactl -json` prints the response BODY; it does not print the entire socket
response envelope. Also distinguish outer API success from a `doctor` body
whose own `status` is `warning` or `failure`.

## Troubleshooting in order

1. Check service state and its executable/configuration paths.
2. Inspect startup logs before repeatedly restarting a failing service.
3. Confirm the local admin endpoint is enabled and listening.
4. If administration works, inspect peers, TUN, routes and the target application.
5. Redact private keys/passwords and sensitive network metadata before sharing logs.

`Connection refused` means no listener accepted that connection at that address
and time; it is not by itself proof of bad peers or an invalid key. A generic
MSI service-start error is likewise not proof that the user's password was wrong.
See [Windows diagnostics](windows-installation.md) for that case.
