package pipelineapi

import "time"

// Pipeline module configuration
type PipelineConfig struct {
	EnableAgent          bool
	EnableAuthentication bool
	EnableAuthorization  bool
	EnableDiscovery      bool
	EnableDirectory      bool
	EnableDigitwin       bool
	EnableHistory        bool
	EnableHttpBasic      bool
	EnableLogging        bool
	EnableWebsocket      bool

	// certificate directory
	certsDir string

	// The clientID this pipeline uses to connect outside
	// clientID string
	// authentication token ...?
	// authToken string
	// the root directory of the configuration storage directory
	configRoot string
	// the root directory of the application storage area (subdir per module)
	storageRoot string
	// the default communication timeout for transport modules
	rpcTimeout time.Duration
}
