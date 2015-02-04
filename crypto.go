package debora

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/hex"
	"fmt"
)

/*
	Basic crypto ops for RSA encryption and HMAC
	All strings are hex encoded DER (encoding provided by x509 package)
*/

// Generate a new 2048 bit RSA key
func GenerateKey() (*rsa.PrivateKey, error) {
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	return k, nil
}

// Returns hex encoded DER of private and public key
func EncodeKey(k *rsa.PrivateKey) (string, string, error) {
	privbytes := x509.MarshalPKCS1PrivateKey(k)
	privHex := hex.EncodeToString(privbytes)
	pub := k.PublicKey
	pubbytes, err := x509.MarshalPKIXPublicKey(&pub)
	if err != nil {
		return "", "", err
	}
	pubHex := hex.EncodeToString(pubbytes)
	return privHex, pubHex, nil
}

// Decodes hex encoded DER private key to native type
func DecodePrivateKey(privHex string) (*rsa.PrivateKey, error) {
	privBytes, err := hex.DecodeString(privHex)
	if err != nil {
		return nil, err
	}
	k, err := x509.ParsePKCS1PrivateKey(privBytes)
	if err != nil {
		return nil, err
	}
	return k, nil

}

// Decodes hex encoded DER public key to native type
func DecodePublicKey(pubHex string) (*rsa.PublicKey, error) {
	pubBytes, err := hex.DecodeString(pubHex)
	if err != nil {
		return nil, err
	}
	p, err := x509.ParsePKIXPublicKey(pubBytes)
	if err != nil {
		return nil, err
	}
	pub, ok := p.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("Parsed public key is of improper type")
	}
	return pub, nil
}

// Takes hex encoded DER public key and attempt to encrypt msg
func Encrypt(pubHex string, msg []byte) ([]byte, error) {
	pub, err := DecodePublicKey(pubHex)
	if err != nil {
		return nil, err
	}
	cipherText, err := rsa.EncryptPKCS1v15(rand.Reader, pub, msg)
	if err != nil {
		return nil, err
	}
	return cipherText, nil
}

// Takes hex encoded DER private key and attempts to decrypt cipherText
func Decrypt(privHex string, cipherText []byte) ([]byte, error) {
	priv, err := DecodePrivateKey(privHex)
	if err != nil {
		return nil, err
	}
	plainText, err := rsa.DecryptPKCS1v15(rand.Reader, priv, cipherText)
	if err != nil {
		return nil, err
	}
	return plainText, nil
}

// Produce an hmac signature
func SignMAC(message, key []byte) []byte {
	mac := hmac.New(sha1.New, key)
	mac.Write(message)
	sig := mac.Sum(nil)
	return sig
}

// CheckMAC returns true if messageMAC is a valid HMAC tag for message given the key
func CheckMAC(message, messageMAC, key []byte) bool {
	mac := hmac.New(sha1.New, key)
	mac.Write(message)
	expectedMAC := mac.Sum(nil)
	return hmac.Equal(messageMAC, expectedMAC)
}
