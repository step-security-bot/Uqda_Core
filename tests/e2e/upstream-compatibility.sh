#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "$0")/common.sh"
prepare_e2e
command -v jq >/dev/null || fail 'jq is required'
[[ -x "${UPSTREAM_BIN:-}" ]] || fail 'pinned upstream executable is required'

# Entirely isolated: no public peers, public services or user configurations.
A="uqu-a-$$"
B="uqu-b-$$"
create_namespace "$A"
create_namespace "$B"
link_namespaces "$A" a0 10.243.11.1/30 "$B" b0 10.243.11.2/30
CFG_A=$(write_node_config a - 'tcp://10.243.11.2:12111')
CFG_B="$E2E_TMP/upstream.conf"
umask 077
"$UPSTREAM_BIN" -genconf -json | jq '
  .AdminListen = "none" |
  .MulticastInterfaces = [] |
  .IfName = "ygg-test" |
  .IfMTU = 1280 |
  .Listen = ["tcp://0.0.0.0:12111"] |
  .Peers = []' > "$CFG_B"
ip netns exec "$B" "$UPSTREAM_BIN" -useconffile "$CFG_B" > "$E2E_TMP/upstream.log" 2>&1 &
E2E_PIDS[upstream]=$!
start_node "$A" a "$CFG_A"
wait_peer_count a 1 30
ADDR_B=$("$UPSTREAM_BIN" -useconffile "$CFG_B" -address)
ADDR_A=$(node_address a)
wait_for_ping "$A" "$ADDR_B" 30
wait_for_ping "$B" "$ADDR_A" 30
echo '[E2E] PASS: bidirectional overlay traffic with pinned upstream Yggdrasil 0.5.14'
