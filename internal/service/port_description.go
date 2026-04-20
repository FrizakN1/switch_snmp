package service

import (
	"fmt"
	"strings"

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

	model := strings.ToUpper(req.SwitchModel)
	isEltex := strings.HasPrefix(model, "MES")

	var primaryOID, secondaryOID string

	if isEltex {
		primaryOID = sw.PortDesc
		secondaryOID = snmpx.DefaultPortDescOID
	} else {
		primaryOID = snmpx.DefaultPortDescOID
		secondaryOID = sw.PortDesc
	}

	if err := setPortDesc(primaryOID); err != nil {
		if fallbackErr := setPortDesc(secondaryOID); fallbackErr != nil {
			return fmt.Errorf("set port description failed with model oid (%s): %v; fallback oid (%s): %w", primaryOID, err, secondaryOID, fallbackErr)
		}
	}

	param := []g.SnmpPDU{{Name: sw.SaveConfig, Value: 1, Type: g.Integer}}
	if _, err := snmp.Set(param); err != nil {
		return err
	}

	return nil
}
