// Package utils with key management for certificates and authn
package utils

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"
	"path"
	"reflect"
)

type KeyType string

const (
	KeyTypeECDSA   KeyType = "ecdsa"
	KeyTypeED25519 KeyType = "ed25519"
	KeyTypeRSA     KeyType = "rsa"
	KeyTypeUnknown KeyType = ""
)

// KPFileExt defines the filename extension under which public/private keys are stored
// in the keys directory.
const KPFileExt = ".key"

// PubKeyFileExt defines the filename extension under which public key is stored
// in the keys directory.
const PubKeyFileExt = ".pub"

// DetermineKeyType returns the type of key
func DetermineKeyType(encKey string) KeyType {
	var derBytes []byte
	var err error
	blockPub, _ := pem.Decode([]byte(encKey))
	if blockPub == nil {
		// Try base64 decoding. Eg PEM content
		derBytes, err = base64.StdEncoding.DecodeString(encKey)
		_ = err
		// todo: support for hex format?
	} else {
		derBytes = blockPub.Bytes
	}
	// first check the public key type
	genericPublicKey, err := x509.ParsePKIXPublicKey(derBytes)
	if err == nil {
		switch genericPublicKey.(type) {
		case *ecdsa.PublicKey:
			return KeyTypeECDSA
		case ed25519.PublicKey: // note: <-- not a pointer
			return KeyTypeED25519
		case *rsa.PublicKey:
			return KeyTypeRSA
		}
	}
	// no luck yet, check private
	// PKCS1 is RSA
	_, err = x509.ParsePKCS1PrivateKey(derBytes)
	if err == nil {
		return KeyTypeRSA
	}
	// try PKCS8 encoding
	rawPrivateKey, err := x509.ParsePKCS8PrivateKey(derBytes)
	if err == nil {
		switch rawPrivateKey.(type) {
		case *ecdsa.PrivateKey:
			return KeyTypeECDSA
		case ed25519.PrivateKey:
			return KeyTypeED25519
		case *rsa.PrivateKey:
			return KeyTypeRSA
		default:
			return KeyTypeUnknown
		}
	}
	// is it a ed25519 seed?
	if len(derBytes) == ed25519.SeedSize {
		privKey := ed25519.NewKeyFromSeed(derBytes)
		_ = privKey
		return KeyTypeED25519
	}
	return KeyTypeUnknown
}

// PemToDer extracts the DER format from the given key PEM
func PemToDer(pemString string) ([]byte, error) {
	var derBytes []byte
	var err error
	blockPub, _ := pem.Decode([]byte(pemString))
	if blockPub == nil {
		// not pem encoded. try base64
		//return fmt.Errorf("not a valid private key PEM string")
		derBytes, err = base64.StdEncoding.DecodeString(pemString)
	} else {
		derBytes = blockPub.Bytes
	}
	if err != nil {
		err = fmt.Errorf("ImportDer: key not pem or base64")
		return nil, err
	}
	return derBytes, nil
}

// LoadCreateKeyPair loads a public/private key pair from file or create it if it doesn't exist
// This will load or create a file <clientID>.key and <clientID>.pub from the keysDir.
//
//	clientID is the client to create the keys for
//	keysDir is the location of the key file
//	keyType is the type of key to create
func LoadCreateKeyPair(clientID string, keysDir string, keyType KeyType) (
	privKey crypto.PrivateKey, pubKey crypto.PublicKey, err error) {

	if keysDir == "" {
		return nil, nil, fmt.Errorf("keys directory must be provided")
	}

	keyFile := path.Join(keysDir, clientID+KPFileExt)
	pubFile := path.Join(keysDir, clientID+PubKeyFileExt)

	// load key from file
	actualKeyType, privKey, pubKey, err := LoadPrivateKey(keyFile)

	if err == nil {
		if actualKeyType != keyType {
			err = fmt.Errorf("Key requested in file '%s' of type '%s' but is of type '%s'",
				keyFile, keyType, actualKeyType)
		}
	} else {
		// no keyfile, create the key
		privKey, pubKey = NewKey(keyType)

		// save the key for future use
		err = SavePrivateKey(privKey, keyFile)
		if err == nil {
			err = SavePublicKey(pubKey, pubFile)
		}
	}

	return privKey, pubKey, err
}

// LoadPrivateKey loads a public/private key pair from file.
// This returns nil if the key type cannot be determined
//
//	keyPath is the path to the file containing the key
func LoadPrivateKey(keyPath string) (
	keyType KeyType, privKey crypto.PrivateKey, pubKey crypto.PublicKey, err error) {

	privPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return KeyTypeUnknown, nil, nil, err
	}
	keyType, privKey, pubKey, err = PrivateKeyFromPem(string(privPEM))
	_ = pubKey
	if err != nil {
		return keyType, nil, pubKey, err
	}
	return keyType, privKey, pubKey, err
}

// LoadPublicKey loads a public key from file.
// This returns nil if the key type cannot be determined
//
//	keyPath is the path to the file containing the key
func LoadPublicKey(keyPath string) (
	pubKey crypto.PublicKey, err error) {

	pubPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}
	pubKey, err = PublicKeyFromPem(string(pubPEM))

	if err != nil {
		return nil, err
	}
	return pubKey, err
}

func NewEcdsaKey() (*ecdsa.PrivateKey, *ecdsa.PublicKey) {
	curve := elliptic.P256()
	// for rsa, ecdsa, ecdha this is a ptr, ed25519 a non-pointer key
	privKey, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		panic("unable to create ECDSA key")
	}
	return privKey, &privKey.PublicKey
}

// NewEd25519Key creates a new ED25519 key
func NewEd25519Key() (ed25519.PrivateKey, ed25519.PublicKey) {
	var err error
	// for rsa, ecdsa, ecdha this is a ptr, ed25519 a non-pointer key
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	_ = pubKey
	if err != nil {
		panic(err.Error())
	}
	return privKey, privKey.Public().(ed25519.PublicKey)
}

// NewRsaKey creates a newRSA Key
func NewRsaKey() (*rsa.PrivateKey, *rsa.PublicKey) {
	var err error
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err.Error())
	}
	pubKey := privKey.Public().(*rsa.PublicKey)
	return privKey, pubKey
}

// NewKey creates a new key of the given type
func NewKey(keyType KeyType) (crypto.PrivateKey, crypto.PublicKey) {
	switch keyType {
	case KeyTypeECDSA:
		return NewEcdsaKey()
	case KeyTypeED25519:
		return NewEd25519Key()
	case KeyTypeRSA:
		return NewRsaKey()
	default:
		return nil, nil
	}
}

// PrivateKeyFromPem reads the key-pair from the PEM private key
// and determines its key type.
// This returns an error if the PEM is not a valid key.
func PrivateKeyFromPem(privatePEM string) (
	keyType KeyType, privKey crypto.PrivateKey, pubKey crypto.PublicKey, err error) {

	var rawPrivateKey crypto.PrivateKey
	derBytes, err := PemToDer(privatePEM)
	if err == nil {
		// rsa, ecdsa, ecdha exports a ptr, ed25519 exports non-pointer key
		rawPrivateKey, err = x509.ParsePKCS8PrivateKey(derBytes)
	}
	if err != nil {
		return KeyTypeUnknown, nil, nil, err
	}

	// for rsa, ecdsa, ecdha this is a ptr, ed25519 a non-pointer key
	ecdsaPrivKey, valid := rawPrivateKey.(*ecdsa.PrivateKey)
	if valid {
		ecdsaPubKey := &ecdsaPrivKey.PublicKey
		return KeyTypeECDSA, ecdsaPrivKey, ecdsaPubKey, nil
	}

	// try ed25519
	ed25519PrivKey, valid := rawPrivateKey.(ed25519.PrivateKey)
	if valid {
		ed25519Pub := ed25519PrivKey.Public()
		return KeyTypeED25519, ed25519PrivKey, ed25519Pub, nil
	}

	// if len(derBytes) == ed25519.SeedSize {
	// 	privKey := ed25519.NewKeyFromSeed(derBytes)
	// 	pubKey := privKey.Public().(ed25519.PublicKey)
	// 	return KeyTypeEd25519, privKey, pubKey, nil
	// }

	// try RSA
	rsaPrivKey, valid := rawPrivateKey.(*rsa.PrivateKey)
	if valid {
		pubKey := &rsaPrivKey.PublicKey
		return KeyTypeRSA, rsaPrivKey, pubKey, nil
	}

	keyTypeName := reflect.TypeOf(pubKey)
	err = fmt.Errorf("not an ECDSA, ED25519 or RSA private key. It looks to be a '%s'", keyTypeName)
	return KeyTypeUnknown, nil, nil, err
}

// PrivateKeyToPem returns the PEM encoded private key
func PrivateKeyToPem(privKey crypto.PrivateKey) string {
	var err error
	var pemEnc []byte
	var keyBytes []byte

	if privKey == nil {
		panic("missing private key")
	}
	// rsa, ecdsa, ecdha expect a ptr, ed25519 expects non-pointer key
	keyBytes, err = x509.MarshalPKCS8PrivateKey(privKey)
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

// PublicKeyFromPem reads the public key from the PEM data.
// This returns an error if the PEM is not a valid public key
func PublicKeyFromPem(publicPEM string) (pubKey crypto.PublicKey, err error) {
	derBytes, err := PemToDer(publicPEM)
	if err != nil {
		return nil, err
	}
	pubKey, err = x509.ParsePKIXPublicKey(derBytes)
	return pubKey, err
}

// PublicKeyToPem returns the PEM encoded public key if available
func PublicKeyToPem(pubKey crypto.PublicKey) (pemKey string) {
	var pemData []byte
	if pubKey == nil {
		panic("missing public key")
	}

	// rsa, ecdsa, ecdha expect a ptr, ed25519 expects non-pointer key
	x509EncodedPub, err := x509.MarshalPKIXPublicKey(pubKey)
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

// SavePublicKeyToFile saves the public key to file in PEM format.
// The file permissions are set to 0644, current user can write, rest can read.
//
//	Returns error in case the public key is invalid or file cannot be written.
func SavePublicKey(pubKey crypto.PublicKey, pemPath string) error {
	pemEncoded := PublicKeyToPem(pubKey)
	err := os.WriteFile(pemPath, []byte(pemEncoded), 0644)
	return err
}

// SavePrivateKeyToFile saves the private key to file in PEM format.
// The file permissions are set to 0400, current user only, read-write permissions.
//
//	Returns error in case the key is invalid or file cannot be written.
func SavePrivateKey(privKey crypto.PrivateKey, pemPath string) error {
	privPEM := PrivateKeyToPem(privKey)
	// remove existing key since perm 0400 doesn't allow overwriting it
	_ = os.Remove(pemPath)
	err := os.WriteFile(pemPath, []byte(privPEM), 0400)
	return err
}

// Sign returns the signature of a message signed using this key
// This signs the SHA256 hash of the message
// this requires a private key to be created or imported
func Sign(msg []byte, k crypto.PrivateKey) (signature []byte, err error) {
	msgHash := sha256.Sum256(msg)

	ed25519Priv, valid := k.(ed25519.PrivateKey)
	if valid && ed25519Priv != nil {
		signature = ed25519.Sign(ed25519Priv, msgHash[:])
		return signature, err
	}

	ecdsaPriv, valid := k.(*ecdsa.PrivateKey)
	if valid && ecdsaPriv != nil {
		signature, err = ecdsa.SignASN1(rand.Reader, ecdsaPriv, msgHash[:])
		return signature, err
	}

	rsaPriv, valid := k.(*rsa.PrivateKey)
	if valid && rsaPriv != nil {
		signature, err = rsa.SignPKCS1v15(rand.Reader, rsaPriv, crypto.SHA256, msgHash[:])
		return signature, err
	}
	return nil, fmt.Errorf("key is not an ed25519, ecdsa or RSA key")
}

// Verify the signature of a message using this key's public key.
// This verifies using the SHA256 hash of the message.
// this requires a public key to be created or imported
// returns true if the signature is valid for the message
func Verify(msg []byte, signature []byte, k crypto.PublicKey) (valid bool) {
	msgHash := sha256.Sum256(msg)

	ed25519Pub, valid := k.(ed25519.PublicKey)
	if valid && ed25519Pub != nil {
		valid = ed25519.Verify(ed25519Pub, msgHash[:], signature)
		return valid
	}
	ecdsaPub, valid := k.(*ecdsa.PublicKey)
	if valid && ecdsaPub != nil {
		valid = ecdsa.VerifyASN1(ecdsaPub, msgHash[:], signature)
		return valid
	}
	rsaPub, valid := k.(*rsa.PublicKey)
	if valid && rsaPub != nil {
		err := rsa.VerifyPKCS1v15(rsaPub, crypto.SHA256, msgHash[:], signature)
		valid = err == nil
		return valid
	}
	return false
}
