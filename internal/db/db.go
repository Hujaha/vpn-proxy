package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/Hujaha/vpn-proxy/internal/models"
	_ "modernc.org/sqlite"
)

var DB *sql.DB

func Init(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	d, err := sql.Open("sqlite", path)
	if err != nil {
		return err
	}
	d.SetMaxOpenConns(1)
	DB = d
	return migrate()
}

func migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			created_at DATETIME NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS inbounds (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			tag TEXT NOT NULL,
			protocol TEXT NOT NULL,
			listen TEXT NOT NULL DEFAULT '0.0.0.0',
			port INTEGER NOT NULL,
			network TEXT NOT NULL DEFAULT 'tcp',
			security TEXT NOT NULL DEFAULT 'none',
			remark TEXT NOT NULL DEFAULT '',
			enabled INTEGER NOT NULL DEFAULT 1,
			uuid TEXT NOT NULL DEFAULT '',
			username TEXT NOT NULL DEFAULT '',
			password TEXT NOT NULL DEFAULT '',
			method TEXT NOT NULL DEFAULT '',
			ws_path TEXT NOT NULL DEFAULT '/',
			sni TEXT NOT NULL DEFAULT '',
			flow TEXT NOT NULL DEFAULT '',
			up_bytes INTEGER NOT NULL DEFAULT 0,
			down_bytes INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);`,
	}
	for _, s := range stmts {
		if _, err := DB.Exec(s); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	// Backfill columns added after the initial release.
	_, _ = DB.Exec(`ALTER TABLE inbounds ADD COLUMN username TEXT NOT NULL DEFAULT ''`)
	return nil
}

func GetUser(username string) (*models.User, error) {
	row := DB.QueryRow(`SELECT id, username, password_hash, created_at FROM users WHERE username = ?`, username)
	u := &models.User{}
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt); err != nil {
		return nil, err
	}
	return u, nil
}

func CountUsers() (int, error) {
	var n int
	err := DB.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}

func CreateUser(username, hash string) error {
	_, err := DB.Exec(`INSERT INTO users (username, password_hash, created_at) VALUES (?, ?, ?)`,
		username, hash, time.Now())
	return err
}

func UpdateUserPassword(username, hash string) error {
	_, err := DB.Exec(`UPDATE users SET password_hash = ? WHERE username = ?`, hash, username)
	return err
}

func ListInbounds() ([]models.Inbound, error) {
	rows, err := DB.Query(`SELECT id, tag, protocol, listen, port, network, security, remark, enabled,
		uuid, username, password, method, ws_path, sni, flow, up_bytes, down_bytes, created_at, updated_at
		FROM inbounds ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Inbound
	for rows.Next() {
		var i models.Inbound
		var enabled int
		if err := rows.Scan(&i.ID, &i.Tag, &i.Protocol, &i.Listen, &i.Port, &i.Network, &i.Security,
			&i.Remark, &enabled, &i.UUID, &i.Username, &i.Password, &i.Method, &i.WSPath, &i.SNI, &i.Flow,
			&i.UpBytes, &i.DownBytes, &i.CreatedAt, &i.UpdatedAt); err != nil {
			return nil, err
		}
		i.Enabled = enabled == 1
		out = append(out, i)
	}
	return out, nil
}

func GetInbound(id int64) (*models.Inbound, error) {
	row := DB.QueryRow(`SELECT id, tag, protocol, listen, port, network, security, remark, enabled,
		uuid, username, password, method, ws_path, sni, flow, up_bytes, down_bytes, created_at, updated_at
		FROM inbounds WHERE id = ?`, id)
	var i models.Inbound
	var enabled int
	if err := row.Scan(&i.ID, &i.Tag, &i.Protocol, &i.Listen, &i.Port, &i.Network, &i.Security,
		&i.Remark, &enabled, &i.UUID, &i.Username, &i.Password, &i.Method, &i.WSPath, &i.SNI, &i.Flow,
		&i.UpBytes, &i.DownBytes, &i.CreatedAt, &i.UpdatedAt); err != nil {
		return nil, err
	}
	i.Enabled = enabled == 1
	return &i, nil
}

func CreateInbound(in *models.Inbound) (int64, error) {
	in.CreatedAt = time.Now()
	in.UpdatedAt = in.CreatedAt
	res, err := DB.Exec(`INSERT INTO inbounds
		(tag, protocol, listen, port, network, security, remark, enabled, uuid, username, password, method,
		 ws_path, sni, flow, up_bytes, down_bytes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, 0, ?, ?)`,
		in.Tag, in.Protocol, in.Listen, in.Port, in.Network, in.Security, in.Remark, boolToInt(in.Enabled),
		in.UUID, in.Username, in.Password, in.Method, in.WSPath, in.SNI, in.Flow, in.CreatedAt, in.UpdatedAt)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func UpdateInbound(in *models.Inbound) error {
	in.UpdatedAt = time.Now()
	_, err := DB.Exec(`UPDATE inbounds SET tag=?, protocol=?, listen=?, port=?, network=?, security=?,
		remark=?, enabled=?, uuid=?, username=?, password=?, method=?, ws_path=?, sni=?, flow=?, updated_at=?
		WHERE id=?`,
		in.Tag, in.Protocol, in.Listen, in.Port, in.Network, in.Security, in.Remark, boolToInt(in.Enabled),
		in.UUID, in.Username, in.Password, in.Method, in.WSPath, in.SNI, in.Flow, in.UpdatedAt, in.ID)
	return err
}

func DeleteInbound(id int64) error {
	_, err := DB.Exec(`DELETE FROM inbounds WHERE id = ?`, id)
	return err
}

func GetSetting(key, def string) string {
	var v string
	err := DB.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&v)
	if err != nil {
		return def
	}
	return v
}

func SetSetting(key, value string) error {
	_, err := DB.Exec(`INSERT INTO settings (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func MustClose() {
	if DB != nil {
		if err := DB.Close(); err != nil {
			log.Printf("db close: %v", err)
		}
	}
}
