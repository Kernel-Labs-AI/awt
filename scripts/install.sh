#!/usr/bin/env bash
set -euo pipefail

REPO_OWNER="kernel-labs-ai"
REPO_NAME="awt"
MODULE_PATH="github.com/kernel-labs-ai/awt/cmd/awt"
DEFAULT_INSTALL_DIR="/usr/local/bin"

version="latest"
method="auto"
install_dir="$DEFAULT_INSTALL_DIR"
os=""
arch=""

log() {
  printf "%s\n" "$*"
}

warn() {
  printf "Warning: %s\n" "$*" >&2
}

err() {
  printf "Error: %s\n" "$*" >&2
}

die() {
  err "$*"
  exit 1
}

usage() {
  cat <<"USAGE"
Install awt from GitHub Releases or source.

Usage:
  install.sh [--version <latest|vX.Y.Z>] [--os <linux|darwin>] [--arch <amd64|arm64>] [--install-dir <path>] [--method <auto|release|source>]

Flags:
  --version <value>      Release version to install (default: latest)
  --os <value>           Target OS: linux or darwin (default: detected from uname)
  --arch <value>         Target architecture: amd64 or arm64 (default: detected from uname)
  --install-dir <path>   Installation directory for awt binary (default: /usr/local/bin)
  --method <value>       Install method: auto, release, or source (default: auto)
  -h, --help             Show this help message
USAGE
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    err "Missing prerequisite: $1"
    return 1
  fi
}

normalize_os() {
  local value
  value="$(printf "%s" "$1" | tr "[:upper:]" "[:lower:]")"

  case "$value" in
    linux)
      printf "linux\n"
      ;;
    darwin)
      printf "darwin\n"
      ;;
    *)
      return 1
      ;;
  esac
}

normalize_arch() {
  local value
  value="$(printf "%s" "$1" | tr "[:upper:]" "[:lower:]")"

  case "$value" in
    amd64 | x86_64)
      printf "amd64\n"
      ;;
    arm64 | aarch64)
      printf "arm64\n"
      ;;
    *)
      return 1
      ;;
  esac
}
detect_os() {
  local detected
  detected="$(uname -s)"
  normalize_os "$detected" || return 1
}

detect_arch() {
  local detected
  detected="$(uname -m)"
  normalize_arch "$detected" || return 1
}

validate_version() {
  if [[ "$1" == "latest" ]]; then
    return 0
  fi

  [[ "$1" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]
}

asset_suffix() {
  case "$1/$2" in
    linux/amd64)
      printf "Linux_x86_64\n"
      ;;
    linux/arm64)
      printf "Linux_arm64\n"
      ;;
    darwin/amd64)
      printf "Darwin_x86_64\n"
      ;;
    darwin/arm64)
      printf "Darwin_arm64\n"
      ;;
    *)
      return 1
      ;;
  esac
}

release_base_url() {
  if [[ "$1" == "latest" ]]; then
    printf "https://github.com/%s/%s/releases/latest/download\n" "$REPO_OWNER" "$REPO_NAME"
    return 0
  fi

  printf "https://github.com/%s/%s/releases/download/%s\n" "$REPO_OWNER" "$REPO_NAME" "$1"
}

resolve_release_tag() {
  if [[ "$1" != "latest" ]]; then
    printf "%s\n" "$1"
    return 0
  fi

  require_cmd curl || return 1

  local final_url
  local tag
  final_url="$(curl -fsSL -o /dev/null -w "%{url_effective}" "https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/latest")" || {
    err "Failed to resolve latest release tag from GitHub"
    return 1
  }

  tag="${final_url##*/}"
  if ! validate_version "$tag"; then
    err "Unable to parse latest tag from URL: $final_url"
    return 1
  fi

  printf "%s\n" "$tag"
}

sha256_of() {
  local file="$1"

  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$file" | awk "{print \$1}"
    return 0
  fi

  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$file" | awk "{print \$1}"
    return 0
  fi

  err "Missing prerequisite: sha256sum or shasum"
  return 1
}

install_binary() {
  local src="$1"
  local dest="$install_dir/awt"
  local version_output

  if ! mkdir -p "$install_dir"; then
    err "Failed to create install directory: $install_dir (permission denied?)"
    return 1
  fi

  if ! install -m 0755 "$src" "$dest"; then
    err "Failed to install awt to $dest (permission denied?)"
    return 1
  fi

  if ! version_output="$($dest version 2>&1)"; then
    err "Sanity check failed: '$dest version' returned non-zero"
    return 1
  fi

  log "Sanity check passed: $version_output"
  log "Installed awt to $dest"
}

install_from_release() {
  local requested_version="$1"
  local base_url
  local suffix
  local archive_name
  local tmpdir
  local checksums_path
  local archive_path
  local expected_sha
  local actual_sha
  local binary_path

  require_cmd curl || return 1
  require_cmd tar || return 1
  require_cmd install || return 1

  if ! suffix="$(asset_suffix "$os" "$arch")"; then
    err "Unsupported OS/arch combination: $os/$arch"
    return 1
  fi

  base_url="$(release_base_url "$requested_version")"
  tmpdir="$(mktemp -d)" || {
    err "Failed to create temporary directory"
    return 1
  }

  checksums_path="$tmpdir/checksums.txt"

  if ! curl -fsSL "$base_url/checksums.txt" -o "$checksums_path"; then
    err "Failed to download checksums.txt from $base_url"
    rm -rf "$tmpdir"
    return 1
  fi

  if [[ "$requested_version" == "latest" ]]; then
    archive_name="$(awk -v suffix="_${suffix}.tar.gz" '$2 ~ /^awt_/ && $2 ~ suffix"$" { print $2; exit }' "$checksums_path")"
  else
    archive_name="awt_${requested_version#v}_${suffix}.tar.gz"
  fi

  if [[ -z "$archive_name" ]]; then
    err "Could not find release archive for $os/$arch in checksums.txt"
    rm -rf "$tmpdir"
    return 1
  fi

  expected_sha="$(awk -v target="$archive_name" '$2 == target { print $1; exit }' "$checksums_path")"
  if [[ -z "$expected_sha" ]]; then
    err "No checksum entry found for $archive_name"
    rm -rf "$tmpdir"
    return 1
  fi

  archive_path="$tmpdir/$archive_name"
  if ! curl -fsSL "$base_url/$archive_name" -o "$archive_path"; then
    err "Failed to download release artifact: $base_url/$archive_name"
    rm -rf "$tmpdir"
    return 1
  fi

  actual_sha="$(sha256_of "$archive_path")" || {
    rm -rf "$tmpdir"
    return 1
  }

  if [[ "$actual_sha" != "$expected_sha" ]]; then
    err "Checksum mismatch for $archive_name"
    err "Expected: $expected_sha"
    err "Actual:   $actual_sha"
    rm -rf "$tmpdir"
    return 1
  fi

  if ! tar -xzf "$archive_path" -C "$tmpdir"; then
    err "Failed to extract $archive_name"
    rm -rf "$tmpdir"
    return 1
  fi

  binary_path="$tmpdir/awt"
  if [[ ! -f "$binary_path" ]]; then
    binary_path="$(find "$tmpdir" -type f -name awt | head -n 1 || true)"
  fi

  if [[ -z "$binary_path" || ! -f "$binary_path" ]]; then
    err "Release archive did not contain an 'awt' binary"
    rm -rf "$tmpdir"
    return 1
  fi

  if ! install_binary "$binary_path"; then
    rm -rf "$tmpdir"
    return 1
  fi

  rm -rf "$tmpdir"
  log "Installed from GitHub Releases"
}

install_from_source() {
  local release_tag="$1"
  local tmpdir
  local binary_path

  require_cmd go || return 1
  require_cmd install || return 1

  tmpdir="$(mktemp -d)" || {
    err "Failed to create temporary directory"
    return 1
  }

  if ! GOBIN="$tmpdir" go install "${MODULE_PATH}@${release_tag}"; then
    err "go install failed for ${MODULE_PATH}@${release_tag}"
    rm -rf "$tmpdir"
    return 1
  fi

  binary_path="$tmpdir/awt"
  if [[ ! -f "$binary_path" ]]; then
    err "go install did not produce awt binary"
    rm -rf "$tmpdir"
    return 1
  fi

  if ! install_binary "$binary_path"; then
    rm -rf "$tmpdir"
    return 1
  fi

  rm -rf "$tmpdir"
  log "Installed from source (${release_tag})"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --version)
      [[ $# -ge 2 ]] || die "--version requires a value"
      version="$2"
      shift 2
      ;;
    --os)
      [[ $# -ge 2 ]] || die "--os requires a value"
      os="$2"
      shift 2
      ;;
    --arch)
      [[ $# -ge 2 ]] || die "--arch requires a value"
      arch="$2"
      shift 2
      ;;
    --install-dir)
      [[ $# -ge 2 ]] || die "--install-dir requires a value"
      install_dir="$2"
      shift 2
      ;;
    --method)
      [[ $# -ge 2 ]] || die "--method requires a value"
      method="$2"
      shift 2
      ;;
    -h | --help)
      usage
      exit 0
      ;;
    *)
      die "Unknown argument: $1"
      ;;
  esac
done

if [[ -z "$os" ]]; then
  os="$(detect_os)" || die "Unsupported OS from uname. Supported values: linux, darwin"
else
  raw_os="$os"
  os="$(normalize_os "$raw_os")" || die "Unsupported OS: $raw_os. Supported values: linux, darwin"
fi

if [[ -z "$arch" ]]; then
  arch="$(detect_arch)" || die "Unsupported architecture from uname. Supported values: amd64, arm64"
else
  raw_arch="$arch"
  arch="$(normalize_arch "$raw_arch")" || die "Unsupported architecture: $raw_arch. Supported values: amd64, arm64"
fi

case "$method" in
  auto | release | source)
    ;;
  *)
    die "Unsupported method: $method. Supported values: auto, release, source"
    ;;
esac

if ! validate_version "$version"; then
  die "Unsupported version: $version. Expected 'latest' or a tag like 'vX.Y.Z'"
fi

case "$method" in
  release)
    install_from_release "$version" || die "Release installation failed"
    ;;
  source)
    release_tag="$(resolve_release_tag "$version")" || die "Failed to resolve release tag for source install"
    install_from_source "$release_tag" || die "Source installation failed"
    ;;
  auto)
    if install_from_release "$version"; then
      exit 0
    fi

    warn "Release installation failed; falling back to source installation"
    release_tag="$(resolve_release_tag "$version")" || die "Failed to resolve release tag for source fallback"
    install_from_source "$release_tag" || die "Source fallback installation failed"
    ;;
esac
