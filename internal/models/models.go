package models

import "time"

type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

// Inbound represents a single proxy inbound (VMess/VLESS/Trojan/Shadowsocks).
type Inbound struct {
	ID         int64     `json:"id"`
	Tag        string    `json:"tag"`
	Protocol   string    `json:"protocol"` // vmess, vless, trojan, shadowsocks, socks, http
	Listen     string    `json:"listen"`
	Port       int       `json:"port"`
	Network    string    `json:"network"`  // tcp, ws, grpc
	Security   string    `json:"security"` // none, tls, reality
	Remark     string    `json:"remark"`
	Enabled    bool      `json:"enabled"`
	UUID       string    `json:"uuid"`     // for vmess/vless
	Username   string    `json:"username"` // for socks/http auth
	Password   string    `json:"password"` // for trojan/ss/socks/http
	Method     string    `json:"method"`   // for ss (e.g. aes-128-gcm)
	WSPath     string    `json:"ws_path"`
	SNI        string    `json:"sni"`
	Flow       string    `json:"flow"`
	UpBytes    int64     `json:"up_bytes"`
	DownBytes  int64     `json:"down_bytes"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type Settings struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
