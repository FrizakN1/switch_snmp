package httptransport

import (
	"log"
	"math"
	"net"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"snmp/internal/service"
)

func (s *Server) handleGetDlink(c *gin.Context) {
	ip := c.Param("ip")
	if net.ParseIP(ip) == nil {
		log.Printf("invalid ip in GET dlink ip=%q", ip)
		c.HTML(http.StatusBadRequest, "error", nil)
		return
	}

	data, err := s.dlink.Get(ip)
	if err != nil {
		log.Printf("dlink.Get failed ip=%q err=%v", ip, err)
		c.HTML(http.StatusBadGateway, "error", nil)
		return
	}

	c.HTML(http.StatusOK, "index", data)
}

func (s *Server) handleGetCisco(c *gin.Context) {
	ip := c.Param("ip")
	if net.ParseIP(ip) == nil {
		log.Printf("invalid ip in GET cisco ip=%q", ip)
		c.HTML(http.StatusBadRequest, "error", nil)
		return
	}

	data, err := s.cisco.Get(ip)
	if err != nil {
		log.Printf("cisco.Get failed ip=%q err=%v", ip, err)
		c.HTML(http.StatusBadGateway, "error", nil)
		return
	}

	c.HTML(http.StatusOK, "index", data)
}

func (s *Server) handleGetTVBS(c *gin.Context) {
	ip := c.Param("ip")
	if net.ParseIP(ip) == nil {
		log.Printf("invalid ip in GET tvbs ip=%q", ip)
		c.HTML(http.StatusBadRequest, "error", nil)
		return
	}

	data, err := s.tvbs.Get(ip)
	if err != nil {
		log.Printf("tvbs.Get failed ip=%q err=%v", ip, err)
		c.HTML(http.StatusBadGateway, "error", nil)
		return
	}

	c.HTML(http.StatusOK, "tvbs", data)
}

func (s *Server) handleGetDlinkMacs(c *gin.Context) {
	ip := c.Param("ip")
	if net.ParseIP(ip) == nil {
		log.Printf("invalid ip in GET dlink macs ip=%q", ip)
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "invalid ip"})
		return
	}

	ports, err := s.dlink.GetMacs(ip)
	if err != nil {
		log.Printf("dlink.GetMacs failed ip=%q err=%v", ip, err)
		c.JSON(http.StatusBadGateway, gin.H{"ok": false, "error": "snmp failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "ports": ports})
}

func (s *Server) handleGetCiscoMacs(c *gin.Context) {
	ip := c.Param("ip")
	if net.ParseIP(ip) == nil {
		log.Printf("invalid ip in GET cisco macs ip=%q", ip)
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "invalid ip"})
		return
	}

	ports, err := s.cisco.GetMacs(ip)
	if err != nil {
		log.Printf("cisco.GetMacs failed ip=%q err=%v", ip, err)
		c.JSON(http.StatusBadGateway, gin.H{"ok": false, "error": "snmp failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "ports": ports})
}

func (s *Server) handleGetEltex(c *gin.Context) {
	ip := c.Param("ip")
	if net.ParseIP(ip) == nil {
		log.Printf("invalid ip in GET eltex ip=%q", ip)
		c.HTML(http.StatusBadRequest, "error", nil)
		return
	}

	data, err := s.eltex.Get(ip)
	if err != nil {
		log.Printf("eltex.Get failed ip=%q err=%v", ip, err)
		c.HTML(http.StatusBadGateway, "error", nil)
		return
	}

	c.HTML(http.StatusOK, "index", data)
}

func (s *Server) handleGetEltexSwitches(c *gin.Context) {
	data, err := s.eltex.GetSwitchListViewData()
	if err != nil {
		log.Printf("eltex.GetSwitchListViewData failed err=%v", err)
		c.HTML(http.StatusBadGateway, "error", nil)
		return
	}

	c.HTML(http.StatusOK, "switches", data)
}

func (s *Server) handleAddEltexSwitches(c *gin.Context) {
	rawIPs := c.PostForm("ips")

	if err := s.eltex.AddSwitchIPs(rawIPs); err != nil {
		log.Printf("eltex.AddSwitchIPs failed err=%v", err)
		c.JSON(http.StatusBadGateway, gin.H{"ok": false, "error": "failed to add ip list"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) handleDeleteEltexSwitch(c *gin.Context) {
	ip := c.PostForm("ip")
	if net.ParseIP(ip) == nil {
		log.Printf("invalid ip in delete switch ip=%q", ip)
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "invalid ip"})
		return
	}

	if err := s.eltex.DeleteSwitchIP(ip); err != nil {
		log.Printf("eltex.DeleteSwitchIP failed ip=%q err=%v", ip, err)
		c.JSON(http.StatusBadGateway, gin.H{"ok": false, "error": "failed to delete ip"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) handleGetEltexMacs(c *gin.Context) {
	ip := c.Param("ip")
	if net.ParseIP(ip) == nil {
		log.Printf("invalid ip in GET eltex macs ip=%q", ip)
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "invalid ip"})
		return
	}

	ports, err := s.eltex.GetMacs(ip)
	if err != nil {
		log.Printf("eltex.GetMacs failed ip=%q err=%v", ip, err)
		c.JSON(http.StatusBadGateway, gin.H{"ok": false, "error": "snmp failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "ports": ports})
}

func (s *Server) handleGetEltexTransceiverInfo(c *gin.Context) {
	ip := c.Param("ip")
	if net.ParseIP(ip) == nil {
		log.Printf("invalid ip in transceiver-info ip=%q", ip)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ip"})
		return
	}

	var portKey int
	if err := c.BindJSON(&portKey); err != nil {
		log.Printf("BindJSON transceiver-info ip=%q err=%v", ip, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	if portKey <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid port key"})
		return
	}

	info, err := s.eltex.GetTransceiverInfo(ip, portKey)
	if err != nil {
		log.Printf("GetTransceiverInfo failed ip=%q port=%d err=%v", ip, portKey, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "snmp failed"})
		return
	}

	// Keep output shape compatible with the legacy JS: numbers are rounded, or "-" strings.
	switch v := info.TransceiverTransmission.(type) {
	case float64:
		info.TransceiverTransmission = math.Round(v*100) / 100
	case string:
		// ok
	default:
		info.TransceiverTransmission = "-"
	}

	switch v := info.TransceiverReception.(type) {
	case float64:
		info.TransceiverReception = math.Round(v*100) / 100
	case string:
		// ok
	default:
		info.TransceiverReception = "-"
	}

	c.JSON(http.StatusOK, gin.H{
		"TransceiverTransmission": info.TransceiverTransmission,
		"TransceiverReception":    info.TransceiverReception,
	})
}

type changePortRequest struct {
	Index       int    `json:"Index"`
	SwitchModel string `json:"SwitchModel"`
	Description string `json:"Description"`
}

type changeBandwidthRequest struct {
	Index       int    `json:"Index"`
	SwitchModel string `json:"SwitchModel"`
	BandwidthTX int    `json:"BandwidthTX"`
	BandwidthRX int    `json:"BandwidthRX"`
}

type cableDiagnosticRequest struct {
	Index       int    `json:"Index"`
	SwitchModel string `json:"SwitchModel"`
}

func (s *Server) handleChangePortDescription(c *gin.Context) {
	if token := os.Getenv("SNMP_API_TOKEN"); token != "" {
		if c.GetHeader("Authorization") != "Bearer "+token {
			log.Printf("unauthorized change_port_description ip=%q", c.Param("ip"))
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "unauthorized"})
			return
		}
	}

	ip := c.Param("ip")
	if net.ParseIP(ip) == nil {
		log.Printf("invalid ip in change_port_description ip=%q", ip)
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "invalid ip"})
		return
	}

	var req changePortRequest
	if err := c.BindJSON(&req); err != nil {
		log.Printf("BindJSON change_port_description ip=%q err=%v", ip, err)
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "invalid json"})
		return
	}
	if req.Index <= 0 || req.SwitchModel == "" {
		log.Printf("missing port fields change_port_description ip=%q index=%d model=%q", ip, req.Index, req.SwitchModel)
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "missing port fields"})
		return
	}

	err := service.ChangePortDescription(ip, s.cfg.ReadWriteCommunity, service.ChangePortDescriptionRequest{
		Index:       req.Index,
		SwitchModel: req.SwitchModel,
		Description: req.Description,
	})
	if err != nil {
		log.Printf("ChangePortDescription failed ip=%q idx=%d model=%q err=%v", ip, req.Index, req.SwitchModel, err)
		c.JSON(http.StatusBadGateway, gin.H{"ok": false, "error": "snmp set failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) handleChangeBandwidth(c *gin.Context) {
	if token := os.Getenv("SNMP_API_TOKEN"); token != "" {
		if c.GetHeader("Authorization") != "Bearer "+token {
			log.Printf("unauthorized change_bandwidth ip=%q", c.Param("ip"))
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "unauthorized"})
			return
		}
	}

	ip := c.Param("ip")
	if net.ParseIP(ip) == nil {
		log.Printf("invalid ip in change_bandwidth ip=%q", ip)
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "invalid ip"})
		return
	}

	var req changeBandwidthRequest
	if err := c.BindJSON(&req); err != nil {
		log.Printf("BindJSON change_bandwidth ip=%q err=%v", ip, err)
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "invalid json"})
		return
	}
	if req.Index <= 0 || req.SwitchModel == "" {
		log.Printf("missing port fields change_bandwidth ip=%q index=%d model=%q", ip, req.Index, req.SwitchModel)
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "missing port fields"})
		return
	}

	err := s.dlink.ChangeBandwidth(ip, service.ChangeBandwidthRequest{
		Index:       req.Index,
		SwitchModel: req.SwitchModel,
		BandwidthTX: req.BandwidthTX,
		BandwidthRX: req.BandwidthRX,
	})
	if err != nil {
		log.Printf("ChangeBandwidth failed ip=%q idx=%d model=%q tx=%d rx=%d err=%v", ip, req.Index, req.SwitchModel, req.BandwidthTX, req.BandwidthRX, err)
		c.JSON(http.StatusBadGateway, gin.H{"ok": false, "error": "snmp set failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) handleGetDlinkCableDiagnostic(c *gin.Context) {
	if token := os.Getenv("SNMP_API_TOKEN"); token != "" {
		if c.GetHeader("Authorization") != "Bearer "+token {
			log.Printf("unauthorized cable-diagnostic ip=%q", c.Param("ip"))
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "unauthorized"})
			return
		}
	}

	ip := c.Param("ip")
	if net.ParseIP(ip) == nil {
		log.Printf("invalid ip in cable-diagnostic ip=%q", ip)
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "invalid ip"})
		return
	}

	var req cableDiagnosticRequest
	if err := c.BindJSON(&req); err != nil {
		log.Printf("BindJSON cable-diagnostic ip=%q err=%v", ip, err)
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "invalid json"})
		return
	}
	if req.Index <= 0 || req.SwitchModel == "" {
		log.Printf("missing port fields cable-diagnostic ip=%q index=%d model=%q", ip, req.Index, req.SwitchModel)
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "missing port fields"})
		return
	}

	result, err := s.dlink.GetCableDiagnostic(ip, service.CableDiagnosticRequest{
		Index:       req.Index,
		SwitchModel: req.SwitchModel,
	})
	if err != nil {
		log.Printf("GetCableDiagnostic failed ip=%q idx=%d model=%q err=%v", ip, req.Index, req.SwitchModel, err)
		c.JSON(http.StatusBadGateway, gin.H{"ok": false, "error": "snmp failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "diagnostic": result})
}
