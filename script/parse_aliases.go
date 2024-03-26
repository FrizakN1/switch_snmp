package main

import (
	"encoding/json"
	"fmt"
	"golang.org/x/text/encoding/charmap"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type Aliases struct {
	Mac map[string]Mac `json:"aliases"`
}

type Mac struct {
	IPAddress string `json:"IPAddress"`
	Comment   string `json:"Comment"`
}

func main() {
	aliases := make(map[string]Mac)

	err := filepath.Walk("../aliases", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if strings.HasPrefix(info.Name(), "vlan") {
				macMap, err := parseVLAN(path)
				if err != nil {
					return err
				}

				for mac, macData := range macMap {
					aliases[mac] = macData
				}
			}
		}
		return nil
	})
	if err != nil {
		fmt.Println("Ошибка при переборе директорий:", err)
		return
	}

	vlanJSON, err := json.MarshalIndent(Aliases{aliases}, "", "   ")
	if err != nil {
		fmt.Println("Ошибка при преобразовании в JSON:", err)
		return
	}

	err = ioutil.WriteFile("../aliases.json", vlanJSON, 0644)
	if err != nil {
		fmt.Println("Ошибка при записи в файл:", err)
		return
	}

	fmt.Println("JSON файл успешно создан: aliases.json")
}

func parseVLAN(path string) (map[string]Mac, error) {
	macs := make(map[string]Mac)

	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if file.IsDir() && isValidIP(file.Name()) {
			ip := file.Name()
			ipContent, err := ioutil.ReadDir(filepath.Join(path, ip))
			if err != nil {
				return nil, err
			}

			for _, file1 := range ipContent {
				var ipAddr string
				var mac []byte
				var comment []byte
				var macFin string

				if !file1.IsDir() {
					if len(ipContent) > 20 {
						if isValidIP(strings.TrimRight(file1.Name(), ".comment")) {
							ipAddr = strings.TrimRight(file1.Name(), ".comment")

							commentContent, err := ioutil.ReadFile(filepath.Join(path, ip, file1.Name()))
							if err != nil {
								return nil, err
							}

							comment, err = charmap.KOI8R.NewDecoder().Bytes(commentContent)
							if err != nil {
								return nil, err
							}

							mac, err = ioutil.ReadFile(filepath.Join(path, ip, ipAddr))
							if err != nil {
								return nil, err
							}

							macFin = strings.TrimRight(string(mac), "\n")
						}
					} else {
						if isValidIP(file1.Name()) {
							ipAddr = file1.Name()
							mac, err = ioutil.ReadFile(filepath.Join(path, ip, ipAddr))
							macFin = strings.TrimRight(string(mac), "\n")
							if err != nil {
								return nil, err
							}

							_, err = os.Stat(filepath.Join(path, ip, "comment"))
							if err != nil {
								continue
							}

							commentContent, err := ioutil.ReadFile(filepath.Join(path, ip, "comment"))
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

				macData := Mac{
					IPAddress: ipAddr,
					Comment:   string(comment),
				}

				macs[macFin] = macData
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
