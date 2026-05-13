package xray

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/Hujaha/vpn-proxy/internal/models"
)

// ShareLink builds a client share link (vless://, vmess://, trojan://, ss://)
// for the given inbound and host. host is the publicly reachable address
// (IP or domain) that clients should connect to.
func ShareLink(in *models.Inbound, host string) string {
	if host == "" {
		host = in.Listen
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "your-server"
	}
	name := in.Remark
	if name == "" {
		name = in.Tag
	}

	switch in.Protocol {
	case "vless":
		q := url.Values{}
		q.Set("type", orStr(in.Network, "tcp"))
		q.Set("security", orStr(in.Security, "none"))
		if in.SNI != "" {
			q.Set("sni", in.SNI)
		}
		if in.Flow != "" {
			q.Set("flow", in.Flow)
		}
		if in.Network == "ws" {
			q.Set("path", orStr(in.WSPath, "/"))
			if in.SNI != "" {
				q.Set("host", in.SNI)
			}
		}
		return fmt.Sprintf("vless://%s@%s:%d?%s#%s",
			in.UUID, host, in.Port, q.Encode(), url.PathEscape(name))

	case "vmess":
		obj := map[string]any{
			"v":    "2",
			"ps":   name,
			"add":  host,
			"port": strconv.Itoa(in.Port),
			"id":   in.UUID,
			"aid":  "0",
			"net":  orStr(in.Network, "tcp"),
			"type": "none",
			"host": in.SNI,
			"path": orStr(in.WSPath, ""),
			"tls":  ifStr(in.Security == "tls", "tls", ""),
			"sni":  in.SNI,
		}
		b, _ := json.Marshal(obj)
		return "vmess://" + base64.StdEncoding.EncodeToString(b)

	case "trojan":
		q := url.Values{}
		q.Set("type", orStr(in.Network, "tcp"))
		q.Set("security", orStr(in.Security, "tls"))
		if in.SNI != "" {
			q.Set("sni", in.SNI)
		}
		if in.Network == "ws" {
			q.Set("path", orStr(in.WSPath, "/"))
		}
		return fmt.Sprintf("trojan://%s@%s:%d?%s#%s",
			url.QueryEscape(in.Password), host, in.Port, q.Encode(), url.PathEscape(name))

	case "shadowsocks":
		userInfo := base64.RawURLEncoding.EncodeToString([]byte(in.Method + ":" + in.Password))
		return fmt.Sprintf("ss://%s@%s:%d#%s",
			userInfo, host, in.Port, url.PathEscape(name))

	case "socks":
		auth := ""
		if in.Username != "" {
			auth = url.QueryEscape(in.Username) + ":" + url.QueryEscape(in.Password) + "@"
		}
		return fmt.Sprintf("socks5://%s%s:%d#%s", auth, host, in.Port, url.PathEscape(name))

	case "http":
		auth := ""
		if in.Username != "" {
			auth = url.QueryEscape(in.Username) + ":" + url.QueryEscape(in.Password) + "@"
		}
		return fmt.Sprintf("http://%s%s:%d#%s", auth, host, in.Port, url.PathEscape(name))
	}
	return ""
}

func orStr(s, d string) string {
	if strings.TrimSpace(s) == "" {
		return d
	}
	return s
}

func ifStr(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}
