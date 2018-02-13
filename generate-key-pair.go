/*

Ink Miner.

Usage:
go run ink-miner.go [server ip:port] [pubKey] [privKey]
server ip:port: server IP addr
pubKey + privKey: key pair to validate connecting art nodes
*/

package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"fmt"
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

// BELOW IS JUST A TEST FOR COMPARING PUBKEYS

// var globalPubKey ecdsa.PublicKey
// var globalPrivKey ecdsa.PrivateKey

// func main() {
// 	priv, _ := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
// 	priv2, _ := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)

// 	privateKeyBytes, _ := x509.MarshalECPrivateKey(priv)
// 	privateKeyBytes2, _ := x509.MarshalECPrivateKey(priv2)
// 	//publicKeyBytes, _ := x509.MarshalPKIXPublicKey(&priv.PublicKey)

// 	encodedPrivateKeyBytes := hex.EncodeToString(privateKeyBytes)
// 	encodedPrivateKeyBytes2 := hex.EncodeToString(privateKeyBytes2)
// 	//hex.EncodeToString(publicKeyBytes)

// 	privateKeyBytesRestored, _ := hex.DecodeString(encodedPrivateKeyBytes)
// 	privateKeyBytesRestored2, _ := hex.DecodeString(encodedPrivateKeyBytes2)
// 	privKey, _ := x509.ParseECPrivateKey(privateKeyBytesRestored)
// 	privKey2, _ := x509.ParseECPrivateKey(privateKeyBytesRestored2)

// 	globalPrivKey = *privKey
// 	globalPrivKey2 := *privKey2

// 	globalPubKey = globalPrivKey.PublicKey
// 	globalPubKey2 := globalPrivKey2.PublicKey

// 	IsPublicKeySame(globalPubKey)
// 	IsPublicKeySame(globalPubKey2)
// }

// func IsPublicKeySame(incomingPubKey ecdsa.PublicKey) bool {
// 	data := []byte("data")
// 	r, s, _ := ecdsa.Sign(rand.Reader, &globalPrivKey, data)

// 	if ecdsa.Verify(&incomingPubKey, data, r, s) {
// 		fmt.Println("This is the same")
// 		return true
// 	}

// 	fmt.Println("This is not the same")
// 	return false
// }
