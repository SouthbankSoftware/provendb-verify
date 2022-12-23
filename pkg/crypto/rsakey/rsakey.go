/*
 * @Author: guiguan
 * @Date:   2019-05-15T15:36:41+10:00
 * @Last modified by:   guiguan
 * @Last modified time: 2019-05-15T16:57:51+10:00
 */

package rsakey

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
)

// ExportPrivateKeyToPEM exports a RSA private key to its PEM string format
func ExportPrivateKeyToPEM(prv *rsa.PrivateKey) []byte {
	prvBA := x509.MarshalPKCS1PrivateKey(prv)
	return pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: prvBA,
		},
	)
}

// ImportPrivateKeyFromPEM imports a RSA prviate key from its PEM string format
func ImportPrivateKeyFromPEM(prvPem []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(prvPem)
	if block == nil {
		return nil, errors.New("empty PEM block")
	}

	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

// ExportPublicKeyToPEM exports a RSA public key to its PEM string format
func ExportPublicKeyToPEM(pub *rsa.PublicKey) ([]byte, error) {
	pubBA, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return nil, err
	}
	pubPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PUBLIC KEY",
			Bytes: pubBA,
		},
	)

	return pubPEM, nil
}

// ImportPublicKeyFromPEM imports a RSA public key from its PEM string format
func ImportPublicKeyFromPEM(pubPem []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(pubPem)
	if block == nil {
		return nil, errors.New("empty PEM block")
	}

	pubI, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	pub, ok := pubI.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("invalid RSA public key")
	}

	return pub, nil
}
