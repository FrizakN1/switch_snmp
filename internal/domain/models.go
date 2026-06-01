package domain

type Port struct {
	Index       int
	Name        string
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
	SysLocation               string
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
	PortStatus                string
	PortName                  string
	CableDistance             string
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
	CanCableDiagnostic        bool
	CPUUtilizationFiveSeconds string
	CPUUtilizationOneMinutes  string
	CPUUtilizationFiveMinutes string
	CPUTemperature            string
}

type TVBSViewData struct {
	IP           string
	SerialNumber string
	OutputPower  string
	InputPower   string
}

type SwitchSummary struct {
	IP                        string
	SystemName                string
	SysLocation               string
	Firmware                  string
	SN                        string
	BatteryStatus             string
	ColorStatus               string
	BatteryCharge             string
	Uptime                    string
	CPUTemperature            string
	CPUUtilizationFiveSeconds string
	CPUUtilizationOneMinutes  string
	CPUUtilizationFiveMinutes string
}

type SwitchListViewData struct {
	Switches []SwitchSummary
	IPInput  string
	Added    int
	Invalid  []string
}
