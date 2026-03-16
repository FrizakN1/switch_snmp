package httptransport

import (
	"github.com/gin-gonic/gin"

	"snmp/internal/config"
	"snmp/internal/service"
)

type Server struct {
	cfg   *config.Config
	dlink *service.DlinkService
	eltex *service.EltexService
}

func NewServer(cfg *config.Config, dlink *service.DlinkService, eltex *service.EltexService) *Server {
	return &Server{cfg: cfg, dlink: dlink, eltex: eltex}
}

func (s *Server) Router() *gin.Engine {
	r := gin.Default()

	r.LoadHTMLGlob("template/*.html")
	routerSNMP := r.Group("/snmp")
	routerSNMP.Static("assets/", "assets/")

	routerSNMP.GET("/eltex/:ip", s.handleGetEltex)
	routerSNMP.POST("/eltex/:ip/transceiver-info", s.handleGetEltexTransceiverInfo)
	routerSNMP.GET("/dlink/:ip", s.handleGetDlink)
	routerSNMP.POST("/dlink/change_port_description/:ip", s.handleChangePortDescription)
	routerSNMP.POST("/dlink/change_bandwidth/:ip", s.handleChangeBandwidth)

	return r
}
