package main

import (
	"embed"
	"flag"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/Hujaha/vpn-proxy/internal/auth"
	"github.com/Hujaha/vpn-proxy/internal/db"
	"github.com/Hujaha/vpn-proxy/internal/handlers"
	"github.com/gin-gonic/gin"
)

//go:embed web/templates/*.html
var tmplFS embed.FS

//go:embed web/static
var staticFS embed.FS

func main() {
	addr := flag.String("addr", envOr("VPN_PROXY_ADDR", ":2053"), "listen address")
	dbPath := flag.String("db", envOr("VPN_PROXY_DB", "./data/vpn-proxy.db"), "sqlite db path")
	flag.Parse()

	if err := db.Init(*dbPath); err != nil {
		log.Fatalf("db init: %v", err)
	}
	defer db.MustClose()

	if err := ensureAdmin(); err != nil {
		log.Fatalf("admin bootstrap: %v", err)
	}

	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	tmpl := template.Must(template.New("").ParseFS(tmplFS, "web/templates/*.html"))
	r.SetHTMLTemplate(tmpl)

	staticSub, err := fs.Sub(staticFS, "web/static")
	if err != nil {
		log.Fatalf("static fs: %v", err)
	}
	r.StaticFS("/static", http.FS(staticSub))

	r.GET("/", func(c *gin.Context) { c.Redirect(http.StatusFound, "/dashboard") })
	r.GET("/login", handlers.LoginPage)
	r.POST("/api/login", handlers.Login)

	authed := r.Group("/")
	authed.Use(auth.Middleware())
	{
		authed.GET("/dashboard", handlers.DashboardPage)
		authed.GET("/inbounds", handlers.InboundsPage)
		authed.GET("/settings", handlers.SettingsPage)
		authed.GET("/logout", handlers.Logout)

		api := authed.Group("/api")
		api.GET("/stats", handlers.Stats)
		api.GET("/inbounds", handlers.ListInbounds)
		api.POST("/inbounds", handlers.CreateInbound)
		api.PUT("/inbounds/:id", handlers.UpdateInbound)
		api.DELETE("/inbounds/:id", handlers.DeleteInbound)
		api.POST("/xray/restart", handlers.RestartXray)
		api.GET("/xray/config", handlers.XrayConfig)
		api.POST("/account/password", handlers.ChangePassword)
	}

	log.Printf("vpn-proxy listening on %s", *addr)
	if err := r.Run(*addr); err != nil {
		log.Fatal(err)
	}
}

func ensureAdmin() error {
	n, err := db.CountUsers()
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	user := envOr("VPN_PROXY_ADMIN", "admin")
	pw := envOr("VPN_PROXY_PASSWORD", "admin")
	hash, err := auth.HashPassword(pw)
	if err != nil {
		return err
	}
	if err := db.CreateUser(user, hash); err != nil {
		return err
	}
	log.Printf("[init] created admin user %q with password %q -- change it immediately!", user, pw)
	return nil
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
