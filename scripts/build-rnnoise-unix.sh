#!/usr/bin/env bash
set -euo pipefail

platform=""
arch=""
output=""

usage() {
  echo "Usage: $0 --platform linux|macos --arch x64|arm64 [--output <path>] [--force]" >&2
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --platform)
      platform="${2:-}"
      shift 2
      ;;
    --arch)
      arch="${2:-}"
      shift 2
      ;;
    --output)
      output="${2:-}"
      shift 2
      ;;
    --force)
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage
      exit 2
      ;;
  esac
done

case "${platform}" in
  linux|macos) ;;
  *)
    echo "--platform must be linux or macos, got ${platform}" >&2
    usage
    exit 2
    ;;
esac

case "${arch}" in
  x64|amd64) expected_arch="x64" ;;
  arm64|aarch64) expected_arch="arm64" ;;
  *)
    echo "--arch must be x64 or arm64, got ${arch}" >&2
    usage
    exit 2
    ;;
esac

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
root="$(cd "${script_dir}/.." && pwd)"
native_dir="${root}/app/internal/audio/rnnoise/native"
if [ -z "${output}" ]; then
  if [ "${platform}" = "macos" ]; then
    output="${root}/app/tools/librnnoise.dylib"
  else
    output="${root}/app/tools/librnnoise.so"
  fi
fi
mkdir -p "$(dirname "${output}")"

cc="${CC:-}"
if [ -z "${cc}" ]; then
  if command -v clang >/dev/null 2>&1; then
    cc="clang"
  elif command -v gcc >/dev/null 2>&1; then
    cc="gcc"
  elif command -v cc >/dev/null 2>&1; then
    cc="cc"
  else
    echo "No C compiler found. Install clang or gcc before building RNNoise." >&2
    exit 1
  fi
fi

sources=(
  "${native_dir}/likely_voice_enhancer.c"
  "${native_dir}/denoise.c"
  "${native_dir}/rnn.c"
  "${native_dir}/rnn_data.c"
  "${native_dir}/pitch.c"
  "${native_dir}/celt_lpc.c"
  "${native_dir}/kiss_fft.c"
)

args=(
  -shared
  -fPIC
  -O2
  -std=c99
  -D_GNU_SOURCE
  -DRNNOISE_BUILD
  -DLIKELY_VOICE_ENHANCER_BUILD_DLL
  "-I${native_dir}"
)

if [ "${platform}" = "macos" ]; then
  args+=(-Wl,-install_name,@rpath/librnnoise.dylib)
fi

args+=("${sources[@]}" -lm -o "${output}")

echo "Building RNNoise dynamic module: ${cc} ${args[*]}"
"${cc}" "${args[@]}"

if [ ! -s "${output}" ]; then
  echo "RNNoise dynamic module was not produced: ${output}" >&2
  exit 1
fi

if command -v file >/dev/null 2>&1; then
  case "${platform}:${expected_arch}" in
    macos:x64) pattern="Mach-O.*x86_64" ;;
    macos:arm64) pattern="Mach-O.*arm64" ;;
    linux:x64) pattern="ELF 64-bit.*(x86-64|x86_64)" ;;
    linux:arm64) pattern="ELF 64-bit.*(ARM aarch64|ARM64|aarch64)" ;;
  esac
  if ! file "${output}" | grep -Eq "${pattern}"; then
    echo "${output} does not match expected ${platform} ${expected_arch} architecture." >&2
    file "${output}" >&2
    exit 1
  fi
fi

echo "RNNoise dynamic module built: ${output}"
