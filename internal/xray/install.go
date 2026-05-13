package xray

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	xrayRepo    = "XTLS/Xray-core"
	servicePath = "/etc/systemd/system/xray.service"
)

// BinaryPath is the on-disk location of the xray binary.
var BinaryPath = "/usr/local/bin/xray"

func init() {
	if v := os.Getenv("VPN_PROXY_XRAY_BIN"); v != "" {
		BinaryPath = v
	}
}

// Status describes the current Xray runtime state.
type Status struct {
	Installed bool   `json:"installed"`
	Version   string `json:"version"`
	Running   bool   `json:"running"`
	Path      string `json:"path"`
}

func GetStatus() Status {
	s := Status{Path: BinaryPath}
	if fi, err := os.Stat(BinaryPath); err == nil && !fi.IsDir() {
		s.Installed = true
		if out, err := exec.Command(BinaryPath, "version").Output(); err == nil {
			s.Version = firstWord(string(out), "Xray")
		}
	}
	if _, err := exec.LookPath("systemctl"); err == nil {
		out, _ := exec.Command("systemctl", "is-active", "xray").Output()
		s.Running = strings.TrimSpace(string(out)) == "active"
	}
	return s
}

func firstWord(out, marker string) string {
	for _, ln := range strings.Split(out, "\n") {
		if strings.Contains(ln, marker) {
			return strings.TrimSpace(ln)
		}
	}
	if ln := strings.SplitN(out, "\n", 2)[0]; ln != "" {
		return strings.TrimSpace(ln)
	}
	return ""
}

// Install fetches the latest Xray-core release for the host architecture,
// installs the binary to BinaryPath, writes a systemd unit, and starts the
// service. It is a no-op if installation appears already current and `force`
// is false.
func Install(force bool) (string, error) {
	if !force {
		if s := GetStatus(); s.Installed && s.Running {
			return "already installed and running: " + s.Version, nil
		}
	}
	assetName, err := assetForHost()
	if err != nil {
		return "", err
	}
	rel, err := latestRelease()
	if err != nil {
		return "", fmt.Errorf("fetch release: %w", err)
	}
	url := pickAsset(rel, assetName)
	if url == "" {
		return "", fmt.Errorf("no asset %q in latest release", assetName)
	}
	if err := downloadAndExtract(url); err != nil {
		return "", err
	}
	if err := writeServiceUnit(); err != nil {
		return "", err
	}
	if err := WriteConfig(); err != nil {
		return "", fmt.Errorf("write config: %w", err)
	}
	if err := runSystemctl("daemon-reload"); err != nil {
		return "", err
	}
	_ = runSystemctl("enable", "xray")
	if err := runSystemctl("restart", "xray"); err != nil {
		return "", err
	}
	s := GetStatus()
	return fmt.Sprintf("installed %s (running=%v)", s.Version, s.Running), nil
}

func runSystemctl(args ...string) error {
	if _, err := exec.LookPath("systemctl"); err != nil {
		return nil
	}
	out, err := exec.Command("systemctl", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl %s: %v: %s", strings.Join(args, " "), err, string(out))
	}
	return nil
}

func ServiceCommand(action string) error {
	switch action {
	case "start", "stop", "restart":
	default:
		return errors.New("invalid action")
	}
	return runSystemctl(action, "xray")
}

func assetForHost() (string, error) {
	if runtime.GOOS != "linux" {
		return "", fmt.Errorf("auto-install only supported on linux, got %s", runtime.GOOS)
	}
	switch runtime.GOARCH {
	case "amd64":
		return "Xray-linux-64.zip", nil
	case "arm64":
		return "Xray-linux-arm64-v8a.zip", nil
	case "arm":
		return "Xray-linux-arm32-v7a.zip", nil
	}
	return "", fmt.Errorf("unsupported arch %s", runtime.GOARCH)
}

type ghRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}

func latestRelease() (*ghRelease, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/repos/"+xrayRepo+"/releases/latest", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github api %d: %s", resp.StatusCode, string(b))
	}
	var r ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, err
	}
	return &r, nil
}

func pickAsset(r *ghRelease, name string) string {
	for _, a := range r.Assets {
		if a.Name == name {
			return a.URL
		}
	}
	return ""
}

func downloadAndExtract(url string) error {
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("download status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return fmt.Errorf("unzip: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(BinaryPath), 0o755); err != nil {
		return err
	}
	for _, f := range zr.File {
		base := filepath.Base(f.Name)
		if base != "xray" && base != "xray.exe" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(BinaryPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
		if err != nil {
			rc.Close()
			return err
		}
		if _, err := io.Copy(out, rc); err != nil {
			rc.Close()
			out.Close()
			return err
		}
		rc.Close()
		out.Close()
		return os.Chmod(BinaryPath, 0o755)
	}
	return errors.New("xray binary not found in archive")
}

func writeServiceUnit() error {
	if _, err := exec.LookPath("systemctl"); err != nil {
		return nil // skip on systems without systemd
	}
	unit := fmt.Sprintf(`[Unit]
Description=Xray Service
Documentation=https://github.com/%s
After=network.target nss-lookup.target

[Service]
User=root
NoNewPrivileges=true
ExecStart=%s run -config %s
Restart=on-failure
RestartPreventExitStatus=23
LimitNPROC=10000
LimitNOFILE=1000000

[Install]
WantedBy=multi-user.target
`, xrayRepo, BinaryPath, ConfigPath)
	return os.WriteFile(servicePath, []byte(unit), 0o644)
}
