package snmp

import (
	g "github.com/gosnmp/gosnmp"
	"time"
)

func NewClient(ip, community string) *g.GoSNMP {
	return &g.GoSNMP{
		Target:    ip,
		Port:      161,
		Community: community,
		Version:   g.Version2c,
		Timeout:   3 * time.Second,
		Retries:   1,
	}
}

func AsInt(v any) (int, bool) {
	switch t := v.(type) {
	case int:
		return t, true
	case int8:
		return int(t), true
	case int16:
		return int(t), true
	case int32:
		return int(t), true
	case int64:
		return int(t), true
	case uint:
		return int(t), true
	case uint8:
		return int(t), true
	case uint16:
		return int(t), true
	case uint32:
		return int(t), true
	case uint64:
		return int(t), true
	default:
		return 0, false
	}
}
