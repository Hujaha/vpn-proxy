package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/Hujaha/vpn-proxy/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const sessionCookie = "vpn_proxy_session"

func secret() []byte {
	if v := os.Getenv("VPN_PROXY_SECRET"); v != "" {
		return []byte(v)
	}
	v := db.GetSetting("jwt_secret", "")
	if v == "" {
		v = randHex(32)
		_ = db.SetSetting("jwt_secret", v)
	}
	return []byte(v)
}

func HashPassword(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), 12)
	return string(b), err
}

func CheckPassword(hash, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

type Claims struct {
	Username string `json:"u"`
	jwt.RegisteredClaims
}

func IssueToken(username string) (string, error) {
	c := Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return t.SignedString(secret())
}

func ParseToken(s string) (*Claims, error) {
	c := &Claims{}
	tok, err := jwt.ParseWithClaims(s, c, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return secret(), nil
	})
	if err != nil || !tok.Valid {
		return nil, errors.New("invalid token")
	}
	return c, nil
}

func SetSessionCookie(c *gin.Context, token string) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(sessionCookie, token, 7*24*3600, "/", "", false, true)
}

func ClearSessionCookie(c *gin.Context) {
	c.SetCookie(sessionCookie, "", -1, "/", "", false, true)
}

func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tok, err := c.Cookie(sessionCookie)
		if err != nil || tok == "" {
			if isAPI(c) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
				return
			}
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}
		claims, err := ParseToken(tok)
		if err != nil {
			ClearSessionCookie(c)
			if isAPI(c) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
				return
			}
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}
		c.Set("user", claims.Username)
		c.Next()
	}
}

func isAPI(c *gin.Context) bool {
	return len(c.Request.URL.Path) >= 5 && c.Request.URL.Path[:5] == "/api/"
}

func randHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return time.Now().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(b)
}
