package httptransport

import (
	"github.com/gin-gonic/gin"

	"snmp/internal/config"
	"snmp/internal/service"
)

type Server struct {
	cfg   *config.Config
	dlink *service.DlinkService
	cisco *service.CiscoService
	eltex *service.EltexService
	tvbs  *service.TVBSService
}

func NewServer(cfg *config.Config, dlink *service.DlinkService, cisco *service.CiscoService, eltex *service.EltexService, tvbs *service.TVBSService) *Server {
	return &Server{cfg: cfg, dlink: dlink, cisco: cisco, eltex: eltex, tvbs: tvbs}
}

func (s *Server) Router() *gin.Engine {
	r := gin.Default()

	r.LoadHTMLFiles(
		"template/index.html",
		"template/error.html",
		"template/switches.html",
		"template/tvbs.html",
	)
	routerSNMP := r.Group("/snmp")
	routerSNMP.Static("assets/", "assets/")

	routerSNMP.GET("/eltex/switches", s.handleGetEltexSwitches)
	routerSNMP.POST("/eltex/switches", s.handleAddEltexSwitches)
	routerSNMP.POST("/eltex/switches/delete", s.handleDeleteEltexSwitch)
	routerSNMP.GET("/eltex/:ip", s.handleGetEltex)
	routerSNMP.POST("/eltex/:ip/transceiver-info", s.handleGetEltexTransceiverInfo)
	routerSNMP.GET("/eltex/:ip/macs", s.handleGetEltexMacs)
	routerSNMP.GET("/dlink/:ip", s.handleGetDlink)
	routerSNMP.GET("/dlink/:ip/macs", s.handleGetDlinkMacs)
	routerSNMP.GET("/cisco/:ip", s.handleGetCisco)
	routerSNMP.GET("/cisco/:ip/macs", s.handleGetCiscoMacs)
	routerSNMP.GET("/tvbs/:ip", s.handleGetTVBS)
	routerSNMP.POST("/change_port_description/:ip", s.handleChangePortDescription)
	routerSNMP.POST("/dlink/change_port_description/:ip", s.handleChangePortDescription)
	routerSNMP.POST("/dlink/change_bandwidth/:ip", s.handleChangeBandwidth)
	routerSNMP.POST("/dlink/:ip/cable-diagnostic", s.handleGetDlinkCableDiagnostic)

	return r
}
