#!/usr/bin/env bash
set -euo pipefail

destination_dir=""
platform=""
arch=""
force="false"

while [ "$#" -gt 0 ]; do
  case "$1" in
    --destination-dir)
      destination_dir="${2:-}"
      shift 2
      ;;
    --platform)
      platform="${2:-}"
      shift 2
      ;;
    --arch)
      arch="${2:-}"
      shift 2
      ;;
    --force)
      force="true"
      shift
      ;;
    *)
      echo "Unknown argument: $1" >&2
      exit 2
      ;;
  esac
done

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
if [ -z "${destination_dir}" ]; then
  destination_dir="${repo_root}/app/tools"
elif [[ "${destination_dir}" != /* ]]; then
  destination_dir="$(pwd)/${destination_dir}"
fi

if [ -z "${platform}" ]; then
  case "$(uname -s)" in
    Darwin) platform="macos" ;;
    Linux) platform="linux" ;;
    *) echo "Cannot infer platform. Pass --platform macos|linux." >&2; exit 2 ;;
  esac
fi

if [ -z "${arch}" ]; then
  case "$(uname -m)" in
    arm64|aarch64) arch="arm64" ;;
    x86_64|amd64) arch="x64" ;;
    *) echo "Cannot infer architecture. Pass --arch arm64|x64." >&2; exit 2 ;;
  esac
fi

case "${platform}-${arch}" in
  macos-arm64)
    ffmpeg_asset="ffmpeg-osx-arm64"
    ffprobe_asset="ffprobe-osx-arm64"
    ffmpeg_sha256="e7b9fcd97f95f333512d6e8b8ac24d9dbc08f189f36047695499bd7b57214b22"
    ffprobe_sha256="ded4c698b8ff38d0bc1fd30fcc5e768dc46f58bc15a8dfd61f98615ba49cde5c"
    ;;
  linux-x64|linux-amd64)
    ffmpeg_asset="ffmpeg-linux-x64"
    ffprobe_asset="ffprobe-linux-x64"
    ffmpeg_sha256="9eac5b2b5076db5ff853a6fa0dcd6b8de7d0cac8481eadda6c47cd935825f1ee"
    ffprobe_sha256="065d3c56926052a76e884c4e4b51b7d95248da9391ab7effdcca6b94ceab98cf"
    ;;
  *)
    echo "Unsupported FFmpeg bundle target: platform=${platform} arch=${arch}" >&2
    exit 2
    ;;
esac

shaka_tag="n8.1.2-1"
base_url="https://github.com/shaka-project/static-ffmpeg-binaries/releases/download/${shaka_tag}"
mkdir -p "${destination_dir}"

ffmpeg_path="${destination_dir}/ffmpeg"
ffprobe_path="${destination_dir}/ffprobe"
if [ "${force}" != "true" ] && [ -x "${ffmpeg_path}" ] && [ -x "${ffprobe_path}" ]; then
  if "${ffmpeg_path}" -version >/dev/null 2>&1 && "${ffprobe_path}" -version >/dev/null 2>&1; then
    echo "Using existing FFmpeg tools in ${destination_dir}"
    exit 0
  fi
fi

work_dir="$(mktemp -d "${TMPDIR:-/tmp}/recordingfreedom-ffmpeg.XXXXXX")"
cleanup() {
  rm -rf "${work_dir}"
}
trap cleanup EXIT

download_asset() {
  local asset="$1"
  local sha256="$2"
  local output="$3"
  local url="${base_url}/${asset}"

  echo "Downloading ${asset}"
  curl -fL --retry 5 --retry-delay 5 --connect-timeout 30 -o "${output}" "${url}"
  local actual
  if command -v sha256sum >/dev/null 2>&1; then
    actual="$(sha256sum "${output}" | awk '{print tolower($1)}')"
  else
    actual="$(shasum -a 256 "${output}" | awk '{print tolower($1)}')"
  fi
  if [ "${actual}" != "${sha256}" ]; then
    echo "SHA256 mismatch for ${asset}. Expected ${sha256}, got ${actual}" >&2
    exit 1
  fi
  chmod +x "${output}"
}

download_asset "${ffmpeg_asset}" "${ffmpeg_sha256}" "${work_dir}/ffmpeg"
download_asset "${ffprobe_asset}" "${ffprobe_sha256}" "${work_dir}/ffprobe"

install -m 0755 "${work_dir}/ffmpeg" "${ffmpeg_path}"
install -m 0755 "${work_dir}/ffprobe" "${ffprobe_path}"

"${ffmpeg_path}" -version >/dev/null
"${ffprobe_path}" -version >/dev/null

cat > "${destination_dir}/THIRD_PARTY_FFMPEG.txt" <<NOTICE
RecordingFreedom bundled FFmpeg dependency

Source: ${base_url}
FFmpeg asset: ${ffmpeg_asset}
FFmpeg SHA256: ${ffmpeg_sha256}
FFprobe asset: ${ffprobe_asset}
FFprobe SHA256: ${ffprobe_sha256}
RetrievedAtUtc: $(date -u +"%Y-%m-%dT%H:%M:%SZ")

FFmpeg is provided by its upstream/build distribution and is governed by
the license terms shipped by that distribution and by the FFmpeg project.
Review FFmpeg licensing before publishing a public, signed release.
NOTICE

echo "Bundled FFmpeg tools ready in ${destination_dir}"
