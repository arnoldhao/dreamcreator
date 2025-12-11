#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<USAGE
Usage: $0 --platform <os/arch> [--ytdlp-version <version>] [--ffmpeg-version <version>] [--deno-version <version>]

Downloads (or prepares placeholders for) embedded dependencies required by the
backend/embedded package. The command must be run from the repository root.

Options:
  --platform         Target platform expressed as os/arch (e.g. windows/amd64)
  --ytdlp-version    Explicit yt-dlp version to download; defaults to the value
                     defined in backend/consts/mirrors.go
  --ffmpeg-version   Explicit FFmpeg version to download; defaults to the value
                     defined in backend/consts/mirrors.go for the target OS
  --deno-version     Explicit Deno version to download; defaults to the value
                     defined in backend/consts/mirrors.go
  -h, --help         Show this help and exit
USAGE
}

fail() {
  echo "[fetch-embedded-binaries] $1" >&2
  exit 1
}

PROJECT_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
BINARY_DIR="$PROJECT_ROOT/backend/embedded/binaries"
CONSTS_FILE="$PROJECT_ROOT/backend/consts/mirrors.go"

read_const() {
  local name="$1"
  local value
  value=$(grep -E "${name}[[:space:]]*=" "$CONSTS_FILE" | head -n1 | sed -E 's/.*"([^"]+)".*/\1/') || true
  echo "$value"
}

ensure_tools() {
  local missing=()
  for tool in "$@"; do
    if ! command -v "$tool" >/dev/null 2>&1; then
      missing+=("$tool")
    fi
  done
  if [[ ${#missing[@]} -gt 0 ]]; then
    fail "Missing required tools: ${missing[*]}"
  fi
}

create_placeholder() {
  local dest="$1"
  cat <<'PLACEHOLDER' >"$dest"
#!/usr/bin/env bash
cat <<'MSG'
This is a placeholder binary generated for CI builds running on Linux.
The real dependency binaries are bundled during platform-specific release jobs.
MSG
exit 1
PLACEHOLDER
  chmod +x "$dest"
}

PLATFORM=""
YTDLP_VERSION=""
FFMPEG_VERSION=""
DENO_VERSION=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --platform)
      [[ $# -ge 2 ]] || fail "--platform requires a value"
      PLATFORM="$2"
      shift 2
      ;;
    --ytdlp-version)
      [[ $# -ge 2 ]] || fail "--ytdlp-version requires a value"
      YTDLP_VERSION="$2"
      shift 2
      ;;
    --ffmpeg-version)
      [[ $# -ge 2 ]] || fail "--ffmpeg-version requires a value"
      FFMPEG_VERSION="$2"
      shift 2
      ;;
    --deno-version)
      [[ $# -ge 2 ]] || fail "--deno-version requires a value"
      DENO_VERSION="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      fail "Unknown argument: $1"
      ;;
  esac
done

[[ -n "$PLATFORM" ]] || { usage; fail "--platform is required"; }

IFS='/' read -r OS ARCH <<<"$PLATFORM" || fail "Invalid platform format: $PLATFORM"
[[ -n "$OS" && -n "$ARCH" ]] || fail "Invalid platform format: $PLATFORM"

mkdir -p "$BINARY_DIR"

if [[ -z "$YTDLP_VERSION" ]]; then
  YTDLP_VERSION=$(read_const "EMBEDDED_YTDLP_VERSION")
fi
[[ -n "$YTDLP_VERSION" ]] || fail "Unable to determine yt-dlp version"

if [[ -z "$DENO_VERSION" ]]; then
  DENO_VERSION=$(read_const "EMBEDDED_DENO_VERSION")
fi
[[ -n "$DENO_VERSION" ]] || fail "Unable to determine Deno version"

case "$OS" in
  windows)
    if [[ -z "$FFMPEG_VERSION" ]]; then
      FFMPEG_VERSION=$(read_const "EMBEDDED_FFMPEG_VERSION_WINDOWS")
    fi
    [[ -n "$FFMPEG_VERSION" ]] || fail "Unable to determine Windows FFmpeg version"
    ensure_tools curl unzip find chmod

    echo "Fetching yt-dlp ${YTDLP_VERSION} for windows/${ARCH}"
    ytdlp_url="https://github.com/yt-dlp/yt-dlp/releases/download/${YTDLP_VERSION}/yt-dlp.exe"
    ytdlp_dest="$BINARY_DIR/yt-dlp_${YTDLP_VERSION}_windows_${ARCH}.exe"
    curl -L "$ytdlp_url" -o "$ytdlp_dest"
    chmod +x "$ytdlp_dest"

    echo "Fetching FFmpeg ${FFMPEG_VERSION} for windows/${ARCH}"
    case "$ARCH" in
      amd64) ffmpeg_filename="jellyfin-ffmpeg_${FFMPEG_VERSION}_portable_win64-clang-gpl" ;;
      arm64) ffmpeg_filename="jellyfin-ffmpeg_${FFMPEG_VERSION}_portable_winarm64-clang-gpl" ;;
      *) fail "Unsupported Windows arch: $ARCH" ;;
    esac
    ffmpeg_url="https://gh-proxy.com/github.com/jellyfin/jellyfin-ffmpeg/releases/download/v${FFMPEG_VERSION}/${ffmpeg_filename}.zip"
    tmp_dir=$(mktemp -d)
    trap 'rm -rf "$tmp_dir"' EXIT
    curl -L "$ffmpeg_url" -o "$tmp_dir/ffmpeg.zip"
    unzip -q "$tmp_dir/ffmpeg.zip" -d "$tmp_dir/extracted"
    ffmpeg_path=$(find "$tmp_dir/extracted" -name 'ffmpeg.exe' -print -quit)
    [[ -n "$ffmpeg_path" ]] || fail "ffmpeg.exe not found in extracted archive"
    ffmpeg_dest="$BINARY_DIR/ffmpeg_${FFMPEG_VERSION}_windows_${ARCH}.exe"
    cp "$ffmpeg_path" "$ffmpeg_dest"
    chmod +x "$ffmpeg_dest"
    rm -rf "$tmp_dir"
    trap - EXIT

    echo "Fetching Deno ${DENO_VERSION} for windows/${ARCH}"
    case "$ARCH" in
      amd64) deno_filename="deno-x86_64-pc-windows-msvc.zip" ;;
      arm64) deno_filename="deno-aarch64-pc-windows-msvc.zip" ;;
      *) fail "Unsupported Windows arch for Deno: $ARCH" ;;
    esac
    deno_url="https://github.com/denoland/deno/releases/download/${DENO_VERSION}/${deno_filename}"
    tmp_dir=$(mktemp -d)
    trap 'rm -rf "$tmp_dir"' EXIT
    curl -L "$deno_url" -o "$tmp_dir/deno.zip"
    unzip -q "$tmp_dir/deno.zip" -d "$tmp_dir/extracted"
    deno_path=$(find "$tmp_dir/extracted" -name 'deno.exe' -print -quit)
    [[ -n "$deno_path" ]] || fail "deno.exe not found in extracted archive"
    deno_dest="$BINARY_DIR/deno_${DENO_VERSION}_windows_${ARCH}.exe"
    cp "$deno_path" "$deno_dest"
    chmod +x "$deno_dest"
    rm -rf "$tmp_dir"
    trap - EXIT
    ;;
  darwin)
    if [[ -z "$FFMPEG_VERSION" ]]; then
      FFMPEG_VERSION=$(read_const "EMBEDDED_FFMPEG_VERSION_DARWIN")
    fi
    [[ -n "$FFMPEG_VERSION" ]] || fail "Unable to determine macOS FFmpeg version"
    ensure_tools curl unzip chmod mv

    echo "Fetching yt-dlp ${YTDLP_VERSION} for darwin/${ARCH}"
    ytdlp_filename="yt-dlp_macos"
    ytdlp_url="https://github.com/yt-dlp/yt-dlp/releases/download/${YTDLP_VERSION}/${ytdlp_filename}"
    ytdlp_dest="$BINARY_DIR/yt-dlp_${YTDLP_VERSION}_darwin_${ARCH}"
    curl -L "$ytdlp_url" -o "$ytdlp_dest"
    chmod +x "$ytdlp_dest"

    echo "Fetching FFmpeg ${FFMPEG_VERSION} for darwin/${ARCH}"
    ffmpeg_url="https://evermeet.cx/ffmpeg/ffmpeg-${FFMPEG_VERSION}.zip"
    tmp_dir=$(mktemp -d)
    trap 'rm -rf "$tmp_dir"' EXIT
    curl -L "$ffmpeg_url" -o "$tmp_dir/ffmpeg.zip"
    unzip -q "$tmp_dir/ffmpeg.zip" -d "$tmp_dir/extracted"
    ffmpeg_path=$(find "$tmp_dir/extracted" -type f -name 'ffmpeg' -print -quit)
    [[ -n "$ffmpeg_path" ]] || fail "ffmpeg binary not found in extracted archive"
    ffmpeg_dest="$BINARY_DIR/ffmpeg_${FFMPEG_VERSION}_darwin_${ARCH}"
    cp "$ffmpeg_path" "$ffmpeg_dest"
    chmod +x "$ffmpeg_dest"
    rm -rf "$tmp_dir"
    trap - EXIT

    echo "Fetching Deno ${DENO_VERSION} for darwin/${ARCH}"
    case "$ARCH" in
      amd64) deno_filename="deno-x86_64-apple-darwin.zip" ;;
      arm64) deno_filename="deno-aarch64-apple-darwin.zip" ;;
      *) fail "Unsupported macOS arch for Deno: $ARCH" ;;
    esac
    deno_url="https://github.com/denoland/deno/releases/download/${DENO_VERSION}/${deno_filename}"
    tmp_dir=$(mktemp -d)
    trap 'rm -rf "$tmp_dir"' EXIT
    curl -L "$deno_url" -o "$tmp_dir/deno.zip"
    unzip -q "$tmp_dir/deno.zip" -d "$tmp_dir/extracted"
    deno_path=$(find "$tmp_dir/extracted" -type f -name 'deno' -print -quit)
    [[ -n "$deno_path" ]] || fail "deno binary not found in extracted archive"
    deno_dest="$BINARY_DIR/deno_${DENO_VERSION}_darwin_${ARCH}"
    cp "$deno_path" "$deno_dest"
    chmod +x "$deno_dest"
    rm -rf "$tmp_dir"
    trap - EXIT
    ;;
  linux)
    ensure_tools chmod
    echo "Creating placeholder embedded binaries for linux/${ARCH}"
    ytdlp_dest="$BINARY_DIR/yt-dlp_${YTDLP_VERSION}_linux_${ARCH}"
    create_placeholder "$ytdlp_dest"
    if [[ -n "$FFMPEG_VERSION" ]]; then
      ffmpeg_dest="$BINARY_DIR/ffmpeg_${FFMPEG_VERSION}_linux_${ARCH}"
    else
      ffmpeg_dest="$BINARY_DIR/ffmpeg_placeholder_linux_${ARCH}"
    fi
    create_placeholder "$ffmpeg_dest"
    deno_dest="$BINARY_DIR/deno_${DENO_VERSION}_linux_${ARCH}"
    create_placeholder "$deno_dest"
    ;;
  *)
    fail "Unsupported OS: $OS"
    ;;

esac

echo "Embedded binaries prepared in $BINARY_DIR"
