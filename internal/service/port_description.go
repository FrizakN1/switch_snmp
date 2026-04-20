package service

import (
	"fmt"

	g "github.com/gosnmp/gosnmp"

	snmpx "snmp/internal/snmp"
	"snmp/internal/switchdb"
)

type ChangePortDescriptionRequest struct {
	Index       int
	SwitchModel string
	Description string
}

func ChangePortDescription(ip string, readWriteCommunity string, req ChangePortDescriptionRequest) error {
	sw, ok := switchdb.Switches[req.SwitchModel]
	if !ok || sw.SaveConfig == "" {
		return fmt.Errorf("unknown switch model or SaveConfig not supported")
	}

	snmp := snmpx.NewClient(ip, readWriteCommunity)
	if err := snmp.Connect(); err != nil {
		return err
	}
	defer snmp.Conn.Close()

	setPortDesc := func(baseOID string) error {
		oid := fmt.Sprintf("%s.%d", baseOID, req.Index)
		param := []g.SnmpPDU{{Name: oid, Value: req.Description, Type: g.OctetString}}
		_, err := snmp.Set(param)
		return err
	}

	primaryOID := sw.PortDesc
	if primaryOID == "" {
		primaryOID = snmpx.DefaultPortDescOID
	}

	if err := setPortDesc(primaryOID); err != nil {
		if primaryOID == snmpx.DefaultPortDescOID {
			return err
		}
		if fallbackErr := setPortDesc(snmpx.DefaultPortDescOID); fallbackErr != nil {
			return fmt.Errorf("set port description failed with model oid (%s): %v; fallback oid (%s): %w", primaryOID, err, snmpx.DefaultPortDescOID, fallbackErr)
		}
	}

	param := []g.SnmpPDU{{Name: sw.SaveConfig, Value: 1, Type: g.Integer}}
	if _, err := snmp.Set(param); err != nil {
		return err
	}

	return nil
}
