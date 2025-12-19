package selfsigned

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
)

// VerifyCert verifies whether the given certificate is a valid client certificate
// This returns the certificate CN as the clientID
func VerifyCert(cert *x509.Certificate, caCert *x509.Certificate) (cn string, err error) {
	caCertPool := x509.NewCertPool()
	caCertPool.AddCert(caCert)

	opts := x509.VerifyOptions{
		Roots:     caCertPool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	if cert.Subject.CommonName == "" {
		err = fmt.Errorf("cert has no CommonName")
	}

	//if err == nil {
	//	x509Cert, err := x509.ParseCertificate(clientCert.Certificate[0])
	//}
	if err == nil {
		// FIXME: TestCertAuth: certificate specifies incompatible key usage
		// why? Is the certpool invalid? Yet the test succeeds
		_, err = cert.Verify(opts)
	}
	return cert.Subject.CommonName, err
}

// VerifyCert verifies whether the given certificate in PEM format is a valid client certificate
// This returns the certificate CN as the clientID
func VerifyPemCert(certPEM string, caCert *x509.Certificate) (string, error) {
	caCertPool := x509.NewCertPool()
	caCertPool.AddCert(caCert)

	opts := x509.VerifyOptions{
		Roots:     caCertPool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	// cert, err := X509CertFromPEM(certPEM)
	certBlock, _ := pem.Decode([]byte(certPEM))
	if certBlock == nil {
		return "", errors.New("pem.Decode failed")
	}
	cert, err := x509.ParseCertificate(certBlock.Bytes)

	if err == nil {
		if cert.Subject.CommonName == "" {
			err = fmt.Errorf("cert has no CommonName")
		}
	}
	//if err == nil {
	//	x509Cert, err := x509.ParseCertificate(clientCert.Certificate[0])
	//}
	if err == nil {
		// FIXME: TestCertAuth: certificate specifies incompatible key usage
		// why? Is the certpool invalid? Yet the test succeeds
		_, err = cert.Verify(opts)
	}
	return cert.Subject.CommonName, err
}
