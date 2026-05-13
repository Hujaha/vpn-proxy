# vpn-proxy

A lightweight 3x-ui–style proxy panel written in Go, with a UI inspired by [claude.ai](https://claude.ai) — warm neutrals, soft contrast, light & dark themes.

> Educational / personal-use clone. Not affiliated with the original [3x-ui](https://github.com/MHSanaei/3x-ui) project.

## Features

- 🔐 JWT-cookie login (single admin, bcrypt-hashed)
- 🧭 Dashboard with live CPU / RAM / disk / load / network / uptime
- ⚡ Inbound CRUD: **VMess**, **VLESS**, **Trojan**, **Shadowsocks**
  - Auto-generated UUIDs / passwords
  - TCP / WS / gRPC, TLS / Reality flags
- 🧦 Plain **SOCKS5** and **HTTP CONNECT** proxies (with optional user/password auth) — drop-in replacement for dante / squid
- 🔥 **UFW integration** — opens the inbound port automatically on create, removes it on delete
- 🛠 **Xray config generator** + one-click `systemctl restart xray`
- 🌗 Persistent **dark / light theme toggle** (claude.ai-inspired palette)
- 📦 Single static binary, embedded HTML/CSS/JS, SQLite (no CGO)
- 🔗 **Client share links** (`vless://`, `vmess://`, `ss://`, `trojan://`) + **QR codes**
- 🔑 One-click **Reality X25519 keypair** generator

## Quick start

### One-line installer (Linux + systemd)

```bash
curl -sSL https://raw.githubusercontent.com/Hujaha/vpn-proxy/main/install.sh | sudo bash
```

This installs Go (if missing), clones the repo to `/opt/vpn-proxy`, builds the binary, registers a `vpn-proxy.service`, and opens port 2053 in UFW.

### Docker

```bash
docker compose up -d
```

### Manual build

```bash
git clone https://github.com/Hujaha/vpn-proxy.git
cd vpn-proxy
go build -o vpn-proxy .
./vpn-proxy
```

The first launch creates an admin user (defaults: `admin` / `admin`). Open <http://server-ip:2053/login> and **change the password immediately** in *Settings*.

### Environment variables

| Variable | Default | Description |
| --- | --- | --- |
| `VPN_PROXY_ADDR` | `:2053` | Listen address |
| `VPN_PROXY_DB` | `./data/vpn-proxy.db` | SQLite database path |
| `VPN_PROXY_ADMIN` | `admin` | Initial admin username |
| `VPN_PROXY_PASSWORD` | `admin` | Initial admin password |
| `VPN_PROXY_SECRET` | random | JWT signing secret (auto-generated on first run) |
| `VPN_PROXY_XRAY_CONFIG` | `/usr/local/etc/xray/config.json` | Where to write the generated Xray config |

### Systemd unit

```ini
# /etc/systemd/system/vpn-proxy.service
[Unit]
Description=vpn-proxy panel
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/vpn-proxy
Environment=VPN_PROXY_ADDR=:2053
ExecStart=/opt/vpn-proxy/vpn-proxy
Restart=always

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now vpn-proxy
```

## UFW

If `ufw` is installed, the panel will:

- run `ufw allow <port>/tcp` (or `tcp+udp` for Shadowsocks) when an inbound is created with the **Open port in UFW** switch on,
- run `ufw delete allow <port>/...` when the inbound is deleted or its port is changed.

If `ufw` is not installed, these calls are no-ops, so the panel is safe to use without it.

## Xray

The panel doesn't bundle Xray. Install it separately (e.g. with the official installer) and the *Restart Xray* button will run `systemctl restart xray` after writing the generated config to `VPN_PROXY_XRAY_CONFIG`.

## Development

```bash
go run .
# panel on :2053
```

Tech stack: Go 1.22+, [Gin](https://github.com/gin-gonic/gin), [modernc.org/sqlite](https://gitlab.com/cznic/sqlite) (pure Go), [gopsutil](https://github.com/shirou/gopsutil), vanilla HTML/CSS/JS.

## Screenshots

Light and dark themes follow the same palette as claude.ai — warm off-white `#faf9f6` and warm dark `#1f1e1b`, with the signature orange-brown accent.

## License

[MIT](LICENSE) © 2026 Hujaha
