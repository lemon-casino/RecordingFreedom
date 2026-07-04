#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -ne 1 ]; then
  echo "Usage: $0 <RecordingFreedom-linux-*.tar.gz>" >&2
  exit 2
fi

archive_path="$1"
if [ ! -s "${archive_path}" ]; then
  echo "Linux portable archive is missing or empty: ${archive_path}" >&2
  exit 1
fi

work_dir="$(mktemp -d "${TMPDIR:-/tmp}/recordingfreedom-linux-verify.XXXXXX")"
cleanup() {
  rm -rf "${work_dir}"
}
trap cleanup EXIT

tar -xzf "${archive_path}" -C "${work_dir}"
portable_root="$(find "${work_dir}" -mindepth 1 -maxdepth 1 -type d -name "RecordingFreedom-linux-*" -print -quit)"
if [ -z "${portable_root}" ]; then
  echo "RecordingFreedom-linux-* root directory was not found in ${archive_path}" >&2
  exit 1
fi

test -x "${portable_root}/recordingfreedom"
test -x "${portable_root}/tools/ffmpeg"
test -x "${portable_root}/tools/ffprobe"
test -s "${portable_root}/tools/THIRD_PARTY_FFMPEG.txt"
test -s "${portable_root}/tools/THIRD_PARTY_NOTICES.txt"
test -s "${portable_root}/recordingfreedom.desktop"

"${portable_root}/tools/ffmpeg" -version >/dev/null
"${portable_root}/tools/ffprobe" -version >/dev/null

if command -v file >/dev/null 2>&1; then
  file "${portable_root}/recordingfreedom" | grep -Eq "ELF 64-bit.*x86-64|ELF 64-bit.*x86_64"
fi

echo "Linux portable archive verified: ${archive_path}"
