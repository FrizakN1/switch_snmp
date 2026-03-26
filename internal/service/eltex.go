package service

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	g "github.com/gosnmp/gosnmp"

	"snmp/internal/aliases"
	"snmp/internal/config"
	"snmp/internal/domain"
	snmpx "snmp/internal/snmp"
	"snmp/internal/switchdb"
)

type EltexService struct {
	cfg     *config.Config
	aliases *aliases.Store
}

func NewEltex(cfg *config.Config, aliases *aliases.Store) *EltexService {
	return &EltexService{cfg: cfg, aliases: aliases}
}

func (s *EltexService) Get(ip string) (*domain.ViewData, error) {
	snmp := snmpx.NewClient(ip, s.cfg.EltexReadOnlyCommunity)
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

	portMap := make(map[int]domain.Port)

	if switchModel == "MES2324FB" || switchModel == "MES2300-24" {
		for i := 1; i < 5; i++ {
			oid := "1.3.6.1.4.1.89.48.68.1." + strconv.Itoa(i)
			if err := getEltexPortsVlan(snmp, portMap, oid, i); err != nil {
				return nil, err
			}
		}
	} else {
		for i := 1; i <= sw.PortAmount; i++ {
			portMap[i] = domain.Port{Index: i}
		}

		if err := getDGSPortsVlan(snmp, portMap, sw, switchModel); err != nil {
			return nil, err
		}
		formatVlans(portMap)
	}

	_ = snmpx.GetPortsDescription(snmp, portMap, sw, "")
	_ = snmpx.GetPortsSpeed(snmp, portMap)
	_ = getPortsMode(snmp, portMap, sw.PortMode, switchModel)

	systemName, _ := snmpx.GetStringValue(snmp, "1.3.6.1.2.1.1.5.0")
	batteryStatus, colorStatus, _ := getBatteryStatus(snmp, sw.BatteryStatus, switchModel)
	firmware, _ := snmpx.GetStringValue(snmp, sw.Firmware)
	SN, _ := snmpx.GetStringValue(snmp, sw.SN)
	cpuFiveSec, _ := snmpx.GetIntValueString(snmp, sw.CPUUtilizationFiveSeconds)
	cpuOneMin, _ := snmpx.GetIntValueString(snmp, sw.CPUUtilizationOneMinutes)
	cpuFiveMin, _ := snmpx.GetIntValueString(snmp, sw.CPUUtilizationFiveMinutes)
	cpuTemp, _ := snmpx.GetIntValueString(snmp, sw.CPUTemperature)
	batteryCharge, _ := getBatteryCharge(snmp, sw.BatteryCharge)
	uptime, _ := snmpx.GetUptime(snmp)

	return &domain.ViewData{
		Ports:                     portMap,
		SystemName:                systemName,
		SN:                        SN,
		IP:                        ip,
		BatteryStatus:             batteryStatus,
		ColorStatus:               colorStatus,
		Firmware:                  firmware,
		BatteryCharge:             batteryCharge,
		Uptime:                    uptime,
		Type:                      "Eltex MES",
		CanChange:                 false,
		CPUUtilizationFiveSeconds: cpuFiveSec,
		CPUUtilizationOneSeconds:  cpuOneMin,
		CPUUtilizationFiveMinutes: cpuFiveMin,
		CPUTemperature:            cpuTemp,
	}, nil
}

type TransceiverInfo struct {
	TransceiverTransmission any
	TransceiverReception    any
}

func (s *EltexService) GetTransceiverInfo(ip string, portKey int) (*TransceiverInfo, error) {
	snmp := snmpx.NewClient(ip, s.cfg.EltexReadOnlyCommunity)
	if err := snmp.Connect(); err != nil {
		return nil, err
	}
	defer snmp.Conn.Close()

	// In the original branch this always used MES2324FB base OID.
	sw := switchdb.Switches["MES2324FB"]
	if sw.TransceiverInfo == "" {
		return &TransceiverInfo{TransceiverTransmission: "-", TransceiverReception: "-"}, nil
	}

	txStr, errTX := snmpx.GetIntValueString(snmp, fmt.Sprintf("%s.%d.8", sw.TransceiverInfo, portKey))
	rxStr, errRX := snmpx.GetIntValueString(snmp, fmt.Sprintf("%s.%d.9", sw.TransceiverInfo, portKey))

	if errTX != nil || errRX != nil || txStr == "#Ошибка" || rxStr == "#Ошибка" {
		return &TransceiverInfo{TransceiverTransmission: "-", TransceiverReception: "-"}, nil
	}

	intTX, err := strconv.Atoi(txStr)
	if err != nil {
		return nil, err
	}
	intRX, err := strconv.Atoi(rxStr)
	if err != nil {
		return nil, err
	}

	// Same formatting as the legacy branch: divide by 1000 and round to 2 decimals.
	tx := math.Round(float64(intTX)/1000*100) / 100
	rx := math.Round(float64(intRX)/1000*100) / 100

	return &TransceiverInfo{TransceiverTransmission: tx, TransceiverReception: rx}, nil
}

func (s *EltexService) GetMacs(ip string) (map[int][]string, error) {
	snmp := snmpx.NewClient(ip, s.cfg.EltexReadOnlyCommunity)
	if err := snmp.Connect(); err != nil {
		return nil, err
	}
	defer snmp.Conn.Close()

	// For Eltex we always used BulkWalkAll in the old code; passing model doesn't change behavior
	// unless it matches the special DGS-1100-26/ME case (it won't).
	portMap := make(map[int]domain.Port)
	if err := snmpx.GetMacAddresses(snmp, s.aliases, portMap, ""); err != nil {
		return nil, err
	}

	out := make(map[int][]string, len(portMap))
	for k, p := range portMap {
		out[k] = p.Macs
	}
	return out, nil
}

func getPortsMode(snmp *g.GoSNMP, portMap map[int]domain.Port, oid, switchModel string) error {
	if oid == "" {
		return nil
	}

	portModeOids := make([]string, 0, len(portMap))
	for key := range portMap {
		portModeOids = append(portModeOids, oid+"."+strconv.Itoa(key))
	}

	result, err := snmp.Get(portModeOids)
	if err != nil {
		return err
	}

	for _, variable := range result.Variables {
		oidParts := strings.Split(variable.Name, ".")
		key, err := strconv.Atoi(oidParts[len(oidParts)-1])
		if err != nil {
			return err
		}

		port := portMap[key]
		intValue, ok := snmpx.AsInt(variable.Value)
		if !ok {
			continue
		}

		if switchModel == "MES2428B" {
			switch intValue {
			case 1:
				port.Mode = "access"
			case 2:
				port.Mode = "trunk"
			case 3:
				port.Mode = "general"
			default:
				port.Mode = "unknown"
			}
		} else {
			switch intValue {
			case 1:
				port.Mode = "general"
			case 2:
				port.Mode = "access"
			case 3:
				port.Mode = "trunk"
			case 7:
				port.Mode = "customer"
			default:
				port.Mode = "unknown"
			}
		}

		portMap[key] = port
	}

	return nil
}

func getBatteryStatus(snmp *g.GoSNMP, oid, switchModel string) (string, string, error) {
	if oid == "" {
		return "Неизвестно", "black", nil
	}

	result, err := snmp.BulkWalkAll(oid)
	if err != nil {
		return "Неизвестно", "black", err
	}

	if len(result) == 0 {
		return "Неизвестно", "black", nil
	}

	status, ok := snmpx.AsInt(result[0].Value)
	if !ok {
		return "Неизвестно", "black", fmt.Errorf("unexpected battery status type %T", result[0].Value)
	}

	var batteryStatus string
	var colorStatus string
	if switchModel == "MES2428B" {
		switch status {
		case 1:
			batteryStatus = "Батарея не поддерживается"
			colorStatus = "red"
		case 2:
			batteryStatus = "Батарея не подключена"
			colorStatus = "black"
		case 3:
			batteryStatus = "Батарея заряжается"
			colorStatus = "blue"
		case 4:
			batteryStatus = "Низкий заряд батареи"
			colorStatus = "red"
		case 5:
			batteryStatus = "Батарея разряжается"
			colorStatus = "orange"
		case 6:
			batteryStatus = "Батарея подключена и заряжена"
			colorStatus = "green"
		default:
			batteryStatus = "Неизвестно"
			colorStatus = "black"
		}
	} else {
		switch status {
		case 1:
			batteryStatus = "Батарея заряжена"
			colorStatus = "green"
		case 2:
			batteryStatus = "Батарея разряжается"
			colorStatus = "orange"
		case 3:
			batteryStatus = "Низкий уровень заряда батареи"
			colorStatus = "red"
		case 5:
			batteryStatus = "Батарея отсутствует"
			colorStatus = "black"
		case 6:
			batteryStatus = "Авария расцепителя"
			colorStatus = "red"
		case 7:
			batteryStatus = "Батарея заряжается"
			colorStatus = "blue"
		default:
			batteryStatus = "Неизвестно"
			colorStatus = "black"
		}
	}

	return batteryStatus, colorStatus, nil
}

func getBatteryCharge(snmp *g.GoSNMP, oid string) (int, error) {
	if oid == "" {
		return 255, nil
	}

	result, err := snmp.BulkWalkAll(oid)
	if err != nil {
		return 255, err
	}
	if len(result) == 0 {
		return 255, nil
	}

	v, ok := snmpx.AsInt(result[0].Value)
	if !ok {
		return 255, fmt.Errorf("unexpected battery charge type %T", result[0].Value)
	}
	return v, nil
}

func getEltexPortsVlan(snmp *g.GoSNMP, portMap map[int]domain.Port, oid string, step int) error {
	result, err := snmp.BulkWalkAll(oid)
	if err != nil {
		return err
	}

	for i, variable := range result {
		oidParts := strings.Split(variable.Name, ".")
		key, err := strconv.Atoi(oidParts[len(oidParts)-1])
		if err != nil {
			return err
		}

		if key > 108 {
			return nil
		}

		port := portMap[key]
		var vlans []int

		if port.Index == 0 {
			port.Index = i + 1
		}

		fieldArr, ok := variable.Value.([]byte)
		if !ok {
			portMap[key] = port
			continue
		}

		vlan := 256 * (step - 1) * 4
		for _, item := range fieldArr {
			if item == 0 {
				vlan += 8
				continue
			}

			if item < 16 {
				vlan += 4
				field := strconv.FormatInt(int64(item), 2)
				vlan += 4 - len(field)
				for _, char := range field {
					vlan++
					if char == '1' {
						vlans = append(vlans, vlan)
					}
				}
				continue
			}

			hexString := strconv.FormatInt(int64(item), 16)
			for _, el := range strings.Split(hexString, "") {
				if el == "0" {
					vlan += 4
					continue
				}
				field, err := snmpx.HexToBinary(el)
				if err != nil {
					return err
				}
				vlan += 4 - len(field)
				for _, char := range field {
					vlan++
					if char == '1' {
						vlans = append(vlans, vlan)
					}
				}
			}
		}

		if step > 1 && len(vlans) > 0 && len(port.Vlan) > 0 {
			port.Vlan += ", "
		}
		port.Vlan += snmpx.FormatRanges(vlans)
		portMap[key] = port
	}

	return nil
}
