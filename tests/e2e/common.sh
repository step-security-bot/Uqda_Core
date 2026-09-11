#!/usr/bin/env bash

set -euo pipefail
export LC_ALL=C

ROOT=$(CDPATH= cd -- "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
UQDA_BIN=${UQDA_BIN:-"$ROOT/uqda"}
UQDACTL_BIN=${UQDACTL_BIN:-"$ROOT/uqdactl"}
E2E_TMP=${UQDA_E2E_TMP:-$(mktemp -d "${TMPDIR:-/tmp}/uqda-e2e.XXXXXX")}

declare -a E2E_NAMESPACES=()
declare -A E2E_PIDS=()
declare -A E2E_LOGS=()

fail() {
  echo "[E2E] ERROR: $*" >&2
  return 1
}

prepare_e2e() {
  if [[ ${EUID:-$(id -u)} -ne 0 ]]; then
    fail "run E2E tests as root (for example: sudo bash tests/e2e/2node.sh)"
  fi
  for command in ip tc ping sed grep awk; do
    command -v "$command" >/dev/null 2>&1 || fail "required command not found: $command"
  done
  [[ -x "$UQDA_BIN" ]] || fail "UQDA binary not found; run ./build first"
  [[ -x "$UQDACTL_BIN" ]] || fail "uqdactl binary not found; run ./build first"
  if [[ ! -c /dev/net/tun ]]; then
    command -v modprobe >/dev/null 2>&1 && modprobe tun >/dev/null 2>&1 || true
  fi
  [[ -c /dev/net/tun ]] || fail "/dev/net/tun is unavailable"
}

create_namespace() {
  local ns=$1
  ip netns add "$ns"
  ip -n "$ns" link set lo up
  E2E_NAMESPACES+=("$ns")
}

link_namespaces() {
  local left_ns=$1 left_if=$2 left_addr=$3 right_ns=$4 right_if=$5 right_addr=$6
  ip link add "$left_if" type veth peer name "$right_if"
  ip link set "$left_if" netns "$left_ns"
  ip link set "$right_if" netns "$right_ns"
  ip -n "$left_ns" addr add "$left_addr" dev "$left_if"
  ip -n "$right_ns" addr add "$right_addr" dev "$right_if"
  ip -n "$left_ns" link set "$left_if" up
  ip -n "$right_ns" link set "$right_if" up
}

socket_path() {
  printf '%s/%s.sock\n' "$E2E_TMP" "$1"
}

fixture_private_key() {
  # Fixed test-only Ed25519 private keys make topology/routing behavior
  # reproducible between CI runs. They are never used outside ephemeral tests.
  case "$1" in
    a) printf '%s\n' '958e239c091df868b98b9d38fe35d657e0b372c920bab72b9ec4394f508f1723b16962d00c9b7397fbbddb6f637ed3a634c0f2c99e5318c23d7128b49a6e1963' ;;
    b) printf '%s\n' '6c32fc994e4a679dcc753a0145138724bc54d3879d12a1130c44694df6f273306d81990cd622cb587f40f0674b561a2ea67358736aa7dab63452cfe03ff33c36' ;;
    c) printf '%s\n' '2bbd449a26c13ff8daee9d76ca448b6940b5a52a84d929d2aa27ce849e66e63fff5f9916cfc13754bf335e2085f82582e1e0478e09c7a1e0b139180b7b592c42' ;;
    d) printf '%s\n' '4e64a0237edfd539d6dcc20b4405b3c238eb5ecae0bcd4de01cee8cd6c697f4168929f5e6e1a24e2fb4cfb4ac97357116577e7d8d7c009297e1dac53f35be94b' ;;
    s) printf '%s\n' '176f944f0929486715f945ad6ff5c68eaf07b0d489ce17a9c251cc66bf126a18b00a32507adfe003b18e6acb1756f94e3021e212cdd791ac798357743a383a3e' ;;
    *) fail "no deterministic E2E private key for node $1" ;;
  esac
}

write_node_config() {
  local node=$1 listen=$2
  shift 2
  local cfg="$E2E_TMP/$node.conf"
  local socket private_key
  socket=$(socket_path "$node")
  private_key=$(fixture_private_key "$node")

  {
    echo '{'
    printf '  PrivateKey: "%s"\n' "$private_key"
    printf '  AdminListen: "unix://%s"\n' "$socket"
    echo '  IfName: "uqda0"'
    echo '  IfMTU: 1280'
    echo '  NodeInfoPrivacy: true'
    echo '  MulticastInterfaces: []'
    echo '  Listen: ['
    if [[ "$listen" != "-" ]]; then
      printf '    "%s"\n' "$listen"
    fi
    echo '  ]'
    echo '  Peers: ['
    local peer
    for peer in "$@"; do
      printf '    "%s"\n' "$peer"
    done
    echo '  ]'
    echo '}'
  } > "$cfg"
  printf '%s\n' "$cfg"
}

start_node() {
  local ns=$1 node=$2 cfg=$3
  local socket
  socket=$(socket_path "$node")
  local log="$E2E_TMP/$node.log"
  rm -f "$socket"
  E2E_LOGS["$node"]=$log

  ip netns exec "$ns" "$UQDA_BIN" -useconffile "$cfg" -loglevel debug >"$log" 2>&1 &
  local pid=$!
  E2E_PIDS["$node"]=$pid

  local deadline=$((SECONDS + 15))
  while (( SECONDS < deadline )); do
    if ! kill -0 "$pid" 2>/dev/null; then
      cat "$log" >&2 || true
      fail "node $node exited during startup"
    fi
    if [[ -S "$socket" ]] && "$UQDACTL_BIN" -endpoint="unix://$socket" getSelf >/dev/null 2>&1; then
      echo "[E2E] node $node is ready"
      return 0
    fi
    sleep 0.1
  done
  cat "$log" >&2 || true
  fail "node $node did not become ready"
}

stop_node() {
  local node=$1
  local pid=${E2E_PIDS[$node]:-}
  [[ -n "$pid" ]] || return 0
  if kill -0 "$pid" 2>/dev/null; then
    kill -TERM "$pid" 2>/dev/null || true
    local deadline=$((SECONDS + 5))
    while kill -0 "$pid" 2>/dev/null && (( SECONDS < deadline )); do
      sleep 0.1
    done
    kill -KILL "$pid" 2>/dev/null || true
  fi
  wait "$pid" 2>/dev/null || true
  unset 'E2E_PIDS[$node]'
}

ctl_json() {
  local node=$1 command=$2
  local socket
  socket=$(socket_path "$node")
  "$UQDACTL_BIN" -json -endpoint="unix://$socket" "$command"
}

node_address() {
  ctl_json "$1" getSelf | sed -n 's/^[[:space:]]*"address":[[:space:]]*"\([^"]*\)".*/\1/p' | head -1
}

routing_entries() {
  ctl_json "$1" getSelf | sed -n 's/^[[:space:]]*"routing_entries":[[:space:]]*\([0-9][0-9]*\).*/\1/p' | head -1
}

peer_up_count() {
  local output
  output=$(ctl_json "$1" getPeers)
  grep -c '"up":[[:space:]]*true' <<<"$output" || true
}

wait_peer_count() {
  local node=$1 expected=$2 timeout=${3:-20}
  local deadline=$((SECONDS + timeout))
  while (( SECONDS < deadline )); do
    if (( $(peer_up_count "$node") >= expected )); then
      return 0
    fi
    sleep 0.2
  done
  ctl_json "$node" getPeers >&2 || true
  fail "node $node did not reach $expected connected peer(s)"
}

wait_peer_error() {
  local node=$1 text=$2 timeout=${3:-15}
  local deadline=$((SECONDS + timeout))
  while (( SECONDS < deadline )); do
    if ctl_json "$node" getPeers 2>/dev/null | grep -F "$text" >/dev/null; then
      return 0
    fi
    sleep 0.2
  done
  ctl_json "$node" getPeers >&2 || true
  fail "node $node did not report expected peer error: $text"
}

wait_for_ping() {
  local ns=$1 address=$2 timeout=${3:-25} interface=${4:-uqda0}
  local deadline=$((SECONDS + timeout))
  while (( SECONDS < deadline )); do
    if ip netns exec "$ns" ping -6 -n -I "$interface" -c 1 -W 1 "$address" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.25
  done
  fail "IPv6 ping from $ns to $address did not succeed"
}

assert_routing_entries_at_least() {
  local node=$1 expected=$2
  local actual
  actual=$(routing_entries "$node")
  [[ -n "$actual" ]] || fail "could not read routing table size from $node"
  (( actual >= expected )) || fail "node $node has $actual routing entries, expected at least $expected"
}

dump_e2e_state() {
  local node
  for node in "${!E2E_LOGS[@]}"; do
    echo "===== $node: getSelf =====" >&2
    ctl_json "$node" getSelf >&2 2>/dev/null || true
    echo "===== $node: getPeers =====" >&2
    ctl_json "$node" getPeers >&2 2>/dev/null || true
    echo "===== $node: getPaths =====" >&2
    ctl_json "$node" getPaths >&2 2>/dev/null || true
    echo "===== $node: getTree =====" >&2
    ctl_json "$node" getTree >&2 2>/dev/null || true
    echo "===== $node: log =====" >&2
    cat "${E2E_LOGS[$node]}" >&2 2>/dev/null || true
  done
}

cleanup_e2e() {
  local status=$?
  trap - EXIT
  set +e
  if (( status != 0 )); then
    dump_e2e_state
  fi
  local node
  for node in "${!E2E_PIDS[@]}"; do
    stop_node "$node"
  done
  local ns
  for ns in "${E2E_NAMESPACES[@]}"; do
    ip netns del "$ns" >/dev/null 2>&1 || true
  done
  rm -rf "$E2E_TMP"
  exit "$status"
}

trap cleanup_e2e EXIT
