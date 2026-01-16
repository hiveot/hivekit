package selfsigned

// // Load a saved CA certificate and private key from file
// // This returns an error if no valid certificate is found.
// func LoadCA(caCertPath, caKeyPath string) (*x509.Certificate, keys.IHiveKey, error) {

// 	caCert, err := LoadX509CertFromPEM(caCertPath)
// 	if err != nil {
// 		// On first start there might not be a CA. Not a fatal error.
// 		slog.Warn("no valid CA certificate found", "path", caCertPath, "err", err.Error())
// 	} else {
// 		caKey, err := keys.NewKeyFromFile(caKeyPath)
// 		if err != nil {
// 			slog.Info("loaded valid CA key-pair found", "path", caCertPath, "err", err.Error())
// 		} else {
// 			// verify CA cert and key
// 		}
// 		return caCert, caKey, err
// 	}
// 	return nil, nil, err
// }

// // LoadX509CertFromPEM loads the x509 certificate from a PEM file format.
// //
// // Intended to load the CA certificate to validate server and broker.
// //
// //	pemPath is the full path to the X509 PEM file.
// func LoadX509CertFromPEM(pemPath string) (cert *x509.Certificate, err error) {
// 	pemEncoded, err := os.ReadFile(pemPath)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return X509CertFromPEM(string(pemEncoded))
// }
