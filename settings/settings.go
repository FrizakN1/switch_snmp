package settings

import (
	"encoding/json"
	"fmt"
	"snmp/utils"
)

type Setting struct {
	Address                string
	Port                   string
	EltexReadOnlyCommunity string
	DlinkReadOnlyCommunity string
	ReadWriteCommunity     string
}

var settings Setting

func Load(filename string) *Setting {
	bytes, e := utils.LoadFile(filename)
	if e != nil {
		fmt.Println(e)
		return nil
	}
	e = json.Unmarshal(bytes, &settings)
	if e != nil {
		fmt.Println(e)
		return nil
	}
	return &settings
}
