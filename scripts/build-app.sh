#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")/.."
export PATH="/opt/homebrew/bin:/usr/local/go/bin:$PATH"
arch="${ARCH:-$(go env GOARCH)}"
out="dist/$arch/thief.app"
mkdir -p "$out/Contents/MacOS" "$out/Contents/Resources"
CGO_ENABLED=1 GOOS=darwin GOARCH="$arch" go build -trimpath -o "$out/Contents/MacOS/thief" ./cmd/thief
cp assets/Info.plist "$out/Contents/Info.plist"
codesign --force --sign - "$out"
printf '%s\n' "$PWD/$out"
