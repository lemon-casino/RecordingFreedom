#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -ne 1 ]; then
  echo "Usage: $0 <RecordingFreedom-macos-*.zip>" >&2
  exit 2
fi

zip_path="$1"
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

echo "macOS app zip verified: ${zip_path}"
