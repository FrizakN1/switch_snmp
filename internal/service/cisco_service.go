package service

import (
	"fmt"
	"strings"

	"snmp/internal/aliases"
	"snmp/internal/config"
	"snmp/internal/domain"
	snmpx "snmp/internal/snmp"
	"snmp/internal/switchdb"
)

type CiscoService struct {
	cfg     *config.Config
	aliases *aliases.Store
}

func NewCisco(cfg *config.Config, aliases *aliases.Store) *CiscoService {
	return &CiscoService{cfg: cfg, aliases: aliases}
}

func (s *CiscoService) Get(ip string) (*domain.ViewData, error) {
	snmp := snmpx.NewClient(ip, s.cfg.CiscoReadOnlyCommunity)
	if err := snmp.Connect(); err != nil {
		return nil, err
	}
	defer snmp.Conn.Close()

	switchModel, err := getSwitchModel(snmp)
	if err != nil {
		return nil, err
	}

	if strings.Contains(switchModel, "Cisco") {
		switchModel = "Cisco"
	} else {
		return nil, fmt.Errorf("switch model %q is not Cisco", switchModel)
	}

	sw, ok := switchdb.Switches[switchModel]
	if !ok {
		return nil, fmt.Errorf("cisco oid profile is not configured")
	}

	systemName, _ := snmpx.GetStringValue(snmp, sw.SystemName)
	systemName = fmt.Sprintf("%s (%s)", systemName, switchModel)

	portMap, err := getCiscoPorts(snmp, sw)
	if err != nil {
		return nil, err
	}

	firmware, _ := snmpx.GetStringValue(snmp, sw.Firmware)
	snValue, _ := snmpx.GetStringValue(snmp, sw.SN)
	uptime, _ := snmpx.GetUptime(snmp)

	return &domain.ViewData{
		Ports:              portMap,
		SystemName:         systemName,
		SN:                 snValue,
		IP:                 ip,
		Firmware:           firmware,
		Uptime:             uptime,
		Type:               "Cisco",
		CanChange:          false,
		CanChangeBandwidth: false,
	}, nil
}

func (s *CiscoService) GetMacs(ip string) (map[int][]string, error) {
	snmp := snmpx.NewClient(ip, s.cfg.CiscoReadOnlyCommunity)
	if err := snmp.Connect(); err != nil {
		return nil, err
	}
	defer snmp.Conn.Close()

	switchModel, err := getSwitchModel(snmp)
	if err != nil {
		return nil, err
	}
	if !strings.Contains(switchModel, "Cisco") {
		return nil, fmt.Errorf("switch model %q is not Cisco", switchModel)
	}

	portMap := make(map[int]domain.Port)
	if err := snmpx.GetMacAddresses(snmp, s.aliases, portMap, "Cisco"); err != nil {
		return nil, err
	}

	displayOrder, err := getCiscoDisplayOrder(snmp, switchdb.Switches["Cisco"])
	if err != nil {
		return nil, err
	}

	out := make(map[int][]string, len(portMap))
	for ifIndex, p := range portMap {
		if displayKey, ok := displayOrder[ifIndex]; ok {
			out[displayKey] = p.Macs
		}
	}
	return out, nil
}
