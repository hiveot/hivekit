package keys

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"os"
	"reflect"
)

// Ed25519Key contains the ED25519 cryptographic key set for signing and authentication.
// This implements the IHiveKey interface.
type Ed25519Key struct {
	KeyBase
}

// ImportPrivateFromFile loads public/private key pair from PEM file
// and determines its key type.
func (k *Ed25519Key) ImportPrivateFromFile(pemPath string) (err error) {
	pemEncodedPriv, err := os.ReadFile(pemPath)
	if err != nil {
		return err
	}
	err = k.ImportPrivate(string(pemEncodedPriv))
	return err
}

// ImportPrivate reads the key-pair from PEM format.
// This returns an error if the PEM is not a valid key.
func (k *Ed25519Key) ImportPrivate(privatePEM string) (err error) {
	// try PKCS8 encoding
	rawPrivateKey, err := k.KeyBase.ImportPrivate(privatePEM)
	if err == nil {
		// for rsa, ecdsa, ecdha this is a ptr, ed25519 a non-pointer key
		ed25519PK, found := rawPrivateKey.(ed25519.PrivateKey)
		if found {
			privKey := ed25519PK
			pubKey := privKey.Public().(ed25519.PublicKey)
			k.privKey = privKey
			k.pubKey = pubKey
			return nil
		}
	}
	// not a valid ed25519 key
	// is it an ed25519 seed?
	derBytes, err := ImportDer(privatePEM)
	if len(derBytes) != ed25519.SeedSize {
		err = fmt.Errorf("not a ED25519 seed")
		return err
	}
	privKey := ed25519.NewKeyFromSeed(derBytes)
	pubKey := privKey.Public().(ed25519.PublicKey)
	k.privKey = privKey
	k.pubKey = pubKey
	return err
}

// ImportPublic reads the public key from the PEM data.
// This returns an error if the PEM is not a valid public key
//
// publicPEM must contain either a PEM encoded string, or its base64 encoded content
func (k *Ed25519Key) ImportPublic(publicPEM string) (err error) {

	err = k.KeyBase.ImportPublic(publicPEM)

	if err != nil {
		return err
	}
	// for rsa, ecdsa, ecdha this is a ptr, ed25519 a non-pointer key
	_, valid := k.pubKey.(ed25519.PublicKey)
	if !valid {
		keyType := reflect.TypeOf(k.pubKey)
		return fmt.Errorf("not an ED25519 public key. It looks to be a '%s'", keyType)
	}
	return nil
}

// ImportPublicFromFile loads ED25519 public key from PEM file
func (k *Ed25519Key) ImportPublicFromFile(pemPath string) (err error) {
	pemEncodedPub, err := os.ReadFile(pemPath)
	if err != nil {
		return err
	}
	err = k.ImportPublic(string(pemEncodedPub))
	return err
}

// Initialize generates a new key
// This panics if a key could not be generated
func (k *Ed25519Key) Initialize() {
	var err error
	// for rsa, ecdsa, ecdha this is a ptr, ed25519 a non-pointer key
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	k.pubKey = pubKey
	k.privKey = privKey
	if err != nil {
		panic(err.Error())
	}
}

// KeyType returns this key's type, eg rsa
func (k *Ed25519Key) KeyType() KeyType {
	return KeyTypeEd25519
}

// PrivateKey returns the native private key pointer
func (k *Ed25519Key) PrivateKey() crypto.PrivateKey {
	// NOTE: type casting is needed to work with x509 methods
	return k.privKey.(ed25519.PrivateKey)
}

// PublicKey returns the native public key pointer
func (k *Ed25519Key) PublicKey() crypto.PublicKey {
	// NOTE: type casting is needed to work with x509 methods
	return k.pubKey.(ed25519.PublicKey)
}

// Sign returns the signature of a message signed using this key
// This signs the SHA256 hash of the message
// this requires a private key to be created or imported
func (k *Ed25519Key) Sign(msg []byte) (signature []byte, err error) {
	msgHash := sha256.Sum256(msg)
	privKey := k.privKey.(ed25519.PrivateKey)
	signature = ed25519.Sign(privKey, msgHash[:])
	return signature, nil
}

// Verify the signature of a message using this key's public key.
// This verifies using the SHA256 hash of the message.
// this requires a public key to be created or imported
// returns true if the signature is valid for the message
func (k *Ed25519Key) Verify(msg []byte, signature []byte) (valid bool) {
	msgHash := sha256.Sum256(msg)
	pubKey := k.pubKey.(ed25519.PublicKey)
	valid = ed25519.Verify(pubKey, msgHash[:], signature)
	return valid
}

// NewEd25519Key creates and initialize a ED25519 key
func NewEd25519Key() *Ed25519Key {
	k := &Ed25519Key{}
	k.Initialize()
	return k
}

// NewEd25519KeyFromPrivate creates and initialize a Ed25519Key object from an
// existing private key.
func NewEd25519KeyFromPrivate(privKey ed25519.PrivateKey) *Ed25519Key {
	pubKey := privKey.Public()
	k := &Ed25519Key{
		KeyBase: KeyBase{
			privKey: privKey,
			pubKey:  (pubKey.(ed25519.PublicKey)),
		},
		// privKey: privKey,
		// pubKey:  pubKey.(ed25519.PublicKey),
	}
	var _ IHiveKey = k // interface check
	return k
}
