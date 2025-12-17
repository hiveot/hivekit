package certs

import (
	"crypto/x509"

	"github.com/hiveot/hivekit/go/modules/services/certs/keys"
)

// DefaultCertsThingID is the default thingID of the certs module.
const DefaultCertsThingID = "certs"

const DefaultCaCertName = "caCert.pem"
const DefaultCaKeyName = "caKey.pem"

// Certificate service interface
type ICertsService interface {

	// Create a server x509 certificate for a server module.
	// Local use only. nil when queried remotely.
	// TBD is there a use-case for remote services?
	CreateServerCert(moduleName string, hostname string, serverKey keys.IHiveKey) (*x509.Certificate, error)

	// Return string with the CA certificate in PEM format
	// GetCAPem() string

	// Return the server public certificate in PEM format.
	// Returns an error if no server certificate is available for the given moduleID
	//
	// moduleID is the instance ID of the server module.
	// GetServerCertPem(moduleID string) (string, error)

	// Return the x509 certificate of the current CA
	GetCACert() *x509.Certificate
}
