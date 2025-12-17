package keys

import (
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"os"
)

// base key functions
type KeyBase struct {
	// for rsa, ecdsa, ecdha this is a ptr, ed25519 a non-pointer key
	privKey crypto.PrivateKey
	// for rsa, ecdsa, ecdha this is a ptr, ed25519 a non-pointer key
	pubKey crypto.PublicKey
}

// ExportPrivate returns the PEM encoded private key
func (k *KeyBase) ExportPrivate() string {
	var err error
	var pemEnc []byte
	var keyBytes []byte

	if k.privKey == nil {
		panic("private key not initialized")
	}
	// rsa, ecdsa, ecdha expect a ptr, ed25519 expects non-pointer key
	keyBytes, err = x509.MarshalPKCS8PrivateKey(k.privKey)
	block := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: keyBytes,
	}
	pemEnc = pem.EncodeToMemory(block)
	if err != nil {
		panic("private key can't be marshalled: " + err.Error())
	}
	return string(pemEnc)
}

// ExportPrivateToFile saves the private key set to file in PEM format.
// The file permissions are set to 0400, current user only, read-write permissions.
//
//	Returns error in case the key is invalid or file cannot be written.
func (k *KeyBase) ExportPrivateToFile(pemPath string) error {
	privPEM := k.ExportPrivate()
	// remove existing key since perm 0400 doesn't allow overwriting it
	_ = os.Remove(pemPath)
	err := os.WriteFile(pemPath, []byte(privPEM), 0400)
	return err
}

// ExportPublic returns the PEM encoded public key if available
func (k *KeyBase) ExportPublic() (pemKey string) {
	var pemData []byte
	if k.pubKey == nil {
		panic("public key not initialized")
	}

	// rsa, ecdsa, ecdha expect a ptr, ed25519 expects non-pointer key
	x509EncodedPub, err := x509.MarshalPKIXPublicKey(k.pubKey)
	if err != nil {
		panic("public key can't be marshalled")
	}
	block := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: x509EncodedPub,
	}
	pemData = pem.EncodeToMemory(block)
	if pemData == nil {
		panic("public key can't be marshalled")
	}
	return string(pemData)
}

// ExportPublicToFile saves the public key to file in PEM format.
// The file permissions are set to 0644, current user can write, rest can read.
//
//	Returns error in case the public key is invalid or file cannot be written.
func (k *KeyBase) ExportPublicToFile(pemPath string) error {
	pemEncoded := k.ExportPublic()
	err := os.WriteFile(pemPath, []byte(pemEncoded), 0644)
	return err
}

// ImportPrivate reads the key-pair from the PEM private key.
// this decodes RSA, ECDSA, ED25519 or ECDH key
// This returns an error if the PEM is not a valid key.
func (k *KeyBase) ImportPrivate(privatePEM string) (rawPrivateKey any, err error) {
	derBytes, err := ImportDer(privatePEM)
	if err == nil {
		// rsa, ecdsa, ecdha exports a ptr, ed25519 exports non-pointer key
		rawPrivateKey, err = x509.ParsePKCS8PrivateKey(derBytes)
	}
	return rawPrivateKey, err
}

// ImportPublic reads the public key from the PEM data.
// This returns an error if the PEM is not a valid public key
//
// publicPEM must contain either a PEM encoded string, or its base64 encoded content
func (k *KeyBase) ImportPublic(publicPEM string) (err error) {
	derBytes, err := ImportDer(publicPEM)
	if err == nil {
		// rsa, ecdsa, ecdha exports a ptr, ed25519 expects non-pointer key
		k.pubKey, err = x509.ParsePKIXPublicKey(derBytes)
	}
	return err
}

// PrivateKey returns the native private key
// rsa, ecdsa, ecdha exports a ptr, ed25519 exports non-pointer key
func (k *KeyBase) PrivateKey() crypto.PrivateKey {
	return k.privKey
}

// PublicKey returns the native public key
// rsa, ecdsa, ecdha exports a ptr, ed25519 exports non-pointer key
func (k *KeyBase) PublicKey() crypto.PublicKey {
	return k.pubKey
}
