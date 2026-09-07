# UQDA Public and Café Gateway Safety

Running a public visitor gateway is a network-administration task with legal,
privacy, abuse, and isolation risks. UQDA Core does not configure a public Wi-Fi
network, captive portal, firewall, DHCP service, or client isolation.

Use a maintained router platform and have its configuration reviewed by a
qualified network administrator before opening the network to visitors.

## Minimum design requirements

- Put visitors in a dedicated SSID, VLAN, and firewall zone.
- Block visitor-to-visitor traffic.
- Block access from visitors to the gateway's management services and upstream
  private networks.
- Permit only the DHCP, DNS, and essential IPv6 control traffic needed by the
  design.
- Keep management on a separate wired interface or management VLAN.
- Apply bandwidth and client limits appropriate to the venue.
- Keep a tested configuration backup and a spare recovery path.
- Publish acceptable-use and privacy notices required by local law.

## Acceptance tests

Use two visitor devices and one management device. Verify that both visitors
receive intended Internet and UQDA connectivity, cannot discover or reach each
other, cannot reach router administration or private upstream networks, and
cannot bypass isolation over IPv6. Verify that management access still works
and that all configuration survives a reboot.

Repeat the full test after every router firmware, kernel, Wi-Fi driver, firewall,
or UQDA update.

## Emergency procedure

Document and test how to disable the visitor SSID from a wired or local console
and how to restore the last known-good router configuration. Do not depend on
remote Wi-Fi access as the only recovery method.
