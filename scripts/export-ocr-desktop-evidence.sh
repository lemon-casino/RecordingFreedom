#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat >&2 <<'USAGE'
Usage: export-ocr-desktop-evidence.sh --visual-dir DIR [options]

Options:
  --evidence-dir DIR          Output evidence directory. Defaults to release-out/ocr-desktop-evidence/<timestamp>.
  --data-root DIR             RecordingFreedom data root. Defaults to the appdata service root.
  --platform-file FILE        platform.txt captured during the real desktop run.
  --version VALUE             Version under test. Defaults to manual.
  --commit VALUE              Commit under test. Defaults to git rev-parse --short HEAD or unknown.
  --artifact VALUE            Artifact/run description.
  --known-failures VALUE      Known failures string.
  --display-count VALUE       Display count when --platform-file is not provided.
  --display-resolution VALUE  Display resolution such as 1920x1080 when --platform-file is not provided.
  --display-scale VALUE       Display scale/DPI when --platform-file is not provided.
  --must-contain VALUE        OCR text that every required source kind must contain; may be repeated.
  --tools-dir DIR             Release/package tools dir containing ocr-desktop-evidence-export/check/plan/session.
  --require-translation       Require translations/*.json in the exported evidence.
  --skip-translations         Do not copy existing translation JSON files.
USAGE
}

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"
app_dir="${repo_root}/app"

abs_path() {
  case "$1" in
    /*) printf '%s\n' "$1" ;;
    *) printf '%s/%s\n' "$(pwd -P)" "$1" ;;
  esac
}

visual_dir=""
evidence_dir=""
data_root=""
platform_file=""
tools_dir=""
version="manual"
commit=""
artifact="manual desktop run"
known_failures="none"
display_count=""
display_resolution=""
display_scale="unknown"
require_translation=0
skip_translations=0
must_contains=()

while [ "$#" -gt 0 ]; do
  case "$1" in
    --visual-dir) visual_dir="${2:-}"; shift 2 ;;
    --evidence-dir) evidence_dir="${2:-}"; shift 2 ;;
    --data-root) data_root="${2:-}"; shift 2 ;;
    --platform-file) platform_file="${2:-}"; shift 2 ;;
    --version) version="${2:-}"; shift 2 ;;
    --commit) commit="${2:-}"; shift 2 ;;
    --artifact) artifact="${2:-}"; shift 2 ;;
    --known-failures) known_failures="${2:-}"; shift 2 ;;
    --display-count) display_count="${2:-}"; shift 2 ;;
    --display-resolution) display_resolution="${2:-}"; shift 2 ;;
    --display-scale) display_scale="${2:-}"; shift 2 ;;
    --must-contain) must_contains+=("${2:-}"); shift 2 ;;
    --tools-dir) tools_dir="${2:-}"; shift 2 ;;
    --require-translation) require_translation=1; shift ;;
    --skip-translations) skip_translations=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Unknown argument: $1" >&2; usage; exit 2 ;;
  esac
done

if [ -z "${visual_dir}" ]; then
  echo "--visual-dir is required and must contain real desktop visual screenshots." >&2
  usage
  exit 2
fi
visual_dir="$(abs_path "${visual_dir}")"
if [ ! -d "${visual_dir}" ]; then
  echo "Visual evidence directory does not exist: ${visual_dir}" >&2
  exit 1
fi
use_packaged_tools=0
export_tool=""
check_tool=""
plan_tool=""
session_tool=""
if [ -n "${tools_dir}" ]; then
  use_packaged_tools=1
  tools_dir="$(abs_path "${tools_dir}")"
  if [ ! -d "${tools_dir}" ]; then
    echo "RecordingFreedom tools directory was not found: ${tools_dir}" >&2
    exit 1
  fi
  export_tool="${tools_dir}/ocr-desktop-evidence-export"
  check_tool="${tools_dir}/ocr-desktop-evidence-check"
  plan_tool="${tools_dir}/ocr-desktop-evidence-plan"
  session_tool="${tools_dir}/ocr-desktop-evidence-session"
  if [ ! -x "${export_tool}" ]; then
    echo "RecordingFreedom tools directory is missing executable ocr-desktop-evidence-export: ${tools_dir}" >&2
    exit 1
  fi
  if [ ! -x "${check_tool}" ]; then
    echo "RecordingFreedom tools directory is missing executable ocr-desktop-evidence-check: ${tools_dir}" >&2
    exit 1
  fi
  if [ ! -x "${plan_tool}" ]; then
    echo "RecordingFreedom tools directory is missing executable ocr-desktop-evidence-plan: ${tools_dir}" >&2
    exit 1
  fi
  if [ ! -x "${session_tool}" ]; then
    echo "RecordingFreedom tools directory is missing executable ocr-desktop-evidence-session: ${tools_dir}" >&2
    exit 1
  fi
else
  if ! command -v go >/dev/null 2>&1; then
    echo "Required command 'go' was not found in PATH." >&2
    exit 1
  fi
  if [ ! -d "${app_dir}" ]; then
    echo "RecordingFreedom app directory was not found: ${app_dir}" >&2
    exit 1
  fi
fi

if [ -z "${evidence_dir}" ]; then
  evidence_dir="${repo_root}/release-out/ocr-desktop-evidence/$(date +%Y%m%d-%H%M%S)"
else
  evidence_dir="$(abs_path "${evidence_dir}")"
fi
if [ -n "${data_root}" ]; then
  data_root="$(abs_path "${data_root}")"
fi
if [ -z "${commit}" ]; then
  if command -v git >/dev/null 2>&1; then
    commit="$(git -C "${repo_root}" rev-parse --short HEAD 2>/dev/null || true)"
  fi
  if [ -z "${commit}" ]; then
    commit="unknown"
  fi
fi

detect_platform_file() {
  local target="$1"
  local os_name
  local os_version
  local arch
  os_name="$(uname -s)"
  arch="$(uname -m)"
  os_version="$(uname -a)"

  if [ -z "${display_resolution}" ]; then
    if [ "${os_name}" = "Darwin" ] && command -v system_profiler >/dev/null 2>&1; then
      display_resolution="$(system_profiler SPDisplaysDataType 2>/dev/null | awk '/Resolution:/{print $2 "x" $4; exit}')"
      if [ -z "${display_count}" ]; then
        display_count="$(system_profiler SPDisplaysDataType 2>/dev/null | awk '/Resolution:/{count++} END{if (count > 0) print count}')"
      fi
      if [ -z "${display_scale}" ] || [ "${display_scale}" = "unknown" ]; then
        display_scale="$(system_profiler SPDisplaysDataType 2>/dev/null | awk -F': ' '/Retina:/{print "retina=" $2; exit}')"
      fi
    elif command -v xrandr >/dev/null 2>&1; then
      display_resolution="$(xrandr --current 2>/dev/null | awk '/ connected/{if (match($0, /[0-9]+x[0-9]+\+[0-9]+\+[0-9]+/)) {value=substr($0, RSTART, RLENGTH); sub(/\+.*/, "", value); print value; exit}}')"
      if [ -z "${display_count}" ]; then
        display_count="$(xrandr --current 2>/dev/null | awk '/ connected/{count++} END{if (count > 0) print count}')"
      fi
    fi
  fi
  if [ -z "${display_count}" ]; then
    display_count="unknown"
  fi
  if [ -z "${display_scale}" ]; then
    display_scale="unknown"
  fi
  if [ -z "${display_resolution}" ]; then
    echo "Could not infer display resolution. Pass --platform-file or --display-count/--display-resolution/--display-scale." >&2
    exit 1
  fi

  if [ "${os_name}" = "Darwin" ] && command -v sw_vers >/dev/null 2>&1; then
    os_version="$(sw_vers -productName) $(sw_vers -productVersion) build $(sw_vers -buildVersion)"
  fi
  {
    printf 'operating system: %s\n' "${os_name}"
    printf 'version: %s\n' "${os_version}"
    printf 'architecture: %s\n' "${arch}"
    printf 'display count: %s\n' "${display_count}"
    printf 'resolution: %s\n' "${display_resolution}"
    printf 'scale: %s\n' "${display_scale}"
  } > "${target}"
}

generated_platform=""
cleanup() {
  if [ -n "${generated_platform}" ] && [ -f "${generated_platform}" ]; then
    rm -f "${generated_platform}"
  fi
}
trap cleanup EXIT

if [ -z "${platform_file}" ]; then
  generated_platform="$(mktemp "${TMPDIR:-/tmp}/recordingfreedom-ocr-platform.XXXXXX")"
  detect_platform_file "${generated_platform}"
  platform_file="${generated_platform}"
else
  platform_file="$(abs_path "${platform_file}")"
fi
if [ ! -f "${platform_file}" ]; then
  echo "Platform file does not exist: ${platform_file}" >&2
  exit 1
fi

export_args=(
  "-evidence-dir" "${evidence_dir}"
  "-visual-dir" "${visual_dir}"
  "-platform-file" "${platform_file}"
  "-version" "${version}"
  "-commit" "${commit}"
  "-artifact" "${artifact}"
  "-known-failures" "${known_failures}"
)
if [ -n "${data_root}" ]; then
  export_args+=("-data-root" "${data_root}")
fi
if [ "${skip_translations}" -eq 1 ]; then
  export_args+=("-include-translations=false")
fi

check_args=("-evidence-dir" "${evidence_dir}")
if [ "${require_translation}" -eq 1 ]; then
  check_args+=("-require-translation")
fi
for text in "${must_contains[@]}"; do
  if [ -n "${text}" ]; then
    check_args+=("-must-contain" "${text}")
  fi
done
check_report="${evidence_dir}/check-report.json"
plan_args=(
  "-visual-dir" "${visual_dir}"
  "-out-dir" "${evidence_dir}"
  "-check"
)
if [ -n "${data_root}" ]; then
  plan_args+=("-data-root" "${data_root}")
fi
echo "OCR desktop visual capture checklist will be written as visual-capture-checklist.md/json in ${evidence_dir}"

if [ "${use_packaged_tools}" -eq 1 ]; then
  "${plan_tool}" "${plan_args[@]}"
  "${export_tool}" "${export_args[@]}"
  set +e
  "${check_tool}" "${check_args[@]}" > "${check_report}"
  check_status=$?
  set -e
  cat "${check_report}"
  if [ "${check_status}" -ne 0 ]; then
    echo "ocr-desktop-evidence-check failed with exit code ${check_status}; report saved to ${check_report}" >&2
    exit "${check_status}"
  fi
else
  (
    cd "${app_dir}"
    go run "./cmd/ocr-desktop-evidence-plan" "${plan_args[@]}"
    go run "./cmd/ocr-desktop-evidence-export" "${export_args[@]}"
    set +e
    go run "./cmd/ocr-desktop-evidence-check" "${check_args[@]}" > "${check_report}"
    check_status=$?
    set -e
    cat "${check_report}"
    if [ "${check_status}" -ne 0 ]; then
      echo "ocr-desktop-evidence-check failed with exit code ${check_status}; report saved to ${check_report}" >&2
      exit "${check_status}"
    fi
  )
fi

echo "OCR desktop evidence exported and checked: ${evidence_dir}"
