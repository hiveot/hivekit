package letsencryptimpl

import (
	"crypto"
	"crypto/x509"
	"fmt"
	"time"

	"github.com/hiveot/hivekit/go/modules/certs"
	"golang.org/x/crypto/acme/autocert"
)

type LetsEncryptProvider struct {
	autocertManager autocert.Manager
}

func (svc *LetsEncryptProvider) CreateServerCert(
	serverName string, hostname string, validity time.Duration,
	serverPubKey crypto.PublicKey) (serverCert []*x509.Certificate, err error) {

	return nil, fmt.Errorf("Not yet implemented")
}
func (svc *LetsEncryptProvider) GetServerCert(serverName string) ([]*x509.Certificate, error) {
	return nil, fmt.Errorf("Not yet implemented")
}

func (svc *LetsEncryptProvider) Refresh(serverName string) ([]*x509.Certificate, error) {
	return nil, fmt.Errorf("Not yet implemented")
}

func NewLetsEncryptProvider() *LetsEncryptProvider {
	svc := &LetsEncryptProvider{}

	var _ certs.ICertProvider = svc
	return svc
}
