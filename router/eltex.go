package router

import (
	"fmt"
	"github.com/gin-gonic/gin"
	g "github.com/gosnmp/gosnmp"
	"strconv"
	"strings"
)

func getPortsMode(portMap map[int]Port) {
	portModeOids := make([]string, 0)

	for key, _ := range portMap {
		oid := "1.3.6.1.4.1.89.48.22.1.1." + strconv.Itoa(key)
		portModeOids = append(portModeOids, oid)
	}

	result, err := g.Default.Get(portModeOids)
	if err != nil {
		fmt.Println("149: ", err)
		return
	}

	for _, variable := range result.Variables {
		oidParts := strings.Split(variable.Name, ".")

		key, err := strconv.Atoi(oidParts[len(oidParts)-1])
		if err != nil {
			fmt.Println("157: ", err)
			return
		}

		port, _ := portMap[key]

		intValue, ok := variable.Value.(int)
		if ok {
			switch intValue {
			case 1:
				port.Mode = "general"
				break
			case 2:
				port.Mode = "access"
				break
			case 3:
				port.Mode = "trunk"
				break
			case 7:
				port.Mode = "customer"
				break
			default:
				port.Mode = "unknown"
			}
		}

		portMap[key] = port
	}
}

func getBatteryStatus() (string, string) {
	result, err := g.Default.BulkWalkAll("1.3.6.1.4.1.35265.1.23.11.1.1.2")
	if err != nil {
		fmt.Println("80: ", err)
		return "Неизвестно", "black"
	}

	var batteryStatus string
	var colorStatus string
	if len(result) > 0 {
		status := result[0].Value.(int)

		switch status {
		case 1:
			batteryStatus = "Батарея заряжена"
			colorStatus = "green"
			break
		case 2:
			batteryStatus = "Батарея разряжается"
			colorStatus = "orange"
			break
		case 3:
			batteryStatus = "Низкий уровень заряда батареи"
			colorStatus = "red"
			break
		case 5:
			batteryStatus = "Батарея отсутствует"
			colorStatus = "black"
			break
		case 6:
			batteryStatus = "Авария расцепителя тока питания батареи"
			colorStatus = "red"
			break
		case 7:
			batteryStatus = "Батарея заряжается"
			colorStatus = "blue"
			break
		default:
			batteryStatus = "Неизвестно"
		}
	} else {
		batteryStatus = "Неизвестно"
		colorStatus = "black"
	}

	return batteryStatus, colorStatus
}

func getBatteryCharge() int {
	result, err := g.Default.BulkWalkAll("1.3.6.1.4.1.35265.1.23.11.1.1.3")
	if err != nil {
		fmt.Println("133: ", err)
		return 255
	}

	if len(result) > 0 {
		value := result[0].Value.(int)
		return value
	}

	return 255
}

func getEltexPortsVlan(portMap map[int]Port, oid string, step int) error {
	result, err := g.Default.BulkWalkAll(oid) // Get() accepts up to g.MAX_OIDS
	if err != nil {
		fmt.Println("254: ", err)
		return err
	}

	for i, variable := range result {
		oidParts := strings.Split(variable.Name, ".")
		key, err := strconv.Atoi(oidParts[len(oidParts)-1])
		if err != nil {
			fmt.Println("260: ", err)
			return err
		}

		if key > 108 {
			return nil
		}

		port, _ := portMap[key]
		var vlans []int

		if port.Index == 0 {
			port.Index = i + 1
		}

		fieldArr, ok := variable.Value.([]byte)
		if ok {
			var vlan = 256 * (step - 1) * 4

			for _, item := range fieldArr {
				if item == 0 {
					vlan += 8
				} else if item < 16 {

					vlan += 4
					field := strconv.FormatInt(int64(item), 2)
					vlan += 4 - len(field)
					for _, char := range field {
						vlan++
						if char == '1' {
							vlans = append(vlans, vlan)
						}
					}
				} else {
					hexString := strconv.FormatInt(int64(item), 16)
					for _, el := range strings.Split(hexString, "") {
						if el != "0" {
							field, err := hexToBinary(el)
							if err != nil {
								fmt.Println("299: ", err)
								return err
							}

							vlan += 4 - len(field)
							for _, char := range field {
								vlan++
								if char == '1' {
									vlans = append(vlans, vlan)
								}
							}
						} else {
							vlan += 4
						}
					}
				}
			}

			if step > 1 && len(vlans) > 0 && len(port.Vlan) > 0 {
				port.Vlan += ", "
			}

			port.Vlan += formatRanges(vlans)
		}
		portMap[key] = port
	}

	return nil
}

func handlerGetEltex(c *gin.Context) {
	ip := c.Param("ip")
	g.Default.Target = ip
	g.Default.Community = "eltexstat"

	fmt.Printf("start snmp eltex %s \n", ip)

	err := g.Default.Connect()
	if err != nil {
		fmt.Println("44: ", err)
		return
	}
	defer g.Default.Conn.Close()

	portMap := make(map[int]Port)
	_switch := switches["Eltex"]

	for i := 1; i < 5; i++ {
		oid := "1.3.6.1.4.1.89.48.68.1." + strconv.Itoa(i)
		err = getEltexPortsVlan(portMap, oid, i)
		if err != nil {
			c.HTML(200, "error", nil)
			return
		}
	}

	getPortsDescription(portMap, _switch, "")

	getPortsSpeed(portMap)

	getPortsMode(portMap)

	getMacAddresses(portMap, "")

	systemName := getStringValue("1.3.6.1.2.1.1.5.0")

	batteryStatus, colorStatus := getBatteryStatus()

	firmware := getStringValue("1.3.6.1.4.1.89.2.16.1.1.4.1")

	SN := getStringValue("1.3.6.1.4.1.89.53.14.1.5.1")

	batteryCharge := getBatteryCharge()

	uptime := getUptime()

	c.HTML(200, "index", gin.H{
		"Ports":         portMap,
		"SystemName":    systemName,
		"SN":            SN,
		"IP":            ip,
		"BatteryStatus": batteryStatus,
		"ColorStatus":   colorStatus,
		"Firmware":      firmware,
		"BatteryCharge": batteryCharge,
		"Uptime":        uptime,
		"Type":          "Eltex MES",
		"CanChange":     false,
	})
}
