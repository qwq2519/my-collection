#!/usr/bin/env bash
#
# 下载 ffmpeg 到项目本地 persist/bin/ 目录
# 用法: ./scripts/install-ffmpeg.sh
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BIN_DIR="$PROJECT_ROOT/persist/bin"
OS="$(uname -s)"

mkdir -p "$BIN_DIR"

case "$OS" in
  Darwin)
    echo "==> 检测到 macOS，正在下载 ffmpeg ..."
    TMP_FILE="$(mktemp /tmp/ffmpeg-XXXXXX.zip)"
    curl -fSL "https://evermeet.cx/ffmpeg/getrelease/zip" -o "$TMP_FILE"
    unzip -o "$TMP_FILE" -d "$BIN_DIR"
    rm -f "$TMP_FILE"
    chmod +x "$BIN_DIR/ffmpeg"
    ;;
  Linux)
    echo "==> 检测到 Linux，正在下载 ffmpeg ..."
    TMP_FILE="$(mktemp /tmp/ffmpeg-XXXXXX.tar.xz)"
    curl -fSL "https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-amd64-static.tar.xz" -o "$TMP_FILE"
    tar -xf "$TMP_FILE" --strip-components=1 -C "$BIN_DIR" --wildcards "*/ffmpeg"
    rm -f "$TMP_FILE"
    chmod +x "$BIN_DIR/ffmpeg"
    ;;
  *)
    echo "错误: 不支持的操作系统 $OS，请使用 install-ffmpeg.bat (Windows)" >&2
    exit 1
    ;;
esac

# 验证安装
if "$BIN_DIR/ffmpeg" -version >/dev/null 2>&1; then
  VERSION=$("$BIN_DIR/ffmpeg" -version | head -1 | sed 's/ffmpeg version \([^ ]*\).*/\1/')
  echo "==> 安装成功! ffmpeg $VERSION"
  echo "    路径: $BIN_DIR/ffmpeg"
  echo "    请在应用设置页点击「重新检测」"
else
  echo "错误: 安装后验证失败" >&2
  exit 1
fi
