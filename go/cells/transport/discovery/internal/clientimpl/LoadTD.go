package clientimpl

import (
	"crypto/x509"
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	"github.com/hiveot/hivekit/go/api/td"
	tls_client "github.com/hiveot/hivekit/go/cells/transport/tlsclient/client"
)

// LoadTD a TD document from a URL.
//
// Intended for discovery of a thing or directory TD. This downloads the TD Json using
// the URL in the discovery record.
//
// tdURL is the download URL for the TD document
// rootCAs is the certificate pool to validate the download server.
//
// rec points to the discovery record.
//
// This returns the TD, its JSON or an error if none is found
func LoadTD(tdURL string, rootCAs *x509.CertPool) (tdoc *td.TD, tdJSON string, err error) {

	slog.Info("DownloadTD", "url", tdURL)
	parts, err := url.Parse(tdURL)
	if err != nil {
		return nil, "", err
	}
	if strings.ToLower(parts.Scheme) != "https" {
		return nil, "", fmt.Errorf("Unknown scheme '%s', only http is supported", parts.Scheme)
	}
	httpCl := tls_client.NewTLSClient(parts.Host, rootCAs)
	resp, statusCode, err := httpCl.Get(parts.Path)
	_ = statusCode
	if err != nil {
		return nil, "", fmt.Errorf("DownloadTD: download failed: %w", err)
	}
	tdJSON = string(resp)
	tdDoc, err := td.UnmarshalTD(tdJSON)
	if err != nil {
		err = fmt.Errorf("LoadTD: TD loaded from '%s' but it doesn't appear to be valid json: %w",
			parts.Host+"/"+parts.Path, err)
	}
	return tdDoc, tdJSON, err
}
