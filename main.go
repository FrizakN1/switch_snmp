package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	g "github.com/gosnmp/gosnmp"
	"snmp/settings"
	"strconv"
	"strings"
)

type Port struct {
	Index       int
	Vlan        string
	Description string
	Mode        string
	Speed       uint
}

func main() {
	config := settings.Load("settings/settings.json")

	router := gin.Default()

	router.LoadHTMLGlob("template/*.html")

	routerSNMP := router.Group("/snmp")

	routerSNMP.GET("/eltex/:ip", handlerGetEltex)

	_ = router.Run(config.Address + ":" + config.Port)
}

func handlerGetEltex(c *gin.Context) {
	ip := c.Param("ip")
	g.Default.Target = ip
	g.Default.Community = "eltexstat"

	fmt.Printf("start snmp %s \n", ip)

	err := g.Default.Connect()
	if err != nil {
		fmt.Println("44: ", err)
	}
	defer g.Default.Conn.Close()

	portMap := make(map[int]Port)

	for i := 1; i < 5; i++ {
		oid := "1.3.6.1.4.1.89.48.68.1." + strconv.Itoa(i)
		getPortsVlan(portMap, oid, i)
	}

	getPortsDescription(portMap)

	getPortsSpeed(portMap)

	getPortsMode(portMap)

	systemName := getSystemName()

	batteryStatus, colorStatus := getBatteryStatus()

	SN := getSN()

	c.HTML(200, "index", gin.H{
		"Ports":         portMap,
		"SystemName":    systemName,
		"SN":            SN,
		"IP":            ip,
		"BatteryStatus": batteryStatus,
		"ColorStatus":   colorStatus,
	})
}

func getBatteryStatus() (string, string) {
	result, err := g.Default.BulkWalkAll("1.3.6.1.4.1.35265.1.23.11.1.1.2")
	if err != nil {
		fmt.Println("80: ", err)
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

func getSN() string {
	result, err := g.Default.Get([]string{"1.3.6.1.4.1.89.53.14.1.5.1"})
	if err != nil {
		fmt.Println("122: ", err)
	}

	bytes := result.Variables[0].Value.([]byte)
	return string(bytes)
}

func getSystemName() string {
	result, err := g.Default.Get([]string{"1.3.6.1.2.1.1.5.0"})
	if err != nil {
		fmt.Println("132: ", err)
	}

	bytes := result.Variables[0].Value.([]byte)
	return string(bytes)
}

func getPortsMode(portMap map[int]Port) {
	portModeOids := make([]string, 0)

	for key, _ := range portMap {
		oid := "1.3.6.1.4.1.89.48.22.1.1." + strconv.Itoa(key)
		portModeOids = append(portModeOids, oid)
	}

	result, err := g.Default.Get(portModeOids)
	if err != nil {
		fmt.Println("149: ", err)
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

func getPortsSpeed(portMap map[int]Port) {
	portSpeedOids := make([]string, 0)

	for key, _ := range portMap {
		oid := "1.3.6.1.2.1.31.1.1.1.15." + strconv.Itoa(key)
		portSpeedOids = append(portSpeedOids, oid)
	}

	result, err := g.Default.Get(portSpeedOids)
	if err != nil {
		fmt.Println("197: ", err)
	}

	for _, variable := range result.Variables {
		oidParts := strings.Split(variable.Name, ".")

		key, err := strconv.Atoi(oidParts[len(oidParts)-1])
		if err != nil {
			fmt.Println("205: ", err)
			return
		}

		port, _ := portMap[key]

		intValue, ok := variable.Value.(uint)
		if ok {
			port.Speed = intValue
		}

		portMap[key] = port
	}
}

func getPortsDescription(portMap map[int]Port) {
	portDescOids := make([]string, 0)

	for key, _ := range portMap {
		oid := "1.3.6.1.2.1.31.1.1.1.18." + strconv.Itoa(key)
		portDescOids = append(portDescOids, oid)
	}

	result, err := g.Default.Get(portDescOids)
	if err != nil {
		fmt.Println("230: ", err)
	}

	for _, variable := range result.Variables {
		oidParts := strings.Split(variable.Name, ".")

		key, err := strconv.Atoi(oidParts[len(oidParts)-1])
		if err != nil {
			fmt.Println("238: ", err)
			return
		}

		port, _ := portMap[key]

		bytes := variable.Value.([]byte)
		port.Description = string(bytes)

		portMap[key] = port
	}
}

func getPortsVlan(portMap map[int]Port, oid string, step int) {
	result, err := g.Default.BulkWalkAll(oid) // Get() accepts up to g.MAX_OIDS
	if err != nil {
		fmt.Println("254: ", err)
	}

	for i, variable := range result {
		oidParts := strings.Split(variable.Name, ".")
		key, err := strconv.Atoi(oidParts[len(oidParts)-1])
		if err != nil {
			fmt.Println("260: ", err)
			return
		}

		if key > 108 {
			return
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
				} else {
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
					} else {
						hexString := strconv.FormatInt(int64(item), 16)
						for _, el := range strings.Split(hexString, "") {
							if el != "0" {
								field, err := hexToBinary(el)
								if err != nil {
									fmt.Println("299: ", err)
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
			}

			if step > 1 && len(vlans) > 0 && len(port.Vlan) > 0 {
				port.Vlan += ", "
			}

			port.Vlan += formatRanges(vlans)
		}
		portMap[key] = port
	}
}

func formatRanges(numbers []int) string {
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

func hexToBinary(hex string) (string, error) {
	decimal, err := strconv.ParseInt(hex, 16, 64)
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(decimal, 2), nil
}
