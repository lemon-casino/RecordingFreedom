#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -lt 1 ] || [ "$#" -gt 3 ]; then
  echo "Usage: $0 <RecordingFreedom-macos-*.zip> [x64|arm64] [ppocrv5-mobile-zh-en-*.zip|model-dir]" >&2
  exit 2
fi

zip_path="$1"
expected_arch="${2:-}"
ocr_model_package="${3:-}"
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
expected_goarch="amd64"
if [ "${expected_arch}" = "arm64" ]; then
  expected_goarch="arm64"
fi
ocr_worker="${tools_dir}/ocr-worker/darwin-${expected_goarch}/rf-ocr-worker"
onnx_runtime="${tools_dir}/onnxruntime/darwin-${expected_goarch}"

resolve_ocr_model_dir() {
  local package_path="$1"
  local scratch_root="$2"
  if [ -z "${package_path}" ]; then
    printf ''
    return 0
  fi
  if [ ! -e "${package_path}" ]; then
    echo "OCR model package path does not exist: ${package_path}" >&2
    exit 1
  fi
  if [ -d "${package_path}" ]; then
    if [ -f "${package_path}/ppocrv5-mobile-zh-en/manifest.json" ]; then
      printf '%s' "${package_path}/ppocrv5-mobile-zh-en"
      return 0
    fi
    if [ -f "${package_path}/manifest.json" ]; then
      printf '%s' "${package_path}"
      return 0
    fi
    echo "OCR model package directory is missing ppocrv5-mobile-zh-en/manifest.json: ${package_path}" >&2
    exit 1
  fi
  case "${package_path}" in
    *.zip) ;;
    *)
      echo "OCR model package must be a .zip file or extracted model directory: ${package_path}" >&2
      exit 1
      ;;
  esac
  local extract_dir="${scratch_root}/ocr-model-package"
  mkdir -p "${extract_dir}"
  unzip -q "${package_path}" -d "${extract_dir}"
  local model_dir="${extract_dir}/ppocrv5-mobile-zh-en"
  if [ ! -f "${model_dir}/manifest.json" ]; then
    echo "OCR model package zip is missing ppocrv5-mobile-zh-en/manifest.json: ${package_path}" >&2
    exit 1
  fi
  for required in det.onnx cls.onnx rec.onnx keys.txt smoke.png smoke.expected.json; do
    if [ ! -s "${model_dir}/${required}" ]; then
      echo "OCR model package is missing required smoke file: ppocrv5-mobile-zh-en/${required}" >&2
      exit 1
    fi
  done
  printf '%s' "${model_dir}"
}

run_ocr_stable_smoke() {
  local model_dir="$1"
  local smoke_output
  smoke_output="$("${ocr_worker}" --smoke --runtime-dir "${onnx_runtime}" --model-dir "${model_dir}" --must-contain RecordingFreedom --must-contain 文字识别)"
  if ! printf '%s' "${smoke_output}" | grep -Eq '"ok"[[:space:]]*:[[:space:]]*true'; then
    echo "OCR worker stable model smoke failed." >&2
    printf '%s\n' "${smoke_output}" >&2
    exit 1
  fi
  if ! printf '%s' "${smoke_output}" | grep -q "RecordingFreedom"; then
    echo "OCR worker stable model smoke did not recognize RecordingFreedom." >&2
    printf '%s\n' "${smoke_output}" >&2
    exit 1
  fi
  if ! printf '%s' "${smoke_output}" | grep -q "文字识别"; then
    echo "OCR worker stable model smoke did not recognize 文字识别." >&2
    printf '%s\n' "${smoke_output}" >&2
    exit 1
  fi
  echo "OCR worker stable model smoke verified."
}

test -x "${main_binary}"
test -f "${info_plist}"
test -x "${tools_dir}/ffmpeg"
test -x "${tools_dir}/ffprobe"
test -x "${tools_dir}/ocr-desktop-evidence-export"
test -x "${tools_dir}/ocr-desktop-evidence-check"
test -x "${tools_dir}/ocr-desktop-evidence-plan"
test -x "${tools_dir}/ocr-desktop-evidence-session"
test -x "${tools_dir}/ocr-translation-smoke"
test -x "${tools_dir}/ocr-secret-store-smoke"
test -x "${ocr_worker}"
test -s "${onnx_runtime}/libonnxruntime.dylib"
test -s "${onnx_runtime}/libonnxruntime.1.23.2.dylib"
test -s "${onnx_runtime}/THIRD_PARTY_ONNXRUNTIME.txt"
test -s "${tools_dir}/librnnoise.dylib"
test -s "${tools_dir}/THIRD_PARTY_FFMPEG.txt"
test -s "${tools_dir}/THIRD_PARTY_NOTICES.txt"

if ! /usr/libexec/PlistBuddy -c "Print :CFBundleExecutable" "${info_plist}" | grep -qx "recordingfreedom"; then
  echo "CFBundleExecutable must be recordingfreedom" >&2
  exit 1
fi

"${tools_dir}/ffmpeg" -version >/dev/null
"${tools_dir}/ffprobe" -version >/dev/null
ocr_capabilities="$("${ocr_worker}" --capabilities --runtime-dir "${onnx_runtime}")"
if ! printf '%s' "${ocr_capabilities}" | grep -q '"runtimeAvailable":true'; then
  echo "OCR worker did not detect bundled ONNX Runtime at ${onnx_runtime}" >&2
  printf '%s\n' "${ocr_capabilities}" >&2
  exit 1
fi
if [ -n "${ocr_model_package}" ]; then
  ocr_model_dir="$(resolve_ocr_model_dir "${ocr_model_package}" "${work_dir}")"
  run_ocr_stable_smoke "${ocr_model_dir}"
fi

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
      assert_file_arch "${tools_dir}/ocr-desktop-evidence-export" "x86_64"
      assert_file_arch "${tools_dir}/ocr-desktop-evidence-check" "x86_64"
      assert_file_arch "${tools_dir}/ocr-desktop-evidence-plan" "x86_64"
      assert_file_arch "${tools_dir}/ocr-desktop-evidence-session" "x86_64"
      assert_file_arch "${tools_dir}/ocr-translation-smoke" "x86_64"
      assert_file_arch "${tools_dir}/ocr-secret-store-smoke" "x86_64"
      assert_file_arch "${ocr_worker}" "x86_64"
      assert_file_arch "${onnx_runtime}/libonnxruntime.dylib" "x86_64"
      assert_file_arch "${tools_dir}/librnnoise.dylib" "x86_64"
      ;;
    arm64)
      assert_file_arch "${main_binary}" "arm64"
      assert_file_arch "${tools_dir}/ffmpeg" "arm64"
      assert_file_arch "${tools_dir}/ffprobe" "arm64"
      assert_file_arch "${tools_dir}/ocr-desktop-evidence-export" "arm64"
      assert_file_arch "${tools_dir}/ocr-desktop-evidence-check" "arm64"
      assert_file_arch "${tools_dir}/ocr-desktop-evidence-plan" "arm64"
      assert_file_arch "${tools_dir}/ocr-desktop-evidence-session" "arm64"
      assert_file_arch "${tools_dir}/ocr-translation-smoke" "arm64"
      assert_file_arch "${tools_dir}/ocr-secret-store-smoke" "arm64"
      assert_file_arch "${ocr_worker}" "arm64"
      assert_file_arch "${onnx_runtime}/libonnxruntime.dylib" "arm64"
      assert_file_arch "${tools_dir}/librnnoise.dylib" "arm64"
      ;;
    *)
      echo "Unsupported macOS architecture check: ${expected_arch}" >&2
      exit 2
      ;;
  esac
fi

echo "macOS app zip verified: ${zip_path}"
