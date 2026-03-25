package internal

import (
	"crypto/tls"
	"crypto/x509"

	"github.com/hiveot/hivekit/go/modules"
	pipelineapi "github.com/hiveot/hivekit/go/modules/pipeline/api"
	"github.com/hiveot/hivekit/go/modules/transports"
)

// The pipeline service is intended for easy construction of WoT compatible
// devices and services.
//
// This pipeline factory uses the configuration to construct capabilities provided
// by selected modules. The factory ensures each module is configured and linked
// in the proper sequence.
//
// A couple of things are always needed:
// 1. CA certificate to verify the client and/or server connections
//
// If a transport server is enabled (http-basic, websocket, directory)
// 2. Server certificate
// 3. Authenticator to verify the client of incoming connection
//
// Pipelines are modules themselves. A request sent to the pipeline module is
// passed through the pipeline.
type PipelineService struct {
	modules.HiveModuleBase

	cfg pipelineapi.PipelineConfig

	caCert *x509.Certificate
	// the server certification for transport modules
	serverCert *tls.Certificate
	// when connecting a client interface using NewModuleClient

	// the http server with and modules that serve http endpoints
	httpServer transports.IHttpServer
}

// Start the pipeline
func (svc *PipelineService) Start(cfgString string) {

}

// Create a new pipeline instance
func NewPipelineService(cfg pipelineapi.PipelineConfig) *PipelineService {
	p := &PipelineService{}
	return p
}
