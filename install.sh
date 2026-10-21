#!/bin/sh
set -e

INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
TARGET="${INSTALL_DIR}/tinystoredb-cli"

SUDO=""
if [ ! -w "$INSTALL_DIR" ]; then
    SUDO="sudo"
fi

echo "🚀 Installing tinystoredb-cli to $INSTALL_DIR..."

if [ -f "./cmd/tinystoredb-cli/main.go" ] && command -v go >/dev/null 2>&1; then
    echo "📦 Building from local source..."
    go build -o /tmp/tinystoredb-cli ./cmd/tinystoredb-cli/main.go
    $SUDO mv /tmp/tinystoredb-cli "$TARGET"
elif command -v go >/dev/null 2>&1; then
    echo "📦 Building with Go from GitHub..."
    TMP_DIR=$(mktemp -d)
    git clone --depth 1 https://github.com/raghavgh/TinyStoreDB.git "$TMP_DIR" 2>/dev/null
    (cd "$TMP_DIR" && go build -o tinystoredb-cli ./cmd/tinystoredb-cli/main.go)
    $SUDO mv "$TMP_DIR/tinystoredb-cli" "$TARGET"
    rm -rf "$TMP_DIR"
else
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    case "$ARCH" in
        x86_64) ARCH="amd64" ;;
        arm64|aarch64) ARCH="arm64" ;;
    esac
    URL="https://github.com/raghavgh/TinyStoreDB/releases/latest/download/tinystoredb-cli-${OS}-${ARCH}"
    echo "🌐 Downloading pre-built binary (${OS}/${ARCH})..."
    $SUDO curl -fsSL "$URL" -o "$TARGET"
    $SUDO chmod +x "$TARGET"
fi

$SUDO chmod +x "$TARGET"

echo "✅ tinystoredb-cli installed successfully!"
echo "👉 Run 'tinystoredb-cli' to connect to your TinyStoreDB server."
