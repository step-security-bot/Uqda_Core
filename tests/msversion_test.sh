#!/bin/sh
set -eu
ROOT=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
check() {
  actual=$(UQDA_VERSION="$1" sh "$ROOT/contrib/msi/msversion.sh")
  [ "$actual" = "$2" ] || { echo "incorrect MSI mapping for $1: $actual" >&2; exit 1; }
}
check v0.1.11-beta.1 0.1.1101
check v0.1.11-beta.2 0.1.1102
check v0.1.11-beta.49 0.1.1149
check v0.1.11-rc.1 0.1.1150
check v0.1.11-rc.49 0.1.1198
check v0.1.11 0.1.1199
check v0.1.12-beta.1 0.1.1201
check v255.255.654 255.255.65499
for invalid in v256.1.1 v1.256.1 v0.1.655 v1.2.3garbage v01.2.3 v1.2.3-beta.0 v1.2.3-beta.50 v1.2.3-rc.50 v1.2.3+metadata; do
  if UQDA_VERSION="$invalid" sh "$ROOT/contrib/msi/msversion.sh" >/dev/null 2>&1; then
    echo "invalid MSI version accepted: $invalid" >&2
    exit 1
  fi
done
echo 'MSI version ordering and bounds tests passed'
