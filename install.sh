#!/usr/bin/env bash
# vpn-proxy installer.
# Usage:
#   curl -sSL https://raw.githubusercontent.com/Hujaha/vpn-proxy/main/install.sh | sudo bash
set -euo pipefail

REPO="${VPN_PROXY_REPO:-Hujaha/vpn-proxy}"
INSTALL_DIR="${VPN_PROXY_DIR:-/opt/vpn-proxy}"
SERVICE="vpn-proxy"
ADDR="${VPN_PROXY_ADDR:-:2053}"

need_root() {
  if [[ $EUID -ne 0 ]]; then
    echo "Please run as root (sudo bash install.sh)." >&2
    exit 1
  fi
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo "amd64" ;;
    aarch64|arm64) echo "arm64" ;;
    armv7l) echo "armv7" ;;
    *) echo "unsupported"; exit 1 ;;
  esac
}

install_deps() {
  if command -v apt-get >/dev/null 2>&1; then
    apt-get update -qq
    apt-get install -y --no-install-recommends git ca-certificates curl ufw >/dev/null
  elif command -v dnf >/dev/null 2>&1; then
    dnf install -y git ca-certificates curl >/dev/null
  fi
}

install_go() {
  if command -v go >/dev/null 2>&1; then return; fi
  echo "[*] Installing Go..."
  local arch tarball
  arch=$(detect_arch)
  tarball="go1.22.5.linux-${arch}.tar.gz"
  curl -fsSL "https://go.dev/dl/${tarball}" -o /tmp/$tarball
  rm -rf /usr/local/go
  tar -C /usr/local -xzf /tmp/$tarball
  ln -sf /usr/local/go/bin/go /usr/local/bin/go
  rm -f /tmp/$tarball
}

build() {
  echo "[*] Cloning ${REPO}..."
  rm -rf "$INSTALL_DIR"
  git clone "https://github.com/${REPO}.git" "$INSTALL_DIR"
  echo "[*] Building..."
  (cd "$INSTALL_DIR" && go build -o vpn-proxy .)
}

write_service() {
  cat >/etc/systemd/system/${SERVICE}.service <<EOF
[Unit]
Description=vpn-proxy panel
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=${INSTALL_DIR}
Environment=VPN_PROXY_ADDR=${ADDR}
ExecStart=${INSTALL_DIR}/vpn-proxy
Restart=on-failure
RestartSec=3

[Install]
WantedBy=multi-user.target
EOF
  systemctl daemon-reload
  systemctl enable --now ${SERVICE}
}

open_panel_port() {
  if command -v ufw >/dev/null 2>&1; then
    local port="${ADDR##*:}"
    [[ -z "$port" ]] && port=2053
    ufw allow "${port}/tcp" >/dev/null 2>&1 || true
  fi
}

main() {
  need_root
  install_deps
  install_go
  build
  write_service
  open_panel_port
  echo
  echo "[+] vpn-proxy installed."
  echo "    Panel: http://$(hostname -I | awk '{print $1}')${ADDR}"
  echo "    Default credentials: admin / admin (change immediately)"
  echo "    Logs:  journalctl -u ${SERVICE} -f"
}

main "$@"
