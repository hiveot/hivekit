package internal

import (
	"crypto/x509"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"

	"github.com/hiveot/hivekit/go/api/td"
	tls_client "github.com/hiveot/hivekit/go/cells/transport/tlsclient/client"
)

// LoadTD a TD document from a URL.
//
// Intended for retrieving a TD from http server or local filesystem.
// If the tdURL starts with https://, it is downloaded using a TLS client.
// If the tdURL starts with file://, or is a valid local path it is downloaded
// from the file system.
//
// tdURL is the download URL or file path for the TD document.
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
	scheme := strings.ToLower(parts.Scheme)
	if scheme == "https" {
		httpCl := tls_client.NewTLSClient(parts.Host, rootCAs)
		resp, statusCode, err := httpCl.Get(parts.RequestURI())
		_ = statusCode
		if err != nil {
			return nil, "", fmt.Errorf("DownloadTD: download failed: %w", err)
		}
		tdJSON = string(resp)
	} else if scheme == "file" {
		raw, err := os.ReadFile(parts.Path)
		if err != nil {
			return nil, "", fmt.Errorf("DownloadTD: download failed: %w", err)
		}
		tdJSON = string(raw)
	} else {
		return nil, "", fmt.Errorf("Unknown scheme '%s', only http is supported", parts.Scheme)
	}
	tdDoc, err := td.UnmarshalTD(tdJSON)
	if err != nil {
		err = fmt.Errorf("LoadTD: TD loaded from '%s' but it doesn't appear to be valid json: %w",
			parts.Host+"/"+parts.RequestURI(), err)
	}
	return tdDoc, tdJSON, err
}
