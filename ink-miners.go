/*

Ink Miner.

Usage:
go run ink-miner.go [server ip:port] [pubKey] [privKey]
server ip:port: server IP addr
pubKey + privKey: key pair to validate connecting art nodes
*/

package main

import "os"
import "net/rpc"
import "time"

type Register struct {
	MinerAddr string
	PubKey string
}

type HeartBeat struct {
	PubKey string
}

// Initializes the heartbeat sends message to the server (message is the public key of miner so the server will remember it).
func InitHeartbeat(cli *rpc.Client, pubKey string) {
	for {
		cli.Call("ServerKey.Heartbeat", Heartbeat{PubKey: pubKey}, &reply)
		time.Sleep(2 * time.Second)
	}
}

func main() {
	serverAddr := os.Args[1]
	pubKey := os.Args[2]
	privKey := os.Args[3]
	minerAddr := "127.0.0.1:0" // registers minerAddr w/ random port???

	cli, _ := rpc.Dial("tcp", serverAddr)

	minerKey := new(MinerKey)
	rpc.Register(minerKey)


	var reply string
	settings, err := cli.Call("ServerKey.Register", Register{MinerAddr: minerAddr, PubKey: pubKey}, &reply)

	go InitHeartbeat(cli, pubKey)

	// TODO: Listens for other miners/artnodes
	// TODO: Connects to other the other miners with reply

}