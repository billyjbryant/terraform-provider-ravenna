#!/usr/bin/env bash
# Fails when Ravenna changes an endpoint this provider depends on.
# Run manually or in CI; update api/openapi.json deliberately when it fires.
set -euo pipefail

SPEC_URL="https://core.api.ravenna.ai/openapi.json"
VENDORED="api/openapi.json"
LIVE="$(mktemp)"
trap 'rm -f "$LIVE"' EXIT

# Paths the provider calls. Add to this list as resources are added.
PATHS='["/queues","/queues/{id}","/tags","/tags/{id}","/statuses"]'

curl -sSL "$SPEC_URL" -o "$LIVE"

extract() {
  jq -S --argjson want "$PATHS" \
    '.paths | with_entries(select(.key as $k | $want | index($k)))' "$1"
}

if diff -u <(extract "$VENDORED") <(extract "$LIVE"); then
  echo "spec-diff: no changes to the endpoints this provider uses"
else
  echo
  echo "spec-diff: the Ravenna API changed under an endpoint this provider uses." >&2
  echo "Review the diff above, update the client, then refresh the vendored spec:" >&2
  echo "  curl -sSL $SPEC_URL -o $VENDORED" >&2
  exit 1
fi
