#!/usr/bin/env bash
set -euo pipefail

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "This verifier must run on macOS because it exercises Accessibility/AX APIs." >&2
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
APP_DIR="${REPO_ROOT}/app"
POINT="${RECORDINGFREEDOM_REGION_PROBE:-cursor}"

if [[ "$(go env CGO_ENABLED)" != "1" ]]; then
  echo "CGO_ENABLED must be 1 so the macOS Accessibility/AX provider is compiled." >&2
  echo "Run: CGO_ENABLED=1 bash scripts/verify-region-recognition-macos.sh" >&2
  exit 1
fi

cat <<'MSG'
RecordingFreedom macOS region-recognition verifier

Before this runs:
1. Grant Accessibility permission to the Terminal/iTerm app running this script.
2. Move the mouse over a visible UI control inside Finder, a browser, or another normal app.
3. Keep the pointer there until both probes finish.

Starting in 5 seconds...
MSG

sleep 5
cd "${APP_DIR}"

export RECORDINGFREEDOM_REGION_PROBE="${POINT}"
ELEMENT_LOG="$(mktemp)"
ASSIST_LOG="$(mktemp)"
trap 'rm -f "${ELEMENT_LOG}" "${ASSIST_LOG}"' EXIT

go test -run 'TestRegion(UIElement|Accessibility)ProbeFromEnv' -v . | tee "${ELEMENT_LOG}"
if ! grep -q 'candidate\[[0-9][0-9]\].*source=accessibility:' "${ELEMENT_LOG}"; then
  echo "Accessibility probe did not print a native AX candidate chain." >&2
  echo "Keep the cursor over a visible control and verify Accessibility permission for this terminal." >&2
  exit 1
fi

go test -run TestRegionAssistDesktopProbeFromEnv -v . | tee "${ASSIST_LOG}"
if ! grep -q 'region-assist source=element' "${ASSIST_LOG}"; then
  echo "Assist probe did not prove source=element." >&2
  echo "The dynamic selector would fall back to image/window candidates instead of native element recognition." >&2
  exit 1
fi

cat <<'MSG'

Verifier complete. The script proved:
- a native Accessibility candidate chain was printed, and
- region-assist returned source=element with a best element bound containing the pointer.
MSG
