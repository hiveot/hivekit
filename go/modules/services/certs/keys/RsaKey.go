package keys

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"reflect"
)

// RsaKey implements the IHiveKey interface to a RSA key.
type RsaKey struct {
	KeyBase
}

// ImportPrivate reads the key-pair from the PEM private key
// and determines its key type.
// This returns an error if the PEM is not a valid key.
func (k *RsaKey) ImportPrivate(privatePEM string) (err error) {
	// this decodes RSA, ECDSA, ED25519 or ECDH key
	rawPrivateKey, err := k.KeyBase.ImportPrivate(privatePEM)
	if err != nil {
		return err
	}
	// for rsa, ecdsa, ecdha this is a ptr, ed25519 a non-pointer key
	privKey, valid := rawPrivateKey.(*rsa.PrivateKey)
	if !valid {
		keyType := reflect.TypeOf(k.pubKey)
		return fmt.Errorf("not an RSA private key. It looks to be a '%s'", keyType)
	}
	k.privKey = privKey
	k.pubKey = &privKey.PublicKey
	return err
}

// ImportPrivateFromFile loads public/private key pair from PEM file
// and determines its key type.
func (k *RsaKey) ImportPrivateFromFile(pemPath string) (err error) {
	pemEncodedPriv, err := os.ReadFile(pemPath)
	if err != nil {
		return err
	}
	err = k.ImportPrivate(string(pemEncodedPriv))
	return err
}

// ImportPublic reads the public key from the PEM data.
// This returns an error if the PEM is not a valid public key
//
// publicPEM must contain either a PEM encoded string, or its base64 encoded content
func (k *RsaKey) ImportPublic(publicPEM string) (err error) {
	err = k.KeyBase.ImportPublic(publicPEM)
	if err != nil {
		return err
	}
	// for rsa, ecdsa, ecdha this is a ptr, ed25519 a non-pointer key
	_, valid := k.pubKey.(*rsa.PublicKey)
	if !valid {
		keyType := reflect.TypeOf(k.pubKey)
		return fmt.Errorf("not an RSA public key. It looks to be a '%s'", keyType)
	}
	return err
}

// ImportPublicFromFile loads ECDSA public key from PEM file
func (k *RsaKey) ImportPublicFromFile(pemPath string) (err error) {
	pemEncodedPub, err := os.ReadFile(pemPath)
	if err != nil {
		return err
	}
	err = k.ImportPublic(string(pemEncodedPub))
	return err
}

// Initialize generates a new key
func (k *RsaKey) Initialize() {
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err.Error())
	}
	// for rsa, ecdsa, ecdha this is a ptr, ed25519 a non-pointer key
	k.privKey = privKey
	k.pubKey = &privKey.PublicKey
}

// KeyType returns this key's type, eg rsa
func (k *RsaKey) KeyType() KeyType {
	return KeyTypeRSA
}

// Sign returns the signature of a message signed using this key
// this requires a private key to be created or imported
func (k *RsaKey) Sign(msg []byte) (signature []byte, err error) {

	// https://www.sohamkamani.com/golang/rsa-encryption/
	// Before signing, we need to hash our message
	// The hash is what we actually sign
	msgHash := sha256.Sum256(msg)
	privKey := k.privKey.(*rsa.PrivateKey)
	signature, err = rsa.SignPKCS1v15(rand.Reader, privKey, crypto.SHA256, msgHash[:])
	if err != nil {
		log.Fatalf("Error signing message: %v", err)
	}
	return signature, err
}

// Verify the signature of a message using this key's public key
// this requires a public key to be created or imported
// returns true if the signature is valid for the message
func (k *RsaKey) Verify(msg []byte, signature []byte) (valid bool) {
	msgHash := sha256.Sum256(msg)
	pubKey := k.pubKey.(*rsa.PublicKey)
	err := rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, msgHash[:], signature)
	valid = err == nil
	return valid
}

// NewRsaKey generates a RSA key with IHiveKey interface
func NewRsaKey() *RsaKey {
	k := &RsaKey{}
	k.Initialize()
	return k
}

// NewRsaKeyFromPrivate creates and initialize a RsaKey object from an existing RSA private key.
func NewRsaKeyFromPrivate(privKey *rsa.PrivateKey) *RsaKey {
	k := &RsaKey{
		KeyBase{
			privKey: privKey,
			pubKey:  &privKey.PublicKey,
		},
	}
	var _ IHiveKey = k // interface check
	return k
}
