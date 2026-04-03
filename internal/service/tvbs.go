package service

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/gosnmp/gosnmp"

	"snmp/internal/config"
	"snmp/internal/domain"
	snmpx "snmp/internal/snmp"
)

const (
	tvbsOutputPowerOID  = "1.3.6.1.4.1.5591.1.11.1.3.1.1.2.1.2.1"
	tvbsInputPowerOID   = "1.3.6.1.4.1.5591.1.11.1.3.1.1.4.1.2.1"
	tvbsSerialNumberOID = "1.3.6.1.2.1.47.1.1.1.1.11.1"
)

type TVBSService struct {
	cfg *config.Config
}

func NewTVBS(cfg *config.Config) *TVBSService {
	return &TVBSService{cfg: cfg}
}

func (s *TVBSService) Get(ip string) (*domain.TVBSViewData, error) {
	snmp := snmpx.NewClient(ip, s.cfg.TVBSReadOnlyCommunity)
	if err := snmp.Connect(); err != nil {
		return nil, err
	}
	defer snmp.Conn.Close()

	outputPower, err := getAnySNMPValue(snmp, tvbsOutputPowerOID)
	if err != nil {
		return nil, fmt.Errorf("get output power: %w", err)
	}
	outputPower = formatPowerToDBm(outputPower)

	inputPower, err := getAnySNMPValue(snmp, tvbsInputPowerOID)
	if err != nil {
		return nil, fmt.Errorf("get input power: %w", err)
	}
	inputPower = formatPowerToDBm(inputPower)

	serialNumber, err := getAnySNMPValue(snmp, tvbsSerialNumberOID)
	if err != nil {
		return nil, fmt.Errorf("get serial number: %w", err)
	}

	return &domain.TVBSViewData{
		IP:           ip,
		SerialNumber: serialNumber,
		OutputPower:  outputPower,
		InputPower:   inputPower,
	}, nil
}

func getAnySNMPValue(s *gosnmp.GoSNMP, oid string) (string, error) {
	result, err := s.Get([]string{oid})
	if err != nil {
		return "", err
	}
	if len(result.Variables) == 0 || result.Variables[0].Value == nil {
		return "", fmt.Errorf("empty response for oid %s", oid)
	}

	switch v := result.Variables[0].Value.(type) {
	case []byte:
		value := strings.TrimSpace(string(v))
		if value == "" {
			return "", fmt.Errorf("empty string response for oid %s", oid)
		}
		return value, nil
	case string:
		value := strings.TrimSpace(v)
		if value == "" {
			return "", fmt.Errorf("empty string response for oid %s", oid)
		}
		return value, nil
	default:
		if intValue, ok := snmpx.AsInt(v); ok {
			return strconv.Itoa(intValue), nil
		}
		return fmt.Sprint(v), nil
	}
}

func formatPowerToDBm(raw string) string {
	value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return raw
	}

	dbm := math.Round((value/10)*100) / 100
	return strconv.FormatFloat(dbm, 'f', -1, 64) + " dBm"
}
