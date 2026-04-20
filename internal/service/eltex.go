package service

import (
	"bufio"
	"fmt"
	"math"
	"net"
	"os"
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

func (s *EltexService) GetSwitchListViewData() (*domain.SwitchListViewData, error) {
	ips, err := s.readSwitchIPs()
	if err != nil {
		return nil, err
	}

	switches := make([]domain.SwitchSummary, 0, len(ips))
	for _, ip := range ips {
		switches = append(switches, s.GetSwitchSummary(ip))
	}

	return &domain.SwitchListViewData{
		Switches: switches,
	}, nil
}

func (s *EltexService) GetSwitchSummary(ip string) domain.SwitchSummary {
	summary := domain.SwitchSummary{
		IP:                        ip,
		SystemName:                "#Ошибка",
		SysLocation:               "#Ошибка",
		Firmware:                  "#Ошибка",
		SN:                        "#Ошибка",
		BatteryStatus:             "Неизвестно",
		ColorStatus:               "black",
		BatteryCharge:             "-",
		Uptime:                    "#Ошибка",
		CPUTemperature:            "#Ошибка",
		CPUUtilizationFiveSeconds: "#Ошибка",
	}

	snmp := snmpx.NewClient(ip, s.cfg.EltexReadOnlyCommunity)
	if err := snmp.Connect(); err != nil {
		return summary
	}
	defer snmp.Conn.Close()

	sw := switchdb.Switches["MES2324FB"]

	if firmware, err := snmpx.GetStringValue(snmp, sw.Firmware); err == nil {
		summary.Firmware = firmware
	}
	if systemName, err := snmpx.GetStringValue(snmp, sw.SystemName); err == nil {
		summary.SystemName = systemName
	}
	if sysLocation, err := snmpx.GetStringValue(snmp, sw.SysLocation); err == nil {
		summary.SysLocation = sysLocation
	}

	if sn, err := snmpx.GetStringValue(snmp, sw.SN); err == nil {
		summary.SN = sn
	}

	if batteryStatus, colorStatus, err := getBatteryStatus(snmp, sw.BatteryStatus, "MES2324FB"); err == nil {
		summary.BatteryStatus = batteryStatus
		summary.ColorStatus = colorStatus
	}

	if batteryCharge, err := getBatteryCharge(snmp, sw.BatteryCharge); err == nil && batteryCharge != 255 {
		summary.BatteryCharge = strconv.Itoa(batteryCharge)
	}

	if uptime, err := snmpx.GetUptime(snmp); err == nil {
		summary.Uptime = strings.TrimSpace(uptime)
	}

	if cpuTemp, err := snmpx.GetIntValueString(snmp, sw.CPUTemperature); err == nil {
		summary.CPUTemperature = cpuTemp
	}

	if cpuFiveSec, err := snmpx.GetIntValueString(snmp, sw.CPUUtilizationFiveSeconds); err == nil {
		summary.CPUUtilizationFiveSeconds = cpuFiveSec
	}
	if cpuOneMin, err := snmpx.GetIntValueString(snmp, sw.CPUUtilizationOneMinutes); err == nil {
		summary.CPUUtilizationOneMinutes = cpuOneMin
	}
	if cpuFiveMin, err := snmpx.GetIntValueString(snmp, sw.CPUUtilizationFiveMinutes); err == nil {
		summary.CPUUtilizationFiveMinutes = cpuFiveMin
	}

	return summary
}

func (s *EltexService) AddSwitchIPs(raw string) error {
	existing, err := s.readSwitchIPs()
	if err != nil {
		return err
	}

	known := make(map[string]struct{}, len(existing))
	for _, ip := range existing {
		known[ip] = struct{}{}
	}

	var toAppend []string
	for _, token := range splitIPs(raw) {
		ip := strings.TrimSpace(token)
		if ip == "" {
			continue
		}

		parsed := net.ParseIP(ip)
		if parsed == nil || parsed.To4() == nil {
			continue
		}

		if _, ok := known[ip]; ok {
			continue
		}

		known[ip] = struct{}{}
		toAppend = append(toAppend, ip)
	}

	if len(toAppend) == 0 {
		return nil
	}

	f, err := os.OpenFile(s.cfg.SwitchesFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return err
	}

	for i, ip := range toAppend {
		if info.Size() > 0 || i > 0 {
			if _, err = f.WriteString("\n"); err != nil {
				return err
			}
		}
		if _, err = f.WriteString(ip); err != nil {
			return err
		}
	}

	return nil
}

func (s *EltexService) DeleteSwitchIP(ip string) error {
	existing, err := s.readSwitchIPs()
	if err != nil {
		return err
	}

	filtered := make([]string, 0, len(existing))
	for _, existingIP := range existing {
		if existingIP == ip {
			continue
		}
		filtered = append(filtered, existingIP)
	}

	content := strings.Join(filtered, "\n")
	return os.WriteFile(s.cfg.SwitchesFilePath, []byte(content), 0o644)
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

	if switchModel == "MES2428B" {
		for i := 1; i <= sw.PortAmount; i++ {
			portMap[i] = domain.Port{Index: i}
		}

		if err := getDGSPortsVlan(snmp, portMap, sw, switchModel); err != nil {
			return nil, err
		}
		formatVlans(portMap)
	} else {
		for i := 1; i < 5; i++ {
			oid := "1.3.6.1.4.1.89.48.68.1." + strconv.Itoa(i)
			if err := getEltexPortsVlan(snmp, portMap, oid, i); err != nil {
				return nil, err
			}
		}
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

	canChange := switchModel != "MES2324FB"

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
		CanChange:                 canChange,
		CPUUtilizationFiveSeconds: cpuFiveSec,
		CPUUtilizationOneMinutes:  cpuOneMin,
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

func (s *EltexService) readSwitchIPs() ([]string, error) {
	f, err := os.Open(s.cfg.SwitchesFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	defer f.Close()

	var ips []string
	seen := make(map[string]struct{})
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		ip := strings.TrimSpace(scanner.Text())
		if ip == "" {
			continue
		}
		if _, ok := seen[ip]; ok {
			continue
		}
		seen[ip] = struct{}{}
		ips = append(ips, ip)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return ips, nil
}

func splitIPs(raw string) []string {
	return strings.FieldsFunc(raw, func(r rune) bool {
		switch r {
		case '\n', '\r', '\t', ' ', ',', ';':
			return true
		default:
			return false
		}
	})
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
