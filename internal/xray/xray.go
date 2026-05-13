package xray

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Hujaha/vpn-proxy/internal/db"
	"github.com/Hujaha/vpn-proxy/internal/models"
)

// ConfigPath is where the generated Xray config is written.
var ConfigPath = "/usr/local/etc/xray/config.json"

func init() {
	if v := os.Getenv("VPN_PROXY_XRAY_CONFIG"); v != "" {
		ConfigPath = v
	}
}

// BuildConfig assembles a minimal Xray configuration from enabled inbounds.
func BuildConfig(inbounds []models.Inbound) map[string]any {
	ins := []map[string]any{}
	for _, in := range inbounds {
		if !in.Enabled {
			continue
		}
		ins = append(ins, buildInbound(in))
	}
	return map[string]any{
		"log": map[string]any{
			"loglevel": "warning",
		},
		"inbounds": ins,
		"outbounds": []map[string]any{
			{"protocol": "freedom", "tag": "direct"},
			{"protocol": "blackhole", "tag": "block"},
		},
	}
}

func buildInbound(in models.Inbound) map[string]any {
	listen := in.Listen
	if listen == "" {
		listen = "0.0.0.0"
	}
	tag := in.Tag
	if tag == "" {
		tag = fmt.Sprintf("inbound-%d", in.ID)
	}
	settings := map[string]any{}
	switch in.Protocol {
	case "vmess":
		settings["clients"] = []map[string]any{{"id": in.UUID, "alterId": 0}}
	case "vless":
		client := map[string]any{"id": in.UUID, "flow": in.Flow}
		settings["clients"] = []map[string]any{client}
		settings["decryption"] = "none"
	case "trojan":
		settings["clients"] = []map[string]any{{"password": in.Password}}
	case "shadowsocks":
		method := in.Method
		if method == "" {
			method = "aes-128-gcm"
		}
		settings["method"] = method
		settings["password"] = in.Password
		settings["network"] = "tcp,udp"
	case "socks":
		settings["udp"] = true
		settings["ip"] = "127.0.0.1"
		if in.Username != "" {
			settings["auth"] = "password"
			settings["accounts"] = []map[string]any{{"user": in.Username, "pass": in.Password}}
		} else {
			settings["auth"] = "noauth"
		}
	case "http":
		if in.Username != "" {
			settings["accounts"] = []map[string]any{{"user": in.Username, "pass": in.Password}}
		}
		settings["allowTransparent"] = false
	}

	stream := map[string]any{
		"network":  in.Network,
		"security": in.Security,
	}
	if in.Network == "ws" {
		stream["wsSettings"] = map[string]any{
			"path": orDefault(in.WSPath, "/"),
		}
	}
	if in.Security == "tls" && in.SNI != "" {
		stream["tlsSettings"] = map[string]any{
			"serverName": in.SNI,
		}
	}

	return map[string]any{
		"tag":            tag,
		"listen":         listen,
		"port":           in.Port,
		"protocol":       in.Protocol,
		"settings":       settings,
		"streamSettings": stream,
	}
}

func orDefault(s, d string) string {
	if strings.TrimSpace(s) == "" {
		return d
	}
	return s
}

// WriteConfig serializes the active inbound list to disk.
func WriteConfig() error {
	inbounds, err := db.ListInbounds()
	if err != nil {
		return err
	}
	cfg := BuildConfig(inbounds)
	if err := os.MkdirAll(filepath.Dir(ConfigPath), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ConfigPath, b, 0o644)
}

// Restart attempts to restart the system xray service. Best-effort.
func Restart() error {
	if _, err := exec.LookPath("systemctl"); err != nil {
		return nil
	}
	out, err := exec.Command("systemctl", "restart", "xray").CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl restart xray: %v: %s", err, string(out))
	}
	return nil
}
