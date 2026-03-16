package service

import (
	"fmt"
	"strconv"
	"strings"

	g "github.com/gosnmp/gosnmp"

	"snmp/internal/aliases"
	"snmp/internal/config"
	"snmp/internal/domain"
	snmpx "snmp/internal/snmp"
	"snmp/internal/switchdb"
)

type DlinkService struct {
	cfg     *config.Config
	aliases *aliases.Store
}

func NewDlink(cfg *config.Config, aliases *aliases.Store) *DlinkService {
	return &DlinkService{cfg: cfg, aliases: aliases}
}

func (s *DlinkService) Get(ip string) (*domain.ViewData, error) {
	snmp := snmpx.NewClient(ip, s.cfg.DlinkReadOnlyCommunity)
	if err := snmp.Connect(); err != nil {
		return nil, err
	}
	defer snmp.Conn.Close()

	switchModel, err := getSwitchModel(snmp)
	if err != nil {
		return nil, err
	}

	sw, ok := switchdb.Switches[switchModel]
	if !ok {
		for key := range switchdb.Switches {
			if strings.Contains(switchModel, key) {
				sw = switchdb.Switches[key]
				switchModel = key
				break
			}
		}
	}

	systemName, _ := snmpx.GetStringValue(snmp, sw.SystemName)
	if systemName == "#Ошибка" && switchModel == "DES-1210-28" {
		switchModel = "DES-1210-28/ME"
		sw = switchdb.Switches[switchModel]
		systemName, _ = snmpx.GetStringValue(snmp, sw.SystemName)
	}

	sysName, _ := snmpx.GetStringValue(snmp, sw.SystemName)
	systemName = fmt.Sprintf("%s (%s)", sysName, switchModel)

	portMap := make(map[int]domain.Port)
	if sw.PortDesc == "" {
		for i := 1; i <= sw.PortAmount; i++ {
			portMap[i] = domain.Port{Index: i}
		}
	} else {
		_ = snmpx.GetPortsDescription(snmp, portMap, sw, switchModel)
	}

	if err := getDGSPortsVlan(snmp, portMap, sw, switchModel); err != nil {
		return nil, err
	}
	formatVlans(portMap)

	if switchModel == "DGS-1100-26/ME" || switchModel == "DGS-3120-24SC" {
		_ = snmpx.GetPortsSpeed(snmp, portMap)
	} else if sw.Speed != "" {
		_ = getDGSPortsSpeed(snmp, portMap, sw.Speed)
	}

	_ = snmpx.GetMacAddresses(snmp, s.aliases, portMap, switchModel)

	firmware, _ := snmpx.GetStringValue(snmp, sw.Firmware)

	SN := "#Неизвестно"
	if sw.SN != "" {
		SN, _ = snmpx.GetStringValue(snmp, sw.SN)
	}

	uptime, _ := snmpx.GetUptime(snmp)

	canChangeBandwidth := sw.BandwidthTX != "" && sw.BandwidthRX != ""
	if canChangeBandwidth {
		_ = getBandwidthControl(snmp, portMap, sw.BandwidthTX, sw.BandwidthRX)
	}

	return &domain.ViewData{
		Ports:              portMap,
		SystemName:         systemName,
		SN:                 SN,
		IP:                 ip,
		Firmware:           firmware,
		Uptime:             uptime,
		Type:               "DGS",
		CanChange:          sw.PortDesc != "" && sw.SaveConfig != "",
		CanChangeBandwidth: canChangeBandwidth,
	}, nil
}

type ChangePortDescriptionRequest struct {
	Index       int
	SwitchModel string
	Description string
}

func (s *DlinkService) ChangePortDescription(ip string, req ChangePortDescriptionRequest) error {
	sw, ok := switchdb.Switches[req.SwitchModel]
	if !ok || sw.SaveConfig == "" {
		return fmt.Errorf("unknown switch model or SaveConfig not supported")
	}

	snmp := snmpx.NewClient(ip, s.cfg.ReadWriteCommunity)
	if err := snmp.Connect(); err != nil {
		return err
	}
	defer snmp.Conn.Close()

	oid := fmt.Sprintf("1.3.6.1.2.1.31.1.1.1.18.%d", req.Index)
	param := []g.SnmpPDU{{Name: oid, Value: req.Description, Type: g.OctetString}}
	if _, err := snmp.Set(param); err != nil {
		return err
	}

	param = []g.SnmpPDU{{Name: sw.SaveConfig, Value: 1, Type: g.Integer}}
	if _, err := snmp.Set(param); err != nil {
		return err
	}

	return nil
}

type ChangeBandwidthRequest struct {
	Index       int
	SwitchModel string
	BandwidthTX int
	BandwidthRX int
}

func (s *DlinkService) ChangeBandwidth(ip string, req ChangeBandwidthRequest) error {
	sw, ok := switchdb.Switches[req.SwitchModel]
	if !ok || sw.BandwidthTX == "" || sw.BandwidthRX == "" {
		return fmt.Errorf("bandwidth control not supported for this model")
	}

	snmp := snmpx.NewClient(ip, s.cfg.ReadWriteCommunity)
	if err := snmp.Connect(); err != nil {
		return err
	}
	defer snmp.Conn.Close()

	// Some switches expose tables as <oid>.<port>, others as <oid>.1.<port>.
	trySetInt := func(baseOID string, port int, value int) error {
		oid := fmt.Sprintf("%s.%d", baseOID, port)
		_, err := snmp.Set([]g.SnmpPDU{{Name: oid, Value: value, Type: g.Integer}})
		if err == nil {
			return nil
		}
		oid2 := fmt.Sprintf("%s.1.%d", baseOID, port)
		_, err2 := snmp.Set([]g.SnmpPDU{{Name: oid2, Value: value, Type: g.Integer}})
		if err2 == nil {
			return nil
		}
		return err
	}

	if err := trySetInt(sw.BandwidthTX, req.Index, req.BandwidthTX); err != nil {
		return err
	}
	if err := trySetInt(sw.BandwidthRX, req.Index, req.BandwidthRX); err != nil {
		return err
	}

	return nil
}

func getSwitchModel(snmp *g.GoSNMP) (string, error) {
	v, err := snmpx.GetStringValue(snmp, "1.3.6.1.2.1.1.1.0")
	if err != nil {
		return "", err
	}
	return strings.Split(v, " ")[0], nil
}

func formatVlans(portMap map[int]domain.Port) {
	for key, port := range portMap {
		if len(port.Vlan) == 0 {
			continue
		}

		vlansStringArray := strings.Split(port.Vlan[:len(port.Vlan)-1], ",")
		vlans := make([]int, 0, len(vlansStringArray))

		for _, vlanString := range vlansStringArray {
			vlanInt, err := strconv.Atoi(vlanString)
			if err != nil {
				return
			}
			vlans = append(vlans, vlanInt)
		}

		port.Vlan = snmpx.FormatRanges(vlans)
		portMap[key] = port
	}
}

func getDGSPortsVlan(snmp *g.GoSNMP, portMap map[int]domain.Port, sw domain.SwitchOID, switchModel string) error {
	oid := sw.Vlan

	var result []g.SnmpPDU
	var err error

	if switchModel == "DGS-1100-26/ME" {
		result, err = snmp.WalkAll(oid)
	} else {
		result, err = snmp.BulkWalkAll(oid)
	}
	if err != nil {
		return err
	}

	for _, variable := range result {
		parts := strings.Split(variable.Name, oid)
		if len(parts) < 2 || len(parts[1]) < 2 {
			continue
		}
		vlan := parts[1][1:]

		fieldsArr, ok := variable.Value.([]byte)
		if !ok {
			continue
		}
		portNumber := 0

		for _, item := range fieldsArr {
			if item == 0 {
				portNumber += 8
				continue
			}

			if item < 16 {
				portNumber += 4
				field := strconv.FormatInt(int64(item), 2)
				portNumber += 4 - len(field)
				for _, char := range field {
					portNumber++
					if char == '1' && portNumber <= sw.PortAmount {
						port := portMap[portNumber]
						port.Vlan += vlan + ","
						port.Mode = "trunk"
						portMap[portNumber] = port
					}
				}
				continue
			}

			hexString := strconv.FormatInt(int64(item), 16)
			for _, el := range strings.Split(hexString, "") {
				if el == "0" {
					portNumber += 4
					continue
				}

				field, err := snmpx.HexToBinary(el)
				if err != nil {
					continue
				}

				portNumber += 4 - len(field)
				for _, char := range field {
					portNumber++
					if char == '1' && portNumber <= sw.PortAmount {
						port := portMap[portNumber]
						port.Vlan += vlan + ","
						port.Mode = "trunk"
						portMap[portNumber] = port
					}
				}
			}
		}
	}

	oid = sw.VlanUntagged

	if switchModel == "DGS-1100-26/ME" {
		result, err = snmp.WalkAll(oid)
	} else {
		result, err = snmp.BulkWalkAll(oid)
	}
	if err != nil {
		return err
	}

	for _, variable := range result {
		fieldsArr, ok := variable.Value.([]byte)
		if !ok {
			continue
		}

		portNumber := 0
		for _, item := range fieldsArr {
			if item == 0 {
				portNumber += 8
				continue
			}

			if item < 16 {
				portNumber += 4
				field := strconv.FormatInt(int64(item), 2)
				portNumber += 4 - len(field)
				for _, char := range field {
					portNumber++
					if char == '1' && portNumber <= sw.PortAmount {
						port := portMap[portNumber]
						port.Mode = "access"
						portMap[portNumber] = port
					}
				}
				continue
			}

			hexString := strconv.FormatInt(int64(item), 16)
			for _, el := range strings.Split(hexString, "") {
				if el == "0" {
					portNumber += 4
					continue
				}

				field, err := snmpx.HexToBinary(el)
				if err != nil {
					continue
				}

				portNumber += 4 - len(field)
				for _, char := range field {
					portNumber++
					if char == '1' && portNumber <= sw.PortAmount {
						port := portMap[portNumber]
						port.Mode = "access"
						portMap[portNumber] = port
					}
				}
			}
		}
	}

	return nil
}

func getDGSPortsSpeed(snmp *g.GoSNMP, portMap map[int]domain.Port, oid string) error {
	result, err := snmp.BulkWalkAll(oid)
	if err != nil {
		return err
	}

	for _, variable := range result {
		oidParts := strings.Split(variable.Name[len(oid)+2:], ".")
		if len(oidParts) == 0 {
			continue
		}

		key, err := strconv.Atoi(oidParts[0])
		if err != nil {
			return err
		}

		speedType, ok := snmpx.AsInt(variable.Value)
		if !ok {
			continue
		}

		port := portMap[key]
		if port.Speed == 0 {
			switch speedType {
			case 1:
				port.Speed = 0
			case 2:
				port.Speed = 1000
			case 3:
				port.Speed = 100
			case 4:
				port.Speed = 50
			case 5:
				port.Speed = 10
			case 6:
				port.Speed = 5
			default:
				port.Speed = -1
			}
		}

		portMap[key] = port
	}

	return nil
}

func getBandwidthControl(snmp *g.GoSNMP, portMap map[int]domain.Port, oidTX, oidRX string) error {
	result, err := snmp.BulkWalkAll(oidTX)
	if err != nil {
		return err
	}

	for _, variable := range result {
		oidParts := strings.Split(variable.Name[len(oidTX)+2:], ".")
		if len(oidParts) == 0 {
			continue
		}

		key, err := strconv.Atoi(oidParts[0])
		if err != nil {
			return err
		}

		bandwidthTX, ok := snmpx.AsInt(variable.Value)
		if !ok {
			continue
		}

		port := portMap[key]
		port.BandwidthTX = "NoLimit"
		if bandwidthTX != 0 {
			port.BandwidthTX = fmt.Sprint(bandwidthTX)
		}
		portMap[key] = port
	}

	result, err = snmp.BulkWalkAll(oidRX)
	if err != nil {
		return err
	}

	for _, variable := range result {
		oidParts := strings.Split(variable.Name[len(oidRX)+2:], ".")
		if len(oidParts) == 0 {
			continue
		}

		key, err := strconv.Atoi(oidParts[0])
		if err != nil {
			return err
		}

		bandwidthRX, ok := snmpx.AsInt(variable.Value)
		if !ok {
			continue
		}

		port := portMap[key]
		port.BandwidthRX = "NoLimit"
		if bandwidthRX != 0 {
			port.BandwidthRX = fmt.Sprint(bandwidthRX)
		}
		portMap[key] = port
	}

	return nil
}
