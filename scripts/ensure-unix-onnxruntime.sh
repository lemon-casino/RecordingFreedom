#!/usr/bin/env bash
set -euo pipefail

destination_root=""
platform=""
arch=""
manifest_path=""
force="false"

while [ "$#" -gt 0 ]; do
  case "$1" in
    --destination-root)
      destination_root="${2:-}"
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
    --manifest)
      manifest_path="${2:-}"
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
if [ -z "${destination_root}" ]; then
  destination_root="${repo_root}/app/tools/onnxruntime"
elif [[ "${destination_root}" != /* ]]; then
  destination_root="$(pwd)/${destination_root}"
fi
if [ -z "${manifest_path}" ]; then
  manifest_path="${repo_root}/third_party/onnxruntime/manifest.json"
elif [[ "${manifest_path}" != /* ]]; then
  manifest_path="$(pwd)/${manifest_path}"
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

case "${platform}" in
  macos|darwin) goos="darwin" ;;
  linux) goos="linux" ;;
  *) echo "Unsupported ONNX Runtime platform: ${platform}" >&2; exit 2 ;;
esac
case "${arch}" in
  x64|amd64) goarch="amd64" ;;
  arm64|aarch64) goarch="arm64" ;;
  *) echo "Unsupported ONNX Runtime architecture: ${arch}" >&2; exit 2 ;;
esac

target_key="${goos}-${goarch}"
target_dir="${destination_root}/${target_key}"

if ! command -v python3 >/dev/null 2>&1; then
  echo "python3 is required to read ${manifest_path}" >&2
  exit 2
fi

read_manifest_field() {
  local field="$1"
  python3 - "$manifest_path" "$target_key" "$field" <<'PY'
import json
import sys
manifest_path, target_key, field = sys.argv[1:4]
with open(manifest_path, "r", encoding="utf-8") as f:
    manifest = json.load(f)
target = manifest["targets"][target_key]
value = manifest[field] if field in manifest else target[field]
if isinstance(value, list):
    print("\n".join(str(item) for item in value))
else:
    print(value)
PY
}

version="$(read_manifest_field version)"
source="$(read_manifest_field source)"
license_name="$(read_manifest_field license)"
archive_name="$(read_manifest_field archiveName)"
archive_bytes="$(read_manifest_field archiveBytes)"
archive_sha256="$(read_manifest_field archiveSha256)"
download_url="$(read_manifest_field downloadUrl)"
required_library="$(read_manifest_field requiredLibrary)"
notice_path="${target_dir}/THIRD_PARTY_ONNXRUNTIME.txt"

if [ "${force}" != "true" ] && [ -s "${target_dir}/${required_library}" ] && [ -s "${notice_path}" ]; then
  if grep -q "Version: ${version}" "${notice_path}" && grep -q "Target: ${target_key}" "${notice_path}"; then
    echo "Using existing ONNX Runtime bundle: ${target_dir}"
    exit 0
  fi
fi

work_dir="$(mktemp -d "${TMPDIR:-/tmp}/recordingfreedom-onnxruntime.XXXXXX")"
cleanup() {
  rm -rf "${work_dir}"
}
trap cleanup EXIT

archive_path="${work_dir}/${archive_name}"

echo "Downloading ONNX Runtime ${version} for ${target_key}"
curl -fL --retry 5 --retry-delay 5 --connect-timeout 30 -o "${archive_path}" "${download_url}"

actual_bytes="$(wc -c < "${archive_path}" | tr -d ' ')"
if [ "${actual_bytes}" != "${archive_bytes}" ]; then
  echo "ONNX Runtime archive size mismatch for ${archive_name}. Expected ${archive_bytes}, got ${actual_bytes}" >&2
  exit 1
fi

if command -v sha256sum >/dev/null 2>&1; then
  actual_sha256="$(sha256sum "${archive_path}" | awk '{print tolower($1)}')"
else
  actual_sha256="$(shasum -a 256 "${archive_path}" | awk '{print tolower($1)}')"
fi
if [ "${actual_sha256}" != "${archive_sha256}" ]; then
  echo "ONNX Runtime SHA256 mismatch for ${archive_name}. Expected ${archive_sha256}, got ${actual_sha256}" >&2
  exit 1
fi

rm -rf "${target_dir}"
mkdir -p "${target_dir}"

python3 - "$archive_path" "$target_dir" "$manifest_path" "$target_key" <<'PY'
import json
import os
import posixpath
import stat
import sys
import tarfile

archive_path, target_dir, manifest_path, target_key = sys.argv[1:5]
with open(manifest_path, "r", encoding="utf-8") as f:
    manifest = json.load(f)
target = manifest["targets"][target_key]

def safe_name(name: str) -> str:
    normalized = posixpath.normpath(name.replace("\\", "/"))
    if normalized.startswith("../") or normalized == ".." or posixpath.isabs(normalized):
        raise RuntimeError(f"unsafe ONNX Runtime archive path: {name}")
    return normalized.lstrip("./")

def basename(name: str) -> str:
    return posixpath.basename(safe_name(name))

def member_score(member: tarfile.TarInfo, wanted: str) -> int:
    normalized = safe_name(member.name)
    if basename(normalized) != wanted:
        return -1
    if f"/lib/{wanted}" in f"/{normalized}":
        return 2
    return 1

member_by_normalized = {}

def resolve_member(tar: tarfile.TarFile, member: tarfile.TarInfo, seen: set[str]) -> tarfile.TarInfo:
    name = safe_name(member.name)
    if name in seen:
        raise RuntimeError(f"cyclic ONNX Runtime symlink: {member.name}")
    seen.add(name)
    if member.isfile():
        return member
    if member.issym():
        parent = posixpath.dirname(name)
        link_name = safe_name(posixpath.join(parent, member.linkname))
        linked = member_by_normalized.get(link_name)
        if linked is None:
            raise RuntimeError(f"ONNX Runtime symlink target was not found: {member.name} -> {member.linkname}")
        return resolve_member(tar, linked, seen)
    raise RuntimeError(f"unsupported ONNX Runtime archive member: {member.name}")

def copy_member(tar: tarfile.TarFile, wanted: str, required: bool) -> None:
    candidates = [m for m in tar.getmembers() if member_score(m, wanted) >= 0]
    candidates.sort(key=lambda item: member_score(item, wanted), reverse=True)
    if not candidates:
        if required:
            raise RuntimeError(f"Downloaded ONNX Runtime archive did not contain {wanted}")
        return
    source = resolve_member(tar, candidates[0], set())
    extracted = tar.extractfile(source)
    if extracted is None:
        raise RuntimeError(f"Could not read ONNX Runtime archive member {source.name}")
    output_path = os.path.join(target_dir, wanted)
    with extracted, open(output_path, "wb") as out:
        out.write(extracted.read())
    os.chmod(output_path, stat.S_IRUSR | stat.S_IWUSR | stat.S_IRGRP | stat.S_IROTH)

with tarfile.open(archive_path, "r:gz") as tar:
    member_by_normalized = {safe_name(member.name): member for member in tar.getmembers()}
    for library in target["libraryFiles"]:
        copy_member(tar, library, True)
    for optional in ["LICENSE", "ThirdPartyNotices.txt", "VERSION_NUMBER", "GIT_COMMIT_ID"]:
        copy_member(tar, optional, False)
PY

cat > "${notice_path}" <<NOTICE
RecordingFreedom bundled ONNX Runtime dependency

Name: ONNX Runtime CPU
Version: ${version}
Target: ${target_key}
Source: ${download_url}
Release: ${source}
License: ${license_name}
Archive: ${archive_name}
ArchiveSHA256: ${actual_sha256}
RetrievedAtUtc: $(date -u +"%Y-%m-%dT%H:%M:%SZ")

ONNX Runtime is provided by Microsoft under the license terms shipped in
this directory. Review LICENSE and ThirdPartyNotices.txt before publishing
a public, signed release.
NOTICE

test -s "${target_dir}/${required_library}"
echo "Bundled ONNX Runtime ready: ${target_dir}"
