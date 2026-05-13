package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"

	"github.com/Hujaha/vpn-proxy/internal/auth"
	"github.com/Hujaha/vpn-proxy/internal/db"
	"github.com/Hujaha/vpn-proxy/internal/models"
	"github.com/Hujaha/vpn-proxy/internal/system"
	"github.com/Hujaha/vpn-proxy/internal/ufw"
	"github.com/Hujaha/vpn-proxy/internal/xray"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	qrcode "github.com/skip2/go-qrcode"
)

// Page renderers ---------------------------------------------------------

func LoginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", gin.H{"title": "Login"})
}

func DashboardPage(c *gin.Context) {
	c.HTML(http.StatusOK, "dashboard.html", gin.H{"title": "Dashboard", "active": "dashboard"})
}

func InboundsPage(c *gin.Context) {
	c.HTML(http.StatusOK, "inbounds.html", gin.H{"title": "Inbounds", "active": "inbounds"})
}

func SettingsPage(c *gin.Context) {
	c.HTML(http.StatusOK, "settings.html", gin.H{"title": "Settings", "active": "settings"})
}

// Auth -------------------------------------------------------------------

type loginReq struct {
	Username string `json:"username" form:"username"`
	Password string `json:"password" form:"password"`
}

func Login(c *gin.Context) {
	var r loginReq
	if err := c.ShouldBind(&r); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		return
	}
	u, err := db.GetUser(r.Username)
	if err != nil || !auth.CheckPassword(u.PasswordHash, r.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	tok, err := auth.IssueToken(u.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	auth.SetSessionCookie(c, tok)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func Logout(c *gin.Context) {
	auth.ClearSessionCookie(c)
	c.Redirect(http.StatusFound, "/login")
}

type changePwReq struct {
	Old string `json:"old"`
	New string `json:"new"`
}

func ChangePassword(c *gin.Context) {
	var r changePwReq
	if err := c.ShouldBindJSON(&r); err != nil || r.New == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		return
	}
	user := c.GetString("user")
	u, err := db.GetUser(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !auth.CheckPassword(u.PasswordHash, r.Old) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "wrong password"})
		return
	}
	hash, err := auth.HashPassword(r.New)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := db.UpdateUserPassword(user, hash); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// Inbounds ---------------------------------------------------------------

func ListInbounds(c *gin.Context) {
	items, err := db.ListInbounds()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

type inboundReq struct {
	Tag      string `json:"tag"`
	Protocol string `json:"protocol"`
	Listen   string `json:"listen"`
	Port     int    `json:"port"`
	Network  string `json:"network"`
	Security string `json:"security"`
	Remark   string `json:"remark"`
	Enabled  bool   `json:"enabled"`
	UUID     string `json:"uuid"`
	Password string `json:"password"`
	Method   string `json:"method"`
	WSPath   string `json:"ws_path"`
	SNI      string `json:"sni"`
	Flow     string `json:"flow"`
	UFW      bool   `json:"ufw"`
}

func (r *inboundReq) toModel() *models.Inbound {
	return &models.Inbound{
		Tag: r.Tag, Protocol: r.Protocol, Listen: defaultStr(r.Listen, "0.0.0.0"),
		Port: r.Port, Network: defaultStr(r.Network, "tcp"),
		Security: defaultStr(r.Security, "none"), Remark: r.Remark, Enabled: r.Enabled,
		UUID: r.UUID, Password: r.Password, Method: r.Method, WSPath: r.WSPath,
		SNI: r.SNI, Flow: r.Flow,
	}
}

func CreateInbound(c *gin.Context) {
	var r inboundReq
	if err := c.ShouldBindJSON(&r); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if r.Port <= 0 || r.Port > 65535 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid port"})
		return
	}
	if !validProtocol(r.Protocol) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid protocol"})
		return
	}
	in := r.toModel()
	autoFill(in)

	id, err := db.CreateInbound(in)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	in.ID = id
	if r.UFW {
		_ = ufw.AllowPort(in.Port, inboundProto(in))
	}
	_ = xray.WriteConfig()
	c.JSON(http.StatusOK, gin.H{"item": in})
}

func UpdateInbound(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad id"})
		return
	}
	var r inboundReq
	if err := c.ShouldBindJSON(&r); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	existing, err := db.GetInbound(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	in := r.toModel()
	in.ID = id
	autoFill(in)
	if err := db.UpdateInbound(in); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if existing.Port != in.Port {
		_ = ufw.DenyPort(existing.Port, inboundProto(existing))
		if r.UFW {
			_ = ufw.AllowPort(in.Port, inboundProto(in))
		}
	}
	_ = xray.WriteConfig()
	c.JSON(http.StatusOK, gin.H{"item": in})
}

func DeleteInbound(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad id"})
		return
	}
	in, err := db.GetInbound(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err := db.DeleteInbound(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = ufw.DenyPort(in.Port, inboundProto(in))
	_ = xray.WriteConfig()
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// System / actions -------------------------------------------------------

func Stats(c *gin.Context) {
	stats := system.Collect()
	inbounds, _ := db.ListInbounds()
	var enabled int
	for _, i := range inbounds {
		if i.Enabled {
			enabled++
		}
	}
	ufwStatus, _ := ufw.Status()
	c.JSON(http.StatusOK, gin.H{
		"system":    stats,
		"inbounds":  len(inbounds),
		"enabled":   enabled,
		"ufw":       ufwStatus,
		"ufw_avail": ufw.Available(),
		"xray":      xray.GetStatus(),
	})
}

func XrayStatus(c *gin.Context) {
	c.JSON(http.StatusOK, xray.GetStatus())
}

type xrayInstallReq struct {
	Force bool `json:"force"`
}

func InstallXray(c *gin.Context) {
	var r xrayInstallReq
	_ = c.ShouldBindJSON(&r)
	msg, err := xray.Install(r.Force)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "message": msg, "status": xray.GetStatus()})
}

func XrayService(c *gin.Context) {
	action := c.Param("action")
	if action == "start" || action == "restart" {
		if err := xray.WriteConfig(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	if err := xray.ServiceCommand(action); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "status": xray.GetStatus()})
}

func RestartXray(c *gin.Context) {
	if err := xray.WriteConfig(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := xray.Restart(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func XrayConfig(c *gin.Context) {
	inbounds, err := db.ListInbounds()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, xray.BuildConfig(inbounds))
}

func InboundShare(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad id"})
		return
	}
	in, err := db.GetInbound(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	host := db.GetSetting("server_host", "")
	if host == "" {
		host = c.Request.Host
		if i := strings.LastIndex(host, ":"); i > 0 {
			host = host[:i]
		}
	}
	link := xray.ShareLink(in, host)
	c.JSON(http.StatusOK, gin.H{"link": link, "host": host})
}

func InboundQR(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	in, err := db.GetInbound(id)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	host := db.GetSetting("server_host", "")
	if host == "" {
		host = c.Request.Host
		if i := strings.LastIndex(host, ":"); i > 0 {
			host = host[:i]
		}
	}
	link := xray.ShareLink(in, host)
	if link == "" {
		c.Status(http.StatusNotFound)
		return
	}
	png, err := qrcode.Encode(link, qrcode.Medium, 320)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	c.Data(http.StatusOK, "image/png", png)
}

func GenerateRealityKeys(c *gin.Context) {
	keys, err := xray.GenerateRealityKeys()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, keys)
}

type hostReq struct {
	Host string `json:"host"`
}

func SetHost(c *gin.Context) {
	var r hostReq
	if err := c.ShouldBindJSON(&r); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		return
	}
	if err := db.SetSetting("server_host", strings.TrimSpace(r.Host)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "host": r.Host})
}

func GetHost(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"host": db.GetSetting("server_host", "")})
}

// Helpers ----------------------------------------------------------------

func validProtocol(p string) bool {
	switch p {
	case "vmess", "vless", "trojan", "shadowsocks":
		return true
	}
	return false
}

func defaultStr(s, d string) string {
	if strings.TrimSpace(s) == "" {
		return d
	}
	return s
}

func inboundProto(in *models.Inbound) string {
	if in.Protocol == "shadowsocks" {
		return "" // both tcp+udp
	}
	return "tcp"
}

func autoFill(in *models.Inbound) {
	switch in.Protocol {
	case "vmess", "vless":
		if in.UUID == "" {
			in.UUID = uuid.NewString()
		}
	case "trojan":
		if in.Password == "" {
			in.Password = randomToken(16)
		}
	case "shadowsocks":
		if in.Password == "" {
			in.Password = randomToken(16)
		}
		if in.Method == "" {
			in.Method = "aes-128-gcm"
		}
	}
	if in.Tag == "" {
		in.Tag = in.Protocol + "-" + strconv.Itoa(in.Port)
	}
}

func randomToken(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
