package service

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"

	g "github.com/gosnmp/gosnmp"

	"snmp/internal/domain"
	snmpx "snmp/internal/snmp"
)

var ciscoTrunkVlanBitmapOIDs = []struct {
	oid      string
	baseVLAN int
}{
	{oid: "1.3.6.1.4.1.9.9.46.1.6.1.1.4", baseVLAN: 1},
	{oid: "1.3.6.1.4.1.9.9.46.1.6.1.1.17", baseVLAN: 1024},
	{oid: "1.3.6.1.4.1.9.9.46.1.6.1.1.18", baseVLAN: 2048},
	{oid: "1.3.6.1.4.1.9.9.46.1.6.1.1.19", baseVLAN: 3072},
}

func getCiscoPorts(snmp *g.GoSNMP, sw domain.SwitchOID) (map[int]domain.Port, error) {
	portMap := make(map[int]domain.Port)
	modeByIfIndex := make(map[int]int)

	modeResult, err := snmp.BulkWalkAll(sw.PortMode)
	if err != nil {
		return nil, err
	}
	for _, variable := range modeResult {
		ifIndex, ok := oidLastInt(variable.Name)
		if !ok {
			continue
		}

		modeValue, ok := snmpx.AsInt(variable.Value)
		if !ok {
			continue
		}

		modeByIfIndex[ifIndex] = modeValue
		modeText := "unknown"
		switch modeValue {
		case 1:
			modeText = "trunk"
		case 2:
			modeText = "access"
		}

		portMap[ifIndex] = domain.Port{
			Index: ifIndex,
			Mode:  modeText,
		}
	}

	portNameByIfIndex := make(map[int]string, len(portMap))
	nameResult, err := snmp.BulkWalkAll(sw.PortName)
	if err == nil {
		for _, variable := range nameResult {
			ifIndex, ok := oidLastInt(variable.Name)
			if !ok {
				continue
			}
			if _, exists := portMap[ifIndex]; !exists {
				continue
			}
			if bytes, ok := variable.Value.([]byte); ok {
				portNameByIfIndex[ifIndex] = strings.TrimSpace(string(bytes))
			}
		}
	}

	if sw.PortDesc != "" {
		_ = snmpx.GetPortsDescription(snmp, portMap, sw, "")
	}

	for ifIndex, p := range portMap {
		if name, ok := portNameByIfIndex[ifIndex]; ok {
			p.Name = name
		}
		portMap[ifIndex] = p
	}

	accessPorts := make(map[int]struct{})
	untaggedResult, err := snmp.BulkWalkAll(sw.VlanUntagged)
	if err == nil {
		for _, variable := range untaggedResult {
			ifIndex, ok := oidLastInt(variable.Name)
			if !ok {
				continue
			}
			p, exists := portMap[ifIndex]
			if !exists {
				continue
			}

			vlan, ok := snmpx.AsInt(variable.Value)
			if !ok || vlan <= 0 {
				continue
			}

			p.Mode = "access"
			p.Vlan = strconv.Itoa(vlan)
			portMap[ifIndex] = p
			accessPorts[ifIndex] = struct{}{}
		}
	}

	for ifIndex, p := range portMap {
		if _, ok := accessPorts[ifIndex]; ok {
			continue
		}

		if modeByIfIndex[ifIndex] == 2 {
			p.Mode = "access"
			portMap[ifIndex] = p
			continue
		}

		vlans := make([]int, 0, 64)
		for _, item := range ciscoTrunkVlanBitmapOIDs {
			oid := fmt.Sprintf("%s.%d", item.oid, ifIndex)
			result, err := snmp.Get([]string{oid})
			if err != nil || len(result.Variables) == 0 {
				continue
			}

			v := result.Variables[0].Value
			fieldArr, ok := v.([]byte)
			if !ok {
				continue
			}

			vlans = append(vlans, decodeBitmapVlans(fieldArr, item.baseVLAN)...)
		}

		if len(vlans) > 0 {
			sort.Ints(vlans)
			p.Vlan = snmpx.FormatRanges(uniqueInts(vlans))
		}
		portMap[ifIndex] = p
	}

	_ = snmpx.GetPortsSpeed(snmp, portMap)
	_ = applyPortStatus(snmp, portMap, sw.PortStatus)

	return reorderCiscoPorts(portMap), nil
}

func applyPortStatus(snmp *g.GoSNMP, portMap map[int]domain.Port, oid string) error {
	if oid == "" {
		return nil
	}

	statusResult, err := snmp.BulkWalkAll(oid)
	if err != nil {
		return err
	}

	for _, variable := range statusResult {
		ifIndex, ok := oidLastInt(variable.Name)
		if !ok {
			continue
		}
		port, exists := portMap[ifIndex]
		if !exists {
			continue
		}

		status, ok := snmpx.AsInt(variable.Value)
		if !ok {
			continue
		}

		// ifOperStatus: 1=up, 2=down...
		if status != 1 {
			port.Speed = 0
			portMap[ifIndex] = port
		}
	}

	return nil
}

func reorderCiscoPorts(portMap map[int]domain.Port) map[int]domain.Port {
	keys := make([]int, 0, len(portMap))
	for ifIndex := range portMap {
		keys = append(keys, ifIndex)
	}

	sort.Slice(keys, func(i, j int) bool {
		li := portMap[keys[i]].Name
		lj := portMap[keys[j]].Name
		return naturalLess(li, lj)
	})

	out := make(map[int]domain.Port, len(portMap))
	for idx, ifIndex := range keys {
		out[idx+1] = portMap[ifIndex]
	}
	return out
}

func getCiscoDisplayOrder(snmp *g.GoSNMP, sw domain.SwitchOID) (map[int]int, error) {
	portMap := make(map[int]domain.Port)
	modeResult, err := snmp.BulkWalkAll(sw.PortMode)
	if err != nil {
		return nil, err
	}

	for _, variable := range modeResult {
		ifIndex, ok := oidLastInt(variable.Name)
		if !ok {
			continue
		}
		portMap[ifIndex] = domain.Port{Index: ifIndex}
	}

	nameResult, err := snmp.BulkWalkAll(sw.PortName)
	if err == nil {
		for _, variable := range nameResult {
			ifIndex, ok := oidLastInt(variable.Name)
			if !ok {
				continue
			}
			p, exists := portMap[ifIndex]
			if !exists {
				continue
			}
			if bytes, ok := variable.Value.([]byte); ok {
				p.Name = strings.TrimSpace(string(bytes))
				portMap[ifIndex] = p
			}
		}
	}

	keys := make([]int, 0, len(portMap))
	for ifIndex := range portMap {
		keys = append(keys, ifIndex)
	}

	sort.Slice(keys, func(i, j int) bool {
		return naturalLess(portMap[keys[i]].Name, portMap[keys[j]].Name)
	})

	order := make(map[int]int, len(keys))
	for idx, ifIndex := range keys {
		order[ifIndex] = idx + 1
	}
	return order, nil
}

func decodeBitmapVlans(fieldArr []byte, baseVLAN int) []int {
	vlans := make([]int, 0, len(fieldArr)*2)
	offset := 0
	for _, b := range fieldArr {
		for bit := 7; bit >= 0; bit-- {
			if (b & (1 << bit)) != 0 {
				vlan := baseVLAN + offset
				if vlan > 0 && vlan <= 4094 {
					vlans = append(vlans, vlan)
				}
			}
			offset++
		}
	}
	return vlans
}

func uniqueInts(values []int) []int {
	if len(values) == 0 {
		return values
	}
	out := values[:1]
	for i := 1; i < len(values); i++ {
		if values[i] == values[i-1] {
			continue
		}
		out = append(out, values[i])
	}
	return out
}

func oidLastInt(oid string) (int, bool) {
	parts := strings.Split(oid, ".")
	if len(parts) == 0 {
		return 0, false
	}
	v, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return 0, false
	}
	return v, true
}

func naturalLess(a, b string) bool {
	na := splitNatural(a)
	nb := splitNatural(b)
	for i := 0; i < len(na) && i < len(nb); i++ {
		pa := na[i]
		pb := nb[i]

		aNum := isDigits(pa)
		bNum := isDigits(pb)
		if aNum && bNum {
			ai, _ := strconv.Atoi(pa)
			bi, _ := strconv.Atoi(pb)
			if ai != bi {
				return ai < bi
			}
			continue
		}

		la := strings.ToLower(pa)
		lb := strings.ToLower(pb)
		if la != lb {
			return la < lb
		}
	}
	return len(na) < len(nb)
}

func splitNatural(s string) []string {
	if s == "" {
		return []string{""}
	}

	parts := make([]string, 0, len(s)/2)
	start := 0
	currentIsDigit := unicode.IsDigit(rune(s[0]))
	for i, r := range s {
		if i == 0 {
			continue
		}
		isDigit := unicode.IsDigit(r)
		if isDigit != currentIsDigit {
			parts = append(parts, s[start:i])
			start = i
			currentIsDigit = isDigit
		}
	}
	parts = append(parts, s[start:])
	return parts
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}
