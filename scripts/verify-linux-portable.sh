#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -lt 1 ] || [ "$#" -gt 3 ]; then
  echo "Usage: $0 <RecordingFreedom-linux-*.tar.gz> [x64|arm64] [ppocrv5-mobile-zh-en-*.zip|model-dir]" >&2
  exit 2
fi

archive_path="$1"
expected_arch="${2:-x64}"
ocr_model_package="${3:-}"
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
expected_goarch="amd64"
if [ "${expected_arch}" = "arm64" ] || [ "${expected_arch}" = "aarch64" ]; then
  expected_goarch="arm64"
fi
ocr_worker="${portable_root}/tools/ocr-worker/linux-${expected_goarch}/rf-ocr-worker"
onnx_runtime="${portable_root}/tools/onnxruntime/linux-${expected_goarch}"

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

test -x "${portable_root}/recordingfreedom"
test -x "${portable_root}/tools/ffmpeg"
test -x "${portable_root}/tools/ffprobe"
test -x "${portable_root}/tools/pip-export-smoke"
test -x "${portable_root}/tools/ocr-desktop-evidence-export"
test -x "${portable_root}/tools/ocr-desktop-evidence-check"
test -x "${portable_root}/tools/ocr-desktop-evidence-plan"
test -x "${portable_root}/tools/ocr-desktop-evidence-session"
test -x "${portable_root}/tools/ocr-translation-smoke"
test -x "${portable_root}/tools/ocr-secret-store-smoke"
test -x "${ocr_worker}"
test -s "${onnx_runtime}/libonnxruntime.so"
test -s "${onnx_runtime}/libonnxruntime.so.1"
test -s "${onnx_runtime}/libonnxruntime.so.1.23.2"
test -s "${onnx_runtime}/libonnxruntime_providers_shared.so"
test -s "${onnx_runtime}/THIRD_PARTY_ONNXRUNTIME.txt"
test -s "${portable_root}/tools/librnnoise.so"
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
elif [ -n "${ocr_model_package}" ]; then
  echo "Skipping OCR stable model smoke for Linux ${expected_arch} on non-compatible host ${host_arch}."
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
  assert_file_arch "${portable_root}/tools/pip-export-smoke" "${arch_pattern}"
  assert_file_arch "${portable_root}/tools/ocr-desktop-evidence-export" "${arch_pattern}"
  assert_file_arch "${portable_root}/tools/ocr-desktop-evidence-check" "${arch_pattern}"
  assert_file_arch "${portable_root}/tools/ocr-desktop-evidence-plan" "${arch_pattern}"
  assert_file_arch "${portable_root}/tools/ocr-desktop-evidence-session" "${arch_pattern}"
  assert_file_arch "${portable_root}/tools/ocr-translation-smoke" "${arch_pattern}"
  assert_file_arch "${portable_root}/tools/ocr-secret-store-smoke" "${arch_pattern}"
  assert_file_arch "${ocr_worker}" "${arch_pattern}"
  assert_file_arch "${onnx_runtime}/libonnxruntime.so" "${arch_pattern}"
  assert_file_arch "${portable_root}/tools/librnnoise.so" "${arch_pattern}"
fi

echo "Linux portable archive verified: ${archive_path}"
