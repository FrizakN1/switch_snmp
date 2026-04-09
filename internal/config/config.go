package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Address                string
	Port                   string
	EltexReadOnlyCommunity string
	DlinkReadOnlyCommunity string
	TVBSReadOnlyCommunity  string
	ReadWriteCommunity     string
	SwitchesFilePath       string
}

func Load() (*Config, error) {
	// Load .env into the process env if present. Missing file is OK.
	_ = godotenv.Load()

	cfg := Config{
		Address:          "localhost",
		Port:             "8080",
		SwitchesFilePath: "switches.txt",
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
	if v := os.Getenv("SNMP_TVBS_RO_COMMUNITY"); v != "" {
		cfg.TVBSReadOnlyCommunity = v
	}
	if v := os.Getenv("SNMP_RW_COMMUNITY"); v != "" {
		cfg.ReadWriteCommunity = v
	}
	if v := os.Getenv("SNMP_SWITCHES_FILE"); v != "" {
		cfg.SwitchesFilePath = v
	}

	var missing []string
	if cfg.EltexReadOnlyCommunity == "" {
		missing = append(missing, "SNMP_ELTEX_RO_COMMUNITY")
	}
	if cfg.DlinkReadOnlyCommunity == "" {
		missing = append(missing, "SNMP_DLINK_RO_COMMUNITY")
	}
	if cfg.TVBSReadOnlyCommunity == "" {
		missing = append(missing, "SNMP_TVBS_RO_COMMUNITY")
	}
	if cfg.ReadWriteCommunity == "" {
		missing = append(missing, "SNMP_RW_COMMUNITY")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required env vars: %v", missing)
	}

	return &cfg, nil
}
