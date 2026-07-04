#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -lt 1 ] || [ "$#" -gt 2 ]; then
  echo "Usage: $0 <RecordingFreedom-macos-*.zip> [x64|arm64]" >&2
  exit 2
fi

zip_path="$1"
expected_arch="${2:-}"
if [ ! -s "${zip_path}" ]; then
  echo "macOS app zip is missing or empty: ${zip_path}" >&2
  exit 1
fi

work_dir="$(mktemp -d "${TMPDIR:-/tmp}/recordingfreedom-macos-verify.XXXXXX")"
cleanup() {
  rm -rf "${work_dir}"
}
trap cleanup EXIT

ditto -x -k "${zip_path}" "${work_dir}"
app_path="$(find "${work_dir}" -maxdepth 2 -type d -name "RecordingFreedom.app" -print -quit)"
if [ -z "${app_path}" ]; then
  echo "RecordingFreedom.app was not found in ${zip_path}" >&2
  exit 1
fi

main_binary="${app_path}/Contents/MacOS/recordingfreedom"
info_plist="${app_path}/Contents/Info.plist"
tools_dir="${app_path}/Contents/MacOS/tools"

test -x "${main_binary}"
test -f "${info_plist}"
test -x "${tools_dir}/ffmpeg"
test -x "${tools_dir}/ffprobe"
test -s "${tools_dir}/THIRD_PARTY_FFMPEG.txt"
test -s "${tools_dir}/THIRD_PARTY_NOTICES.txt"

if ! /usr/libexec/PlistBuddy -c "Print :CFBundleExecutable" "${info_plist}" | grep -qx "recordingfreedom"; then
  echo "CFBundleExecutable must be recordingfreedom" >&2
  exit 1
fi

"${tools_dir}/ffmpeg" -version >/dev/null
"${tools_dir}/ffprobe" -version >/dev/null

if [ -n "${expected_arch}" ]; then
  assert_file_arch() {
    local path="$1"
    local pattern="$2"
    if ! file "${path}" | grep -Eq "${pattern}"; then
      echo "${path} does not match expected macOS architecture ${expected_arch}." >&2
      file "${path}" >&2
      exit 1
    fi
  }
  case "${expected_arch}" in
    x64|amd64)
      assert_file_arch "${main_binary}" "x86_64"
      assert_file_arch "${tools_dir}/ffmpeg" "x86_64"
      assert_file_arch "${tools_dir}/ffprobe" "x86_64"
      ;;
    arm64)
      assert_file_arch "${main_binary}" "arm64"
      assert_file_arch "${tools_dir}/ffmpeg" "arm64"
      assert_file_arch "${tools_dir}/ffprobe" "arm64"
      ;;
    *)
      echo "Unsupported macOS architecture check: ${expected_arch}" >&2
      exit 2
      ;;
  esac
fi

echo "macOS app zip verified: ${zip_path}"
