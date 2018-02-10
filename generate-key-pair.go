/*

Ink Miner.

Usage:
go run ink-miner.go [server ip:port] [pubKey] [privKey]
server ip:port: server IP addr
pubKey + privKey: key pair to validate connecting art nodes
*/

package main

import (
	"fmt"
	"crypto/rand"
	"crypto/elliptic"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/hex"
)

func main() {
	priv, _ := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)

	privateKeyBytes, _ := x509.MarshalECPrivateKey(priv)
	publicKeyBytes, _ := x509.MarshalPKIXPublicKey(&priv.PublicKey)

	encodedPrivateKeyBytes := hex.EncodeToString(privateKeyBytes)
	encodedPublicKeyBytes := hex.EncodeToString(publicKeyBytes)

	fmt.Println("Public Key is:")
	fmt.Println(encodedPublicKeyBytes)
	fmt.Println("Private Key is:")
	fmt.Println(encodedPrivateKeyBytes)
}