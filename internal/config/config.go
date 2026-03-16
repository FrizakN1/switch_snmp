package config

import (
	"fmt"
	"github.com/joho/godotenv"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Address                string
	Port                   string
	EltexReadOnlyCommunity string
	DlinkReadOnlyCommunity string
	ReadWriteCommunity     string
}

func Load() (*Config, error) {
	// Load .env into the process env.
	// Common failure on VMs: process starts with a different working directory, so ".env" isn't found.
	// Another common failure: ".env" exists but is saved in a wrong encoding, and parsing fails.
	loadedFrom, err := loadDotEnv()
	if err != nil {
		return nil, err
	}

	cfg := Config{
		Address: "localhost",
		Port:    "8080",
	}

	// Environment config (useful for secrets and deployment).
	if v := os.Getenv("SNMP_ADDRESS"); v != "" {
		cfg.Address = v
	}
	if v := os.Getenv("SNMP_PORT"); v != "" {
		cfg.Port = v
	}
	if v := os.Getenv("SNMP_ELTEX_RO_COMMUNITY"); v != "" {
		cfg.EltexReadOnlyCommunity = v
	}
	if v := os.Getenv("SNMP_DLINK_RO_COMMUNITY"); v != "" {
		cfg.DlinkReadOnlyCommunity = v
	}
	if v := os.Getenv("SNMP_RW_COMMUNITY"); v != "" {
		cfg.ReadWriteCommunity = v
	}

	var missing []string
	if cfg.EltexReadOnlyCommunity == "" {
		missing = append(missing, "SNMP_ELTEX_RO_COMMUNITY")
	}
	if cfg.DlinkReadOnlyCommunity == "" {
		missing = append(missing, "SNMP_DLINK_RO_COMMUNITY")
	}
	if cfg.ReadWriteCommunity == "" {
		missing = append(missing, "SNMP_RW_COMMUNITY")
	}
	if len(missing) > 0 {
		hint := ""
		if loadedFrom == "" {
			hint = " (.env was not loaded; set env vars or set SNMP_ENV_FILE=/path/to/.env, or run from the repo root)"
		} else {
			hint = fmt.Sprintf(" (.env loaded from %q)", loadedFrom)
		}
		return nil, fmt.Errorf("missing required env vars: %v%s", missing, hint)
	}

	return &cfg, nil
}

func loadDotEnv() (string, error) {
	// Explicit path takes priority.
	if p := strings.TrimSpace(os.Getenv("SNMP_ENV_FILE")); p != "" {
		if err := godotenv.Load(p); err != nil {
			return "", fmt.Errorf("failed to load SNMP_ENV_FILE=%q: %w", p, err)
		}
		return p, nil
	}

	// 1) Current working directory.
	if _, err := os.Stat(".env"); err == nil {
		if err := godotenv.Load(".env"); err != nil {
			return "", fmt.Errorf("found .env in working directory but failed to parse it: %w", err)
		}
		return ".env", nil
	}

	// 2) Directory of the executable and a couple of parents (useful for service managers).
	exe, err := os.Executable()
	if err == nil {
		dir := filepath.Dir(exe)
		candidates := []string{
			filepath.Join(dir, ".env"),
			filepath.Join(filepath.Dir(dir), ".env"),
			filepath.Join(filepath.Dir(filepath.Dir(dir)), ".env"),
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				if err := godotenv.Load(c); err != nil {
					return "", fmt.Errorf("found .env at %q but failed to parse it: %w", c, err)
				}
				return c, nil
			}
		}
	}

	// Not found: OK, we might run with real environment variables.
	return "", nil
}
