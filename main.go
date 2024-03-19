package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	g "github.com/gosnmp/gosnmp"
	"snmp/settings"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Port struct {
	Index       int
	Vlan        string
	Description string
	Mode        string
	Speed       uint
	Macs        []string
}

func main() {
	config := settings.Load("settings/settings.json")

	router := gin.Default()

	router.LoadHTMLGlob("template/*.html")
	router.Static("./assets/", "assets")

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

	var wg sync.WaitGroup

	for i := 1; i < 5; i++ {
		oid := "1.3.6.1.4.1.89.48.68.1." + strconv.Itoa(i)
		getPortsVlan(portMap, oid, i)
	}

	wg.Wait()

	getPortsDescription(portMap)

	getPortsSpeed(portMap)

	getPortsMode(portMap)

	getMacAddresses(portMap)

	systemName := getStringValue("1.3.6.1.2.1.1.5.0")

	batteryStatus, colorStatus := getBatteryStatus()

	firmware := getStringValue("1.3.6.1.4.1.89.2.16.1.1.4.1")

	SN := getStringValue("1.3.6.1.4.1.89.53.14.1.5.1")

	batteryCharge := getBatteryCharge()

	uptime := getUptime()

	wg.Wait()

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
	})
}

func getMacAddresses(portMap map[int]Port) {
	oid := "1.3.6.1.2.1.17.7.1.2.2.1.2"
	result, err := g.Default.BulkWalkAll(oid)
	if err != nil {
		fmt.Println("133: ", err)
	}

	for _, variable := range result {
		key := variable.Value.(int)
		port, _ := portMap[key]

		macElements := strings.Split(strings.Split(variable.Name, fmt.Sprintf(".%s", oid))[1], ".")[2:8]
		var mac string

		for _, el := range macElements {
			intEl, err := strconv.Atoi(el)
			if err != nil {
				fmt.Println(err)
				return
			}

			var hexEl string
			if intEl < 16 {
				hexEl = "0"
			}

			hexEl += strconv.FormatInt(int64(intEl), 16)

			if hexEl == "0" {
				hexEl = "00"
			}

			mac += hexEl + "."
		}

		port.Macs = append(port.Macs, mac[0:17])

		portMap[key] = port
	}
}

func getUptime() string {
	result, err := g.Default.BulkWalkAll("1.3.6.1.2.1.1.3")
	if err != nil {
		fmt.Println("133: ", err)
	}

	if len(result) > 0 {
		timeTicks := result[0].Value.(uint32)

		duration := time.Duration(timeTicks) * time.Millisecond * 10

		days := duration / (24 * time.Hour)
		duration -= days * (24 * time.Hour)

		hours := duration / time.Hour
		duration -= hours * time.Hour

		minutes := duration / time.Minute
		duration -= minutes * time.Minute

		seconds := duration / time.Second

		return fmt.Sprintf("%d Дней %d:%d:%d\n", days, hours, minutes, seconds)
	}

	return "Неизвестно"
}

func getBatteryCharge() int {
	result, err := g.Default.BulkWalkAll("1.3.6.1.4.1.35265.1.23.11.1.1.3")
	if err != nil {
		fmt.Println("133: ", err)
	}

	if len(result) > 0 {
		value := result[0].Value.(int)
		return value
	}

	return -1
}

func getStringValue(oid string) string {
	result, err := g.Default.Get([]string{oid})
	if err != nil {
		fmt.Println("133: ", err)
	}

	if len(result.Variables) > 0 {
		bytes := result.Variables[0].Value.([]byte)
		return string(bytes)
	} else {
		return "#Ошибка"
	}
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
