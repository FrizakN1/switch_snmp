package snmp

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"

	"snmp/internal/aliases"
	"snmp/internal/domain"
)

const DefaultPortDescOID = "1.3.6.1.2.1.31.1.1.1.18"

func GetUptime(s *gosnmp.GoSNMP) (string, error) {
	result, err := s.Get([]string{"1.3.6.1.2.1.1.3.0"})
	if err != nil {
		return "Неизвестно", err
	}

	if len(result.Variables) == 0 {
		return "Неизвестно", fmt.Errorf("empty uptime response")
	}

	timeTicks, ok := result.Variables[0].Value.(uint32)
	if !ok {
		return "Неизвестно", fmt.Errorf("unexpected uptime type %T", result.Variables[0].Value)
	}

	duration := time.Duration(timeTicks) * time.Millisecond * 10

	days := duration / (24 * time.Hour)
	duration -= days * (24 * time.Hour)

	hours := duration / time.Hour
	duration -= hours * time.Hour

	minutes := duration / time.Minute
	duration -= minutes * time.Minute

	seconds := duration / time.Second

	return fmt.Sprintf("%d Дней %d:%d:%d\n", days, hours, minutes, seconds), nil
}

func GetStringValue(s *gosnmp.GoSNMP, oid string) (string, error) {
	if oid == "" {
		return "#Ошибка", nil
	}
	result, err := s.Get([]string{oid})
	if err != nil {
		return "#Ошибка", err
	}

	if len(result.Variables) == 0 || result.Variables[0].Value == nil {
		return "#Ошибка", fmt.Errorf("empty string response")
	}

	bytes, ok := result.Variables[0].Value.([]byte)
	if !ok {
		return "#Ошибка", fmt.Errorf("unexpected string type %T", result.Variables[0].Value)
	}

	return string(bytes), nil
}

func GetIntValueString(s *gosnmp.GoSNMP, oid string) (string, error) {
	if oid == "" {
		return "#Ошибка", nil
	}
	result, err := s.Get([]string{oid})
	if err != nil {
		return "#Ошибка", err
	}
	if len(result.Variables) == 0 || result.Variables[0].Value == nil {
		return "#Ошибка", fmt.Errorf("empty int response")
	}
	v, ok := AsInt(result.Variables[0].Value)
	if !ok {
		return "#Ошибка", fmt.Errorf("unexpected int type %T", result.Variables[0].Value)
	}
	return fmt.Sprintf("%d", v), nil
}

func GetPortsSpeed(s *gosnmp.GoSNMP, portMap map[int]domain.Port) error {
	portSpeedOids := make([]string, 0, len(portMap))
	for key := range portMap {
		oid := "1.3.6.1.2.1.2.2.1.5." + strconv.Itoa(key)
		portSpeedOids = append(portSpeedOids, oid)
	}

	var combinedResult []gosnmp.SnmpPDU

	if len(portSpeedOids) > 25 {
		result, err := s.Get(portSpeedOids[:25])
		if err != nil {
			return err
		}

		_result, err := s.Get(portSpeedOids[25:])
		if err != nil {
			return err
		}

		combinedResult = append(result.Variables, _result.Variables...)
	} else {
		result, err := s.Get(portSpeedOids)
		if err != nil {
			return err
		}
		combinedResult = result.Variables
	}

	for _, variable := range combinedResult {
		oidParts := strings.Split(variable.Name, ".")

		key, err := strconv.Atoi(oidParts[len(oidParts)-1])
		if err != nil {
			return err
		}

		port := portMap[key]

		intValue, ok := AsInt(variable.Value)
		if ok {
			port.Speed = intValue
			if intValue != 0 {
				port.Speed = intValue / 1000000
			}
		}

		portMap[key] = port
	}

	return nil
}

func GetPortsDescription(s *gosnmp.GoSNMP, portMap map[int]domain.Port, sw domain.SwitchOID, switchModel string) error {
	if sw.PortDesc == "" {
		sw.PortDesc = DefaultPortDescOID
	}

	usedOID := sw.PortDesc
	result, err := walkPortDescription(s, usedOID, switchModel)
	if err != nil || len(result) == 0 {
		if usedOID != DefaultPortDescOID {
			usedOID = DefaultPortDescOID
			result, err = walkPortDescription(s, usedOID, switchModel)
		}
	}
	if err != nil {
		return err
	}

	for _, variable := range result {
		oidParts := strings.Split(variable.Name[len(sw.PortDesc)+2:], ".")

		key, err := strconv.Atoi(oidParts[0])
		if err != nil {
			return err
		}

		if key > 108 || (sw.PortAmount > 0 && key > sw.PortAmount) {
			return nil
		}

		bytes, ok := variable.Value.([]byte)
		if !ok {
			continue
		}

		port, exists := portMap[key]
		if exists {
			if port.Description == "" {
				port.Description = string(bytes)
			}
		} else {
			port.Description = string(bytes)
			port.Index = key
		}

		portMap[key] = port
	}

	return nil
}

func walkPortDescription(s *gosnmp.GoSNMP, oid string, switchModel string) ([]gosnmp.SnmpPDU, error) {
	if switchModel == "DGS-1100-26/ME" {
		return s.WalkAll(oid)
	}
	return s.BulkWalkAll(oid)
}

func GetMacAddresses(s *gosnmp.GoSNMP, aliasStore *aliases.Store, portMap map[int]domain.Port, switchModel string) error {
	oid := "1.3.6.1.2.1.17.7.1.2.2.1.2"

	var result []gosnmp.SnmpPDU
	var err error

	if switchModel == "DGS-1100-26/ME" {
		result, err = s.WalkAll(oid)
	} else {
		result, err = s.BulkWalkAll(oid)
	}
	if err != nil {
		return err
	}

	for _, variable := range result {
		key, ok := AsInt(variable.Value)
		if !ok || key == 0 {
			continue
		}

		port := portMap[key]

		nameParts := strings.Split(variable.Name, fmt.Sprintf(".%s", oid))
		if len(nameParts) < 2 {
			continue
		}
		macDotParts := strings.Split(nameParts[1], ".")
		if len(macDotParts) < 8 {
			continue
		}
		macElements := macDotParts[2:8]
		var mac string

		for _, el := range macElements {
			intEl, err := strconv.Atoi(el)
			if err != nil {
				continue
			}

			var hexEl string
			if intEl < 16 {
				hexEl = "0"
			}
			hexEl += strconv.FormatInt(int64(intEl), 16)

			if hexEl == "0" {
				hexEl = "00"
			}

			mac += hexEl + ":"
		}

		if len(mac) < 17 {
			continue
		}
		macKey := mac[0:17]

		var ipAddr, comment string
		if v, ok := aliasStore.Lookup(macKey); ok {
			ipAddr = v.IPAddress
			comment = v.Comment
		}

		macStr := fmt.Sprintf("%s | %s - %s", macKey, ipAddr, comment)
		port.Macs = append(port.Macs, macStr)
		portMap[key] = port
	}

	return nil
}

func FormatRanges(numbers []int) string {
	if len(numbers) == 0 {
		return ""
	}

	var result string
	var start, end int
	for i := 0; i < len(numbers); i++ {
		if i == 0 {
			start = numbers[i]
			end = numbers[i]
		} else if numbers[i] == end+1 {
			end = numbers[i]
		} else {
			if start == end {
				result += strconv.Itoa(start) + ", "
			} else {
				result += strconv.Itoa(start) + "-" + strconv.Itoa(end) + ", "
			}
			start = numbers[i]
			end = numbers[i]
		}
	}

	if start == end {
		result += strconv.Itoa(start)
	} else {
		result += strconv.Itoa(start) + "-" + strconv.Itoa(end)
	}

	return result
}

func HexToBinary(hex string) (string, error) {
	decimal, err := strconv.ParseInt(hex, 16, 64)
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(decimal, 2), nil
}
