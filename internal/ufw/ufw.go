package ufw

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Available reports whether the `ufw` command exists on this host.
func Available() bool {
	_, err := exec.LookPath("ufw")
	return err == nil
}

// Status returns the active/inactive status reported by ufw.
func Status() (string, error) {
	if !Available() {
		return "not installed", nil
	}
	out, err := exec.Command("ufw", "status").CombinedOutput()
	if err != nil {
		return string(out), err
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(strings.ToLower(line), "status:") {
			return strings.TrimSpace(strings.TrimPrefix(strings.ToLower(line), "status:")), nil
		}
	}
	return "unknown", nil
}

// AllowPort opens the given port in ufw for the given protocol (tcp/udp).
// If proto is empty, both tcp and udp rules are added (mirrors `ufw allow <port>`).
func AllowPort(port int, proto string) error {
	if !Available() {
		return nil
	}
	args := []string{"allow", strconv.Itoa(port)}
	if p := normalizeProto(proto); p != "" {
		args = []string{"allow", fmt.Sprintf("%d/%s", port, p)}
	}
	out, err := exec.Command("ufw", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("ufw allow %d: %v: %s", port, err, string(out))
	}
	return nil
}

// DenyPort removes a previously-allowed rule.
func DenyPort(port int, proto string) error {
	if !Available() {
		return nil
	}
	args := []string{"delete", "allow", strconv.Itoa(port)}
	if p := normalizeProto(proto); p != "" {
		args = []string{"delete", "allow", fmt.Sprintf("%d/%s", port, p)}
	}
	out, err := exec.Command("ufw", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("ufw delete %d: %v: %s", port, err, string(out))
	}
	return nil
}

func normalizeProto(p string) string {
	p = strings.ToLower(strings.TrimSpace(p))
	switch p {
	case "tcp", "udp":
		return p
	}
	return ""
}
