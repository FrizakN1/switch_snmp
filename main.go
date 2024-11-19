package main

import (
	"fmt"
	"snmp/router"
	"snmp/settings"
)

func main() {
	config := settings.Load("settings/settings.json")

	err := router.InitializationAliases()
	if err != nil {
		fmt.Println(err)
		return
	}

	_ = router.Initialization().Run(config.Address + ":" + config.Port)
}
