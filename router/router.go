package router

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	g "github.com/gosnmp/gosnmp"
	"snmp/settings"
	"snmp/utils"
	"strconv"
	"strings"
	"time"
)

type Port struct {
	Index       int
	Vlan        string
	Description string
	Mode        string
	Speed       int
	Macs        []string
	SwitchModel string
}

type Aliases struct {
	Mac map[string]Mac `json:"aliases"`
}

type Mac struct {
	IPAddress string `json:"IPAddress"`
	Comment   string `json:"Comment"`
}

type SwitchOID struct {
	Firmware     string
	SystemName   string
	SN           string
	Uptime       string
	SaveConfig   string
	PortDesc     string
	PortAmount   int
	Vlan         string
	VlanUntagged string
	Speed        string
}

var aliases Aliases
var switches = map[string]SwitchOID{
	"DES-1210-28": {
		Firmware:     "1.3.6.1.4.1.171.10.75.5.2.1.3.0",
		SystemName:   "1.3.6.1.4.1.171.10.75.5.2.1.1.0",
		SN:           "",
		SaveConfig:   "1.3.6.1.4.1.171.10.75.5.2.1.10.0", //1
		PortDesc:     "1.3.6.1.2.1.31.1.1.1.18",
		PortAmount:   28,
		Vlan:         "1.3.6.1.4.1.171.10.75.5.2.7.6.1.2",
		VlanUntagged: "1.3.6.1.4.1.171.10.75.5.2.7.6.1.4",
		Speed:        "1.3.6.1.4.1.171.10.75.5.2.1.13.1.3",
	},
	"DES-1210-28/ME": {
		Firmware:     "1.3.6.1.4.1.171.10.75.15.1.3.0",
		SystemName:   "1.3.6.1.4.1.171.10.75.15.1.1.0",
		SN:           "",
		SaveConfig:   "1.3.6.1.4.1.171.10.75.15.1.10.0", //1
		PortDesc:     "1.3.6.1.4.1.171.10.75.15.1.14.1.3",
		PortAmount:   28,
		Vlan:         "1.3.6.1.4.1.171.10.75.15.7.6.1.2",
		VlanUntagged: "1.3.6.1.4.1.171.10.75.15.7.6.1.4",
		Speed:        "1.3.6.1.4.1.171.10.75.15.1.13.1.4",
	},
	"DES-1210-28/ME/B2": {
		Firmware:     "1.3.6.1.4.1.171.10.75.15.2.1.3.0",
		SystemName:   "1.3.6.1.4.1.171.10.75.15.2.1.1.0",
		SN:           "1.3.6.1.4.1.171.10.75.15.2.1.30.0",
		SaveConfig:   "1.3.6.1.4.1.171.10.75.15.2.1.10.0", //1
		PortDesc:     "1.3.6.1.4.1.171.10.75.15.2.1.14.1.3",
		PortAmount:   28,
		Vlan:         "1.3.6.1.4.1.171.10.75.15.2.7.6.1.3",
		VlanUntagged: "1.3.6.1.4.1.171.10.75.15.2.7.6.1.5",
		Speed:        "1.3.6.1.4.1.171.10.75.15.2.1.13.1.4",
	},
	"DGS-1100-06/ME": {
		Firmware:     "1.3.6.1.4.1.171.10.134.1.1.1.3.0",
		SystemName:   "1.3.6.1.4.1.171.10.134.1.1.1.1.0",
		SN:           "1.3.6.1.4.1.171.10.134.1.1.1.30.0",
		SaveConfig:   "1.3.6.1.4.1.171.10.134.1.1.1.10.0", //1
		PortDesc:     "1.3.6.1.4.1.171.10.134.1.1.1.14.1.3",
		PortAmount:   6,
		Vlan:         "1.3.6.1.4.1.171.10.134.1.1.7.6.1.2",
		VlanUntagged: "1.3.6.1.4.1.171.10.134.1.1.7.6.1.4",
		Speed:        "1.3.6.1.4.1.171.10.134.1.1.1.13.1.4",
	},
	"DGS-1100-10/ME": {
		Firmware:     "1.3.6.1.4.1.171.10.134.2.1.1.1.3.0",
		SystemName:   "1.3.6.1.4.1.171.10.134.2.1.1.1.1.0",
		SN:           "1.3.6.1.4.1.171.10.134.2.1.1.29.0",
		SaveConfig:   "1.3.6.1.4.1.171.10.134.2.1.1.10.0", //1
		PortDesc:     "1.3.6.1.4.1.171.10.134.2.1.1.100.2.1.3",
		PortAmount:   10,
		Vlan:         "1.3.6.1.4.1.171.10.134.2.1.7.6.1.3",
		VlanUntagged: "1.3.6.1.4.1.171.10.134.2.1.7.6.1.4",
		Speed:        "1.3.6.1.4.1.171.10.134.2.1.1.100.1.1.4",
	},
	"DGS-1210-10/C1": {
		Firmware:     "1.3.6.1.4.1.171.10.76.32.1.1.3.0",
		SystemName:   "1.3.6.1.4.1.171.10.76.32.1.1.1.0",
		SN:           "",
		SaveConfig:   "1.3.6.1.4.1.171.10.76.32.1.1.10.0", //1
		PortDesc:     "1.3.6.1.4.1.171.10.76.32.1.1.16.1.2",
		PortAmount:   10,
		Vlan:         "1.3.6.1.4.1.171.10.76.32.1.7.6.1.2",
		VlanUntagged: "1.3.6.1.4.1.171.10.76.32.1.7.6.1.4",
		Speed:        "1.3.6.1.4.1.171.10.76.32.1.1.13.1.3",
	},
	"DGS-1100-26/ME": {
		Firmware:   "1.3.6.1.4.1.171.15.3.1.4.0",
		SystemName: "1.3.6.1.2.1.1.5.0",
		SN:         "1.3.6.1.4.1.171.15.3.1.6.0",
		SaveConfig: "1.3.6.1.4.1.171.15.22.1.6.0", //1
		PortDesc:   "1.3.6.1.2.1.31.1.1.1.18",
		PortAmount: 26,
		Vlan:       "1.3.6.1.2.1.17.7.1.4.3.1.2",
		//Vlan:         "1.3.6.1.4.1.171.15.9.1.2.1.1",
		VlanUntagged: "1.3.6.1.2.1.17.7.1.4.3.1.4",
		Speed:        "1.3.6.1.4.1.171.10.76.32.1.1.13.1.3",
	},
	"DGS-1210-20/C1": {
		Firmware:     "1.3.6.1.4.1.171.10.76.19.1.1.3.0",
		SystemName:   "1.3.6.1.4.1.171.10.76.19.1.1.1.0",
		SN:           "",
		SaveConfig:   "1.3.6.1.4.1.171.10.76.19.1.1.10.0", //1
		PortDesc:     "1.3.6.1.4.1.171.10.76.19.1.1.16.1.2",
		PortAmount:   20,
		Vlan:         "1.3.6.1.4.1.171.10.76.19.1.7.6.1.2",
		VlanUntagged: "1.3.6.1.4.1.171.10.76.19.1.7.6.1.4",
		Speed:        "1.3.6.1.4.1.171.10.76.19.1.1.13.1.3",
	},
	"DGS-1210-20/F1": {
		Firmware:     "1.3.6.1.4.1.171.11.153.1000.1.3.0",
		SystemName:   "1.3.6.1.4.1.171.11.153.1000.1.1.0",
		SN:           "1.3.6.1.4.1.171.15.3.1.6.0",
		SaveConfig:   "1.3.6.1.4.1.171.11.153.1000.1.10.0", //1
		PortDesc:     "1.3.6.1.4.1.171.11.153.1000.1.16.1.2",
		PortAmount:   20,
		Vlan:         "1.3.6.1.4.1.171.11.153.1000.7.6.1.2",
		VlanUntagged: "1.3.6.1.4.1.171.11.153.1000.7.6.1.4",
		Speed:        "1.3.6.1.4.1.171.11.153.1000.1.13.1.3",
	},
	"DGS-1210-20/ME/A1": {
		Firmware:     "1.3.6.1.4.1.171.10.76.31.1.1.3.0",
		SystemName:   "1.3.6.1.4.1.171.10.76.31.1.1.1.0",
		SN:           "1.3.6.1.4.1.171.10.76.31.1.1.30.0",
		SaveConfig:   "1.3.6.1.4.1.171.10.76.31.1.1.10.0",
		PortDesc:     "1.3.6.1.4.1.171.10.76.31.1.1.14.1.3",
		PortAmount:   20,
		Vlan:         "1.3.6.1.4.1.171.10.76.31.1.7.6.1.3",
		VlanUntagged: "1.3.6.1.4.1.171.10.76.31.1.7.6.1.5",
		Speed:        "1.3.6.1.4.1.171.10.76.31.1.1.13.1.4",
	},
	"DGS-1210-28": {
		Firmware:     "1.3.6.1.4.1.171.10.76.15.1.3.0",
		SystemName:   "1.3.6.1.4.1.171.10.76.15.1.1.0",
		SN:           "",
		SaveConfig:   "1.3.6.1.4.1.171.10.76.15.1.10.0",
		PortAmount:   28,
		Vlan:         "1.3.6.1.4.1.171.10.76.15.7.6.1.2",
		VlanUntagged: "1.3.6.1.4.1.171.10.76.15.7.6.1.4",
		Speed:        "1.3.6.1.4.1.171.10.76.15.1.13.1.3",
	},

	"DGS-3120-24SC": {
		Firmware:     "1.3.6.1.4.1.171.12.11.1.9.4.1.11.1",
		SystemName:   "1.3.6.1.2.1.1.5.0",
		SN:           "1.3.6.1.4.1.171.12.11.1.9.4.1.17.1",
		PortAmount:   24,
		Vlan:         "1.3.6.1.2.1.17.7.1.4.3.1.2",
		VlanUntagged: "1.3.6.1.2.1.17.7.1.4.3.1.4",
	},

	"Eltex": { //https://eltexcm.ru/assets/docs/site/MES_configuration_and_monitoring_via_SNMP_4_0_16_5.pdf
		Firmware:   "1.3.6.1.4.1.89.2.16.1.1.4.1",
		SystemName: "1.3.6.1.2.1.1.5.0",
		SN:         "1.3.6.1.4.1.89.53.14.1.5.1",
		SaveConfig: "1.3.6.1.4.1.89.87.2.1",
		PortDesc:   "1.3.6.1.2.1.31.1.1.1.18",
	},
	//snmpset -v2c -c <community> <IP address> \
	//1.3.6.1.4.1.89.87.2.1.3.1 i {local(1)} \
	//1.3.6.1.4.1.89.87.2.1.7.1 i {runningConfig(2)} \
	//1.3.6.1.4.1.89.87.2.1.8.1 i {local(1)} \
	//1.3.6.1.4.1.89.87.2.1.12.1 i {startupConfig (3)} \
	//1.3.6.1.4.1.89.87.2.1.17.1 i {createAndGo (4)}
}

func InitializationAliases() error {
	bytes, e := utils.LoadFile("./aliases.json")
	if e != nil {
		fmt.Println(e)
		return e
	}
	e = json.Unmarshal(bytes, &aliases)
	if e != nil {
		fmt.Println(e)
		return e
	}

	return nil
}

var config *settings.Setting

func Initialization(_config *settings.Setting) *gin.Engine {
	router := gin.Default()

	config = _config

	routerSNMP := router.Group("/snmp")

	router.LoadHTMLGlob("template/*.html")
	routerSNMP.Static("assets/", "assets/")

	routerSNMP.GET("/eltex/:ip", handlerGetEltex)
	routerSNMP.GET("/dlink/:ip", handlerGetDGS)
	routerSNMP.POST("/dlink/change_port_description/:ip", handlerDGSChangePortDescription)

	return router
}

func getMacAddresses(portMap map[int]Port, switchModel string) {
	oid := "1.3.6.1.2.1.17.7.1.2.2.1.2"

	var result []g.SnmpPDU
	var err error

	if switchModel == "DGS-1100-26/ME" {
		result, err = g.Default.WalkAll(oid)
	} else {
		result, err = g.Default.BulkWalkAll(oid)
	}
	if err != nil {
		fmt.Println("230: ", err)
		return
	}

	for _, variable := range result {
		key := variable.Value.(int)
		if key != 0 {
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

				mac += hexEl + ":"
			}

			macStr := fmt.Sprintf("%s | %s - %s", mac[0:17], aliases.Mac[mac[0:17]].IPAddress, aliases.Mac[mac[0:17]].Comment)

			port.Macs = append(port.Macs, macStr)

			portMap[key] = port
		}
	}
}

func getUptime() string {
	result, err := g.Default.Get([]string{"1.3.6.1.2.1.1.3.0"})
	if err != nil {
		fmt.Println("133: ", err)
		return "Неизвестно"
	}

	if len(result.Variables) > 0 {
		timeTicks := result.Variables[0].Value.(uint32)

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

func getStringValue(oid string) string {
	result, err := g.Default.Get([]string{oid})
	if err != nil {
		fmt.Println("133: ", err)
		return "#Ошибка"
	}

	if result.Variables[0].Value != nil {
		bytes := result.Variables[0].Value.([]byte)
		return string(bytes)
	}

	return "#Ошибка"
}

func getPortsSpeed(portMap map[int]Port) {
	portSpeedOids := make([]string, 0)
	//1.3.6.1.2.1.31.1.1.1.15.
	for key, _ := range portMap {
		oid := "1.3.6.1.2.1.2.2.1.5." + strconv.Itoa(key)
		portSpeedOids = append(portSpeedOids, oid)
	}

	var combinedResult []g.SnmpPDU

	if len(portSpeedOids) > 25 {
		result, err := g.Default.Get(portSpeedOids[:25])
		if err != nil {
			fmt.Println("197: ", err)
			return
		}

		_result, err := g.Default.Get(portSpeedOids[25:])
		if err != nil {
			fmt.Println("197: ", err)
			return
		}

		combinedResult = append(result.Variables, _result.Variables...)
	} else {
		result, err := g.Default.Get(portSpeedOids)
		if err != nil {
			fmt.Println("197: ", err)
			return
		}

		combinedResult = result.Variables
	}

	for _, variable := range combinedResult {
		oidParts := strings.Split(variable.Name, ".")

		key, err := strconv.Atoi(oidParts[len(oidParts)-1])
		if err != nil {
			fmt.Println("205: ", err)
			return
		}

		port, _ := portMap[key]

		intValue, ok := variable.Value.(uint)
		if ok {
			port.Speed = int(intValue)
			if intValue != 0 {
				port.Speed = int(intValue / 1000000)
			}
		}

		portMap[key] = port
	}
}

func getPortsDescription(portMap map[int]Port, _switch SwitchOID, switchModel string) {
	var result []g.SnmpPDU
	var err error

	if switchModel == "DGS-1100-26/ME" {
		result, err = g.Default.WalkAll(_switch.PortDesc)
	} else {
		result, err = g.Default.BulkWalkAll(_switch.PortDesc)
	}
	if err != nil {
		fmt.Println("230: ", err)
		return
	}

	for _, variable := range result {
		oidParts := strings.Split(variable.Name[len(_switch.PortDesc)+2:], ".")

		key, err := strconv.Atoi(oidParts[0])
		if err != nil {
			fmt.Println("238: ", err)
			return
		}

		if key > 108 || (_switch.PortAmount > 0 && key > _switch.PortAmount) {
			return
		}

		bytes := variable.Value.([]byte)

		port, ok := portMap[key]

		if ok {
			if port.Description == "" {
				port.Description = string(bytes)
			}
		} else {
			port.Description = string(bytes)
			port.Index = key
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
