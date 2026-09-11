#!/bin/sh

set -eu

TAG=${UQDA_VERSION:-}
if [ -z "$TAG" ]; then
  TAG=$(git describe --abbrev=0 --tags --match="v[0-9]*\.[0-9]*\.[0-9]*" 2>/dev/null || true)
fi

if [ -z "$TAG" ]; then
  # MSI ProductVersion must contain only numeric components.
  COUNT=$(git rev-list --count HEAD 2>/dev/null || printf '0')
  printf '0.0.%d' "$((COUNT % 65535))"
  exit 0
fi

# Reserve 100 build values per patch: beta.1..49, rc.1..49, then stable.
# Introduced after v0.1.10; release new patches, never reissue an old stable tag.
# The application and filename retain the complete SemVer.
SEMVER=${TAG#v}
printf '%s\n' "$SEMVER" | grep -Eq '^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-(beta|rc)\.([1-9]|[1-4][0-9]))?$' || {
  echo "Invalid release version: $TAG" >&2
  exit 1
}

NUMERIC=${SEMVER%%-*}
IFS=. read -r MAJOR MINOR PATCH <<EOF
$NUMERIC
EOF
[ "$MAJOR" -le 255 ] && [ "$MINOR" -le 255 ] && [ "$PATCH" -le 654 ] || {
  echo "MSI version out of range (major/minor <=255, patch <=654): $TAG" >&2
  exit 1
}
STAGE=99
case "$SEMVER" in
  *-beta.*) STAGE=${SEMVER##*.} ;;
  *-rc.*) STAGE=$((49 + ${SEMVER##*.})) ;;
esac
printf '%d.%d.%d' "$MAJOR" "$MINOR" "$((PATCH * 100 + STAGE))"
