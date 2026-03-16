package main

import (
	"fmt"
	"log"

	"snmp/internal/aliases"
	"snmp/internal/config"
	"snmp/internal/service"
	httptransport "snmp/internal/transport/http"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	aliasStore, err := aliases.Load("aliases.json")
	if err != nil {
		log.Fatal(err)
	}

	dlink := service.NewDlink(cfg, aliasStore)
	eltex := service.NewEltex(cfg, aliasStore)

	srv := httptransport.NewServer(cfg, dlink, eltex)
	addr := fmt.Sprintf("%s:%s", cfg.Address, cfg.Port)
	if err := srv.Router().Run(addr); err != nil {
		log.Fatal(err)
	}
}
