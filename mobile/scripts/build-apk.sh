#!/usr/bin/env bash
# 本地完整编译 Android 安装包：前端 → Go 内核 → Capacitor 同步 → Gradle 打包。
#
# 用法：mobile/scripts/build-apk.sh [debug|release]（默认 debug）
# 编 release 版需要签名，设置这些环境变量：ANDROID_KEYSTORE_FILE（.jks 文件的路径）、
# ANDROID_KEYSTORE_PASSWORD、ANDROID_KEY_ALIAS、ANDROID_KEY_PASSWORD。
# 需要 Node.js 22 以上、JDK 21、Go、Android SDK 和 NDK。
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
TYPE="${1:-debug}"
case "$TYPE" in
  debug) TASK=assembleDebug ;;
  release) TASK=assembleRelease ;;
  *) echo "用法：$0 [debug|release]" >&2; exit 1 ;;
esac

echo "== 前端"
(cd "$ROOT/frontend" && npm ci && npm run build)

echo "== Go 内核"
"$ROOT/mobile/scripts/build-aar.sh"

echo "== Capacitor"
(cd "$ROOT/mobile" && npm ci && npx cap sync android)

echo "== Gradle"
(cd "$ROOT/mobile/android" && ./gradlew --no-daemon "$TASK")

ls -l "$ROOT/mobile/android/app/build/outputs/apk/$TYPE/"
