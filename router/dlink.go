package router

import (
	"fmt"
	"github.com/gin-gonic/gin"
	g "github.com/gosnmp/gosnmp"
	"strconv"
	"strings"
)

func formatVlans(portMap map[int]Port) {
	for key, port := range portMap {
		if len(port.Vlan) > 0 {
			vlansStringArray := strings.Split(port.Vlan[:len(port.Vlan)-1], ",")
			vlans := make([]int, 0)

			for _, vlanString := range vlansStringArray {
				vlanInt, err := strconv.Atoi(vlanString)
				if err != nil {
					fmt.Println("20: ", err)
					return
				}

				vlans = append(vlans, vlanInt)
			}

			port.Vlan = formatRanges(vlans)

			portMap[key] = port
		}
	}
}

func getDGSPortsVlan(portMap map[int]Port, _switch SwitchOID, switchModel string) {
	//oid := "1.3.6.1.2.1.17.7.1.4.3.1.2"
	oid := _switch.Vlan

	var result []g.SnmpPDU
	var err error

	if switchModel == "DGS-1100-26/ME" {
		result, err = g.Default.WalkAll(oid)
	} else {
		result, err = g.Default.BulkWalkAll(oid)
	}
	if err != nil {
		fmt.Println("47: ", err)
		return
	}

	for _, variable := range result {

		vlan := strings.Split(variable.Name, oid)[1][1:]

		fieldsArr, ok := variable.Value.([]byte)
		if ok {
			fmt.Println(vlan)
			fmt.Println(fieldsArr)

			portNumber := 0

			for _, item := range fieldsArr {
				if item == 0 {
					portNumber += 8
				} else if item < 16 {
					portNumber += 4
					field := strconv.FormatInt(int64(item), 2)
					portNumber += 4 - len(field)
					for _, char := range field {
						portNumber++

						if char == '1' && portNumber <= _switch.PortAmount {
							port, _ := portMap[portNumber]
							port.Vlan += vlan + ","
							port.Mode = "trunk"
							portMap[portNumber] = port
						}
					}
				} else {
					hexString := strconv.FormatInt(int64(item), 16)
					for _, el := range strings.Split(hexString, "") {
						if el != "0" {
							field, err := hexToBinary(el)
							if err != nil {
								fmt.Println("81: ", err)
							}

							portNumber += 4 - len(field)

							for _, char := range field {
								portNumber++

								if char == '1' && portNumber <= _switch.PortAmount {
									fmt.Println(portNumber)
									port, _ := portMap[portNumber]
									port.Vlan += vlan + ","
									port.Mode = "trunk"
									portMap[portNumber] = port
								}
							}
						} else {
							portNumber += 4
						}
					}
				}
			}
		}
	}

	//oid = "1.3.6.1.2.1.17.7.1.4.3.1.4"
	oid = _switch.VlanUntagged

	if switchModel == "DGS-1100-26/ME" {
		result, err = g.Default.WalkAll(oid)
	} else {
		result, err = g.Default.BulkWalkAll(oid)
	}
	if err != nil {
		fmt.Println("114: ", err)
		return
	}

	for _, variable := range result {
		fieldsArr, ok := variable.Value.([]byte)

		if ok {
			portNumber := 0
			fmt.Println(fieldsArr)

			for _, item := range fieldsArr {
				if item == 0 {
					portNumber += 8
				} else if item < 16 {
					portNumber += 4
					field := strconv.FormatInt(int64(item), 2)
					portNumber += 4 - len(field)
					for _, char := range field {
						portNumber++

						if char == '1' && portNumber <= _switch.PortAmount {
							port, _ := portMap[portNumber]
							port.Mode = "access"
							portMap[portNumber] = port
						}
					}
				} else {
					hexString := strconv.FormatInt(int64(item), 16)
					for _, el := range strings.Split(hexString, "") {
						if el != "0" {
							field, err := hexToBinary(el)
							if err != nil {
								fmt.Println("144: ", err)
							}

							portNumber += 4 - len(field)
							for _, char := range field {
								portNumber++

								if char == '1' && portNumber <= _switch.PortAmount {
									fmt.Println(portNumber)
									port, _ := portMap[portNumber]
									port.Mode = "access"
									portMap[portNumber] = port
								}
							}
						} else {
							portNumber += 4
						}
					}
				}
			}
		}
	}
}

func getDGSPortsSpeed(portMap map[int]Port, oid string) {
	result, err := g.Default.BulkWalkAll(oid)
	if err != nil {
		fmt.Println("171: ", err)
		return
	}

	for _, variable := range result {
		oidParts := strings.Split(variable.Name[len(oid)+2:], ".")

		key, err := strconv.Atoi(oidParts[0])
		if err != nil {
			fmt.Println("180: ", err)
			return
		}

		speedType := variable.Value.(int)

		port, _ := portMap[key]

		if port.Speed == 0 {
			switch speedType {
			case 1:
				port.Speed = 0
				break
			case 2:
				port.Speed = 1000
				break
			case 3:
				port.Speed = 100
				break
			case 4:
				port.Speed = 50
				break
			case 5:
				port.Speed = 10
				break
			case 6:
				port.Speed = 5
				break
			default:
				port.Speed = -1
			}
		}

		portMap[key] = port
	}
}

func handlerDGSChangePortDescription(c *gin.Context) {
	ip := c.Param("ip")

	var port Port
	err := c.BindJSON(&port)
	if err != nil {
		fmt.Println("BindJSON: ", err)
		return
	}

	g.Default.Target = ip
	g.Default.Community = config.ReadWriteCommunity

	err = g.Default.Connect()
	if err != nil {
		fmt.Println("44: ", err)
		return
	}
	defer g.Default.Conn.Close()

	oid := fmt.Sprintf("1.3.6.1.2.1.31.1.1.1.18.%d", port.Index)

	fmt.Println(oid)

	param := []g.SnmpPDU{{Name: oid, Value: port.Description, Type: g.OctetString}}

	_, err = g.Default.Set(param)
	if err != nil {
		fmt.Println("245: ", err)
		c.JSON(200, false)
		c.Abort()
		return
	}

	_switch := switches[port.SwitchModel]

	param = []g.SnmpPDU{{Name: _switch.SaveConfig, Value: 1, Type: g.Integer}}

	_, err = g.Default.Set(param)
	if err != nil {
		fmt.Println("257: ", err)
		c.JSON(200, false)
		c.Abort()
		return
	}

	c.JSON(200, true)
}

func getSwitchModel() string {
	result, err := g.Default.Get([]string{"1.3.6.1.2.1.1.1.0"})
	if err != nil {
		fmt.Println("133: ", err)
		return ""
	}

	bytes := result.Variables[0].Value.([]byte)

	switchData := string(bytes)

	return strings.Split(switchData, " ")[0]
}

func handlerGetDGS(c *gin.Context) {
	ip := c.Param("ip")

	g.Default.Target = ip
	g.Default.Community = config.DlinkReadOnlyCommunity

	fmt.Printf("start snmp dgs %s \n", ip)
	err := g.Default.Connect()
	if err != nil {
		fmt.Println("44: ", err)
		return
	}
	defer g.Default.Conn.Close()

	portMap := make(map[int]Port)

	switchModel := getSwitchModel()

	_switch, ok := switches[switchModel]
	if !ok {
		for key := range switches {
			if strings.Contains(switchModel, key) {
				_switch = switches[key]
				switchModel = key
			}
		}
	}

	systemName := getStringValue(_switch.SystemName)
	if systemName == "#Ошибка" && switchModel == "DES-1210-28" {
		switchModel = "DES-1210-28/ME"
		_switch = switches[switchModel]
		systemName = getStringValue(_switch.SystemName)
	}

	systemName = fmt.Sprintf("%s (%s)", getStringValue(_switch.SystemName), switchModel)

	if _switch.PortDesc == "" {
		for i := 1; i <= _switch.PortAmount; i++ {
			portMap[i] = Port{Index: i}
		}
	} else {
		getPortsDescription(portMap, _switch, switchModel)
	}

	getDGSPortsVlan(portMap, _switch, switchModel)

	formatVlans(portMap)

	if switchModel == "DGS-1100-26/ME" || switchModel == "DGS-3120-24SC" {
		getPortsSpeed(portMap)
	} else {
		getDGSPortsSpeed(portMap, _switch.Speed)
	}

	getMacAddresses(portMap, switchModel)

	firmware := getStringValue(_switch.Firmware)

	SN := "#Неизвестно"

	if _switch.SN != "" {
		SN = getStringValue(_switch.SN)
	}

	uptime := getUptime()

	c.HTML(200, "index", gin.H{
		"Ports":      portMap,
		"SystemName": systemName,
		"SN":         SN,
		"IP":         ip,
		"Firmware":   firmware,
		"Uptime":     uptime,
		"Type":       "DGS",
		"CanChange":  _switch.PortDesc != "" && _switch.SaveConfig != "",
	})
}
