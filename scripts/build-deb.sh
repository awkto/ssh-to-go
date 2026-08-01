#!/bin/bash
# Build the sshtogo .deb (installs /usr/bin/stogo).
# Usage: build-deb.sh <version> <arch> <path-to-stogo-binary> [outdir]
#   version:  1.25.0 (no leading v)
#   arch:     amd64 | arm64
set -euo pipefail

VERSION="${1:?version required}"
ARCH="${2:?arch required}"
BINARY="${3:?binary path required}"
OUTDIR="${4:-.}"

VERSION="${VERSION#v}"
STAGE="$(mktemp -d)"
trap 'rm -rf "$STAGE"' EXIT

PKGDIR="$STAGE/sshtogo_${VERSION}_${ARCH}"
mkdir -p "$PKGDIR/DEBIAN" "$PKGDIR/usr/bin"

install -m 0755 "$BINARY" "$PKGDIR/usr/bin/stogo"

# Bash completion (same file the binary embeds for `stogo completion bash`).
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
mkdir -p "$PKGDIR/usr/share/bash-completion/completions"
install -m 0644 "$SCRIPT_DIR/../cmd/stogo/completion.bash" \
  "$PKGDIR/usr/share/bash-completion/completions/stogo"

cat > "$PKGDIR/DEBIAN/control" <<EOF
Package: sshtogo
Version: ${VERSION}
Section: utils
Priority: optional
Architecture: ${ARCH}
Depends: openssh-client
Maintainer: awkto <me@awkto.dev>
Homepage: https://github.com/awkto/ssh-to-go
Description: Terminal client for ssh-to-go
 stogo lists, attaches to, offloads and kills tmux sessions managed by an
 ssh-to-go server. Attaching hands off to your local ssh client and the
 target host's tmux, exactly like the dashboard's native-terminal handoff.
EOF

dpkg-deb --build --root-owner-group "$PKGDIR" "$OUTDIR/sshtogo_${VERSION}_${ARCH}.deb"
