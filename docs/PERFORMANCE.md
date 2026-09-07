# UQDA Latency and Performance

UQDA uses encrypted peer links and a latency-aware routing metric, but it cannot
remove physical distance, ISP congestion, Wi-Fi interference, or queueing in
the underlying network. Measure the whole path before changing UQDA settings.

## Measure with operating-system tools

Use the native IPv6 ping command against a known UQDA address. Collect at least
20 samples; a single successful reply does not describe stability.

Linux, macOS, FreeBSD, and OpenBSD:

```sh
ping -6 -c 20 21c:f3b0:e941:88bc:2938:a694:9012:37f2
```

Some BSD/macOS versions use `ping6` instead:

```sh
ping6 -c 20 21c:f3b0:e941:88bc:2938:a694:9012:37f2
```

Windows PowerShell:

```powershell
Test-Connection 21c:f3b0:e941:88bc:2938:a694:9012:37f2 -Count 20
```

Inspect direct UQDA peers and their measured round-trip time:

```sh
sudo uqdactl getPeers sort=latency
```

On Windows, run the same `uqdactl.exe getPeers sort=latency` command from an
elevated terminal.

## How routing uses latency

The embedded Ironwood router measures direct-link RTT with signed request and
response traffic, smooths it with an exponentially weighted average, and uses
the resulting link cost when selecting a loop-free next hop. New links receive
a temporary stability penalty so a tiny sample change does not continuously
move routes.

Adding many distant peers can make performance worse. Prefer a small set of
reliable peers near the device or directly peer the two sites that need low
latency. The `priority` URI option selects between duplicate links to the same
public key; it is not an end-to-end latency guarantee.

## Diagnose poor performance

1. Run `uqdactl doctor` and resolve failed checks.
2. Compare direct-peer RTT from `getPeers sort=latency` with end-to-end ping.
3. Test a wired connection to separate Wi-Fi interference from overlay delay.
4. Test while the underlying connection is idle and while it is loaded.
5. Check packet loss, MTU settings, CPU load, and whether the selected route has
   unnecessary hops.
6. Compare with an ordinary Internet target to identify an underlay problem.

For geographically remote nodes, record a realistic baseline. A longer
geographic path cannot reliably meet the same target as two devices on one LAN.

## Operational recommendations

- Prefer wired links for gateways and high-traffic nodes.
- Keep the operating system, network drivers, and UQDA updated.
- Avoid oversized peer lists and unstable peers.
- Use traffic shaping on a persistently loaded uplink to control bufferbloat.
- Re-run measurements after changing peers, Wi-Fi, MTU, or the upstream router.
