#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -lt 1 ] || [ "$#" -gt 2 ]; then
  echo "Usage: $0 <RecordingFreedom-linux-*.tar.gz> [x64|arm64]" >&2
  exit 2
fi

archive_path="$1"
expected_arch="${2:-x64}"
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

host_arch="$(uname -m)"
can_execute_bundled_tools="false"
case "${expected_arch}:${host_arch}" in
  x64:x86_64|amd64:x86_64|arm64:aarch64|arm64:arm64|aarch64:aarch64|aarch64:arm64)
    can_execute_bundled_tools="true"
    ;;
esac
if [ "${can_execute_bundled_tools}" = "true" ]; then
  "${portable_root}/tools/ffmpeg" -version >/dev/null
  "${portable_root}/tools/ffprobe" -version >/dev/null
fi

if command -v file >/dev/null 2>&1; then
  assert_file_arch() {
    local path="$1"
    local pattern="$2"
    if ! file "${path}" | grep -Eq "${pattern}"; then
      echo "${path} does not match expected Linux architecture ${expected_arch}." >&2
      file "${path}" >&2
      exit 1
    fi
  }
  case "${expected_arch}" in
    x64|amd64)
      arch_pattern="ELF 64-bit.*(x86-64|x86_64)"
      ;;
    arm64|aarch64)
      arch_pattern="ELF 64-bit.*(ARM aarch64|ARM64|aarch64)"
      ;;
    *)
      echo "Unsupported Linux architecture check: ${expected_arch}" >&2
      exit 2
      ;;
  esac
  assert_file_arch "${portable_root}/recordingfreedom" "${arch_pattern}"
  assert_file_arch "${portable_root}/tools/ffmpeg" "${arch_pattern}"
  assert_file_arch "${portable_root}/tools/ffprobe" "${arch_pattern}"
fi

echo "Linux portable archive verified: ${archive_path}"
