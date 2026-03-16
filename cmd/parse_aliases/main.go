package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/text/encoding/charmap"
)

type Aliases struct {
	Mac map[string]Mac `json:"aliases"`
}

type Mac struct {
	IPAddress string `json:"IPAddress"`
	Comment   string `json:"Comment"`
}

func main() {
	inDir := flag.String("in", "aliases", "Input directory containing vlan* folders")
	outFile := flag.String("out", "aliases.json", "Output JSON file")
	flag.Parse()

	aliases := make(map[string]Mac)

	err := filepath.WalkDir(*inDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if strings.HasPrefix(d.Name(), "vlan") {
			macMap, err := parseVLAN(path)
			if err != nil {
				return err
			}
			for mac, macData := range macMap {
				aliases[mac] = macData
			}
		}
		return nil
	})
	if err != nil {
		fmt.Println("walk error:", err)
		os.Exit(1)
	}

	vlanJSON, err := json.MarshalIndent(Aliases{aliases}, "", "  ")
	if err != nil {
		fmt.Println("json error:", err)
		os.Exit(1)
	}

	if err := os.WriteFile(*outFile, vlanJSON, 0644); err != nil {
		fmt.Println("write error:", err)
		os.Exit(1)
	}

	fmt.Println("written:", *outFile)
}

func parseVLAN(path string) (map[string]Mac, error) {
	macs := make(map[string]Mac)

	files, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if !file.IsDir() || !isValidIP(file.Name()) {
			continue
		}

		ip := file.Name()
		ipDir := filepath.Join(path, ip)
		ipContent, err := os.ReadDir(ipDir)
		if err != nil {
			return nil, err
		}

		for _, entry := range ipContent {
			if entry.IsDir() {
				continue
			}

			var ipAddr string
			var macFin string
			var comment []byte

			// Heuristic preserved from the original script: folder layout varies depending on size.
			if len(ipContent) > 20 {
				if isValidIP(strings.TrimSuffix(entry.Name(), ".comment")) {
					ipAddr = strings.TrimSuffix(entry.Name(), ".comment")

					commentContent, err := os.ReadFile(filepath.Join(ipDir, entry.Name()))
					if err != nil {
						return nil, err
					}
					comment, err = charmap.KOI8R.NewDecoder().Bytes(commentContent)
					if err != nil {
						return nil, err
					}

					mac, err := os.ReadFile(filepath.Join(ipDir, ipAddr))
					if err != nil {
						return nil, err
					}
					macFin = strings.TrimRight(string(mac), "\n")
				}
			} else {
				if isValidIP(entry.Name()) {
					ipAddr = entry.Name()
					mac, err := os.ReadFile(filepath.Join(ipDir, ipAddr))
					if err != nil {
						return nil, err
					}
					macFin = strings.TrimRight(string(mac), "\n")

					if _, err := os.Stat(filepath.Join(ipDir, "comment")); err == nil {
						commentContent, err := os.ReadFile(filepath.Join(ipDir, "comment"))
						if err != nil {
							return nil, err
						}
						comment, err = charmap.KOI8R.NewDecoder().Bytes(commentContent)
						if err != nil {
							return nil, err
						}
					}
				}
			}

			if macFin == "" {
				continue
			}

			macs[macFin] = Mac{
				IPAddress: ipAddr,
				Comment:   string(comment),
			}
		}
	}

	return macs, nil
}

func isValidIP(ip string) bool {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return false
	}
	for _, part := range parts {
		if len(part) == 0 {
			return false
		}
	}
	return true
}
