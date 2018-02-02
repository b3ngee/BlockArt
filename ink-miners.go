/*

A trivial application to illustrate how the blockartlib library can be
used from an application in project 1 for UBC CS 416 2017W2.

Usage:
go run ink-miner.go [server ip:port] [pubKey] [privKey]
server ip:port: server IP addr
pubKey + privKey: key pair to validate connecting art nodes
*/

package main

import "os"

func main() {
	serverIPAddr := os.Args[1]
	pubKey := os.Args[2]
	privKey := os.Args[3]
}