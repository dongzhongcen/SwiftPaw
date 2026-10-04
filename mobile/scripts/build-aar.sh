#!/usr/bin/env bash
# 用 gomobile 把 Go 内核（mobile/gocore → internal/core）编译成 Android 用的 .aar，
# 输出到 mobile/android/app/libs/swiftpaw-core.aar。
#
# 需要：Go（版本见 go.mod）、Android SDK 和 NDK（设置 ANDROID_HOME 和 ANDROID_NDK_HOME），
# 以及 gomobile（没有时脚本会按 go.mod 里 golang.org/x/mobile 的版本自动安装）。
# 用法（在仓库根目录或任意位置）：mobile/scripts/build-aar.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

# 支持的 CPU：64 位 ARM（绝大多数手机）、32 位 ARM（老手机）、x86_64（模拟器、少数平板）
TARGETS="${SWIFTPAW_ANDROID_TARGETS:-android/arm64,android/arm,android/amd64}"
OUT="mobile/android/app/libs/swiftpaw-core.aar"

: "${ANDROID_HOME:?请设置 ANDROID_HOME（Android SDK 的位置）}"
if [ -z "${ANDROID_NDK_HOME:-}" ]; then
  # 没指定 NDK 时用 SDK 里版本最新的那个
  ANDROID_NDK_HOME="$(ls -d "$ANDROID_HOME"/ndk/* 2>/dev/null | sort -V | tail -n 1 || true)"
  export ANDROID_NDK_HOME
fi
[ -d "${ANDROID_NDK_HOME:-}" ] || { echo "找不到 Android NDK，请设置 ANDROID_NDK_HOME" >&2; exit 1; }

# gomobile 和 gobind 的版本要和 go.mod 里的 golang.org/x/mobile 一致
MOBILE_VERSION="$(go list -m -f '{{.Version}}' golang.org/x/mobile)"
GOBIN="$(go env GOPATH)/bin"
export PATH="$GOBIN:$PATH"
if ! command -v gomobile >/dev/null || ! go version -m "$(command -v gomobile)" | grep -q "$MOBILE_VERSION"; then
  echo "安装 gomobile $MOBILE_VERSION"
  go install "golang.org/x/mobile/cmd/gomobile@$MOBILE_VERSION" "golang.org/x/mobile/cmd/gobind@$MOBILE_VERSION"
fi

mkdir -p "$(dirname "$OUT")"
echo "编译 Go 内核：$TARGETS（NDK：$ANDROID_NDK_HOME）"
# -androidapi 26：最低支持 Android 8.0；-s -w 去掉调试信息，.so 小一些
gomobile bind \
  -target="$TARGETS" \
  -androidapi 26 \
  -javapkg com.dongzhongcen.swiftpaw.core \
  -trimpath \
  -ldflags "-s -w" \
  -o "$OUT" \
  ./mobile/gocore
rm -f "${OUT%.aar}-sources.jar"
ls -l "$OUT"
