# UQDA Home Gateway

A Linux or OpenWrt router can run UQDA and route its assigned IPv6 subnet to
devices on a trusted home LAN. UQDA Core itself does not create Wi-Fi access
points, edit firewall rules, or configure Router Advertisements. Those settings
must be managed with the operating system's supported network tools.

## Before configuring a gateway

- Install and start UQDA on the router.
- Confirm `uqdactl getSelf` reports an IPv6 address and a routed `/64`.
- Confirm `uqdactl getPeers` shows at least one working peer.
- Keep Ethernet or local-console access available while changing networking.
- Back up NetworkManager, firewall, DHCP, and OpenWrt UCI configuration.
- Identify the WAN and LAN interfaces explicitly; never guess them.

## Required network behavior

The router configuration must:

1. enable IPv6 forwarding;
2. assign one address from the UQDA `/64` to the trusted LAN;
3. advertise that prefix to clients using IPv6 Router Advertisements;
4. route the UQDA address range through the UQDA TUN interface;
5. preserve ordinary Internet connectivity independently; and
6. keep the UQDA administration endpoint local to the router.

Use NetworkManager, radvd, nftables, or OpenWrt UCI according to the platform's
documentation. Do not copy a configuration between devices without reviewing
interface names, firewall zones, regulatory country, and recovery access.

## Acceptance tests

Before relying on the gateway:

- reboot it and confirm UQDA and the network configuration return;
- confirm a client receives an address from the routed UQDA subnet;
- reach a known UQDA peer from that client;
- confirm ordinary Internet access still works;
- confirm the UQDA administration endpoint is not reachable from Wi-Fi; and
- restore the saved configuration to prove the recovery procedure works.

Wi-Fi chips, drivers, regulatory domains, and OpenWrt packages vary. Treat a
router model as supported only after these tests pass on that exact hardware.

## Security

Use a strong Wi-Fi password, apply an IPv6 firewall to the router and clients,
and keep the operating system and UQDA updated. Overlay encryption does not make
applications trustworthy and does not replace LAN isolation or host firewalls.
