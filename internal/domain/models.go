package domain

type Port struct {
	Index       int
	Vlan        string
	Description string
	Mode        string
	Speed       int
	Macs        []string
	SwitchModel string
	BandwidthRX string
	BandwidthTX string
}

type MacAlias struct {
	IPAddress string `json:"IPAddress"`
	Comment   string `json:"Comment"`
}

type SwitchOID struct {
	Firmware                  string
	SystemName                string
	SN                        string
	Uptime                    string
	SaveConfig                string
	PortDesc                  string
	PortAmount                int
	Vlan                      string
	VlanUntagged              string
	Speed                     string
	BatteryCharge             string
	BatteryStatus             string
	PortMode                  string
	CPUUtilizationFiveSeconds string
	CPUUtilizationOneMinutes  string
	CPUUtilizationFiveMinutes string
	CPUTemperature            string
	TransceiverInfo           string
	BandwidthRX               string
	BandwidthTX               string
}

// ViewData matches fields used in template/index.html.
type ViewData struct {
	Ports                     map[int]Port
	SystemName                string
	SN                        string
	IP                        string
	BatteryStatus             string
	ColorStatus               string
	Firmware                  string
	BatteryCharge             int
	Uptime                    string
	Type                      string
	CanChange                 bool
	CanChangeBandwidth        bool
	CPUUtilizationFiveSeconds string
	CPUUtilizationOneSeconds  string
	CPUUtilizationFiveMinutes string
	CPUTemperature            string
}
