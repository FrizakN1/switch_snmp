package service

import (
	"fmt"
	"strings"

	g "github.com/gosnmp/gosnmp"

	"snmp/internal/domain"
)

func saveSwitchConfig(snmp *g.GoSNMP, switchModel string, sw domain.SwitchOID) error {
	model := strings.ToUpper(strings.TrimSpace(switchModel))

	// Eltex MES2428* family: save requires two OIDs in one snmpset call.
	if strings.HasPrefix(model, "MES2428") {
		_, err := snmp.Set([]g.SnmpPDU{
			{Name: "1.3.6.1.4.1.2076.81.1.10.0", Value: 4, Type: g.Integer},
			{Name: "1.3.6.1.4.1.2076.81.1.13.0", Value: 1, Type: g.Integer},
		})
		return err
	}

	// Eltex MES2324FB-like family (and similar models sharing this base save tree).
	if strings.HasPrefix(model, "MES") && sw.SaveConfig == "1.3.6.1.4.1.89.87.2.1" {
		_, err := snmp.Set([]g.SnmpPDU{
			{Name: "1.3.6.1.4.1.89.87.2.1.3.1", Value: 1, Type: g.Integer},
			{Name: "1.3.6.1.4.1.89.87.2.1.7.1", Value: 2, Type: g.Integer},
			{Name: "1.3.6.1.4.1.89.87.2.1.8.1", Value: 1, Type: g.Integer},
			{Name: "1.3.6.1.4.1.89.87.2.1.12.1", Value: 3, Type: g.Integer},
			{Name: "1.3.6.1.4.1.89.87.2.1.17.1", Value: 4, Type: g.Integer},
		})
		return err
	}

	// Default behavior (D-Link and other models using single SaveConfig OID).
	if sw.SaveConfig == "" {
		return fmt.Errorf("save config not supported for this model")
	}

	_, err := snmp.Set([]g.SnmpPDU{{Name: sw.SaveConfig, Value: 1, Type: g.Integer}})
	return err
}
