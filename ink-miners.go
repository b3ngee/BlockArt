/*

Ink Miner.

Usage:
go run ink-miner.go [server ip:port] [pubKey] [privKey]
server ip:port: server IP addr
pubKey + privKey: key pair to validate connecting art nodes
*/

package main

import "./blockartlib"

import "os"
import "net"
import "net/rpc"
import "time"
import "fmt"

type MinerKey int

type Register struct {
	MinerAddr string
	PubKey string
}

type MinerPubKey struct {
	PubKey string
}

type MinerInfo struct {
	MinerAddr string
	PubKey string
	Cli *rpc.Client
}

type ArtNodeInfo struct {
	PubKey string
}

// Keeps track of all miners that are connected to this miner. (array/slice or map???)
var connectedMinerMap []MinerInfo

// Keeps track of all art nodes that are connected to this miner.
var connectedArtNodeMap = make(map[string]ArtNodeInfo)


// FUNCTION CALLS

// Registers incoming Miner that wants to connect.
func (minerKey *MinerKey) RegisterMiner(minerInfo *MinerInfo, reply *string) error {
	// TODO: Add the Miner Info to the map or array.
	cli, _ := rpc.Dial("tcp", minerInfo.MinerAddr)

}


// HELPER FUNCTIONS

// Initializes the heartbeat sends message to the server (message is the public key of miner so the server will remember it).
func InitHeartbeat(cli *rpc.Client, pubKey string, heartBeat uint32) {
	for {
		var reply string
		err := cli.Call("ServerKey.Heartbeat", MinerPubKey{PubKey: pubKey}, &reply)
		HandleError(err)
		time.Sleep(heartBeat * time.Millisecond)
	}
}

// Connect to the miners that the server has given.
func ConnectToMiners(addrSet []string) {
	// TODO: Traverse through list, dial the miner's address given in list, call RegisterMiner to notify the other miner.
}

func main() {
	serverAddr := os.Args[1]
	pubKey := os.Args[2]
	privKey := os.Args[3]
	minerAddr := "127.0.0.1:0" // registers minerAddr w/ random port???

	cli, _ := rpc.Dial("tcp", serverAddr)

	minerKey := new(MinerKey)
	rpc.Register(minerKey)

	var settings blockartlib.MinerNetSettings{}
	err := cli.Call("ServerKey.Register", Register{MinerAddr: minerAddr, PubKey: pubKey}, &settings)
	HandleError(err)

	go InitHeartbeat(cli, pubKey, settings.HeartBeat)

	lis, _ := net.Listen("tcp", minerAddr)
	go rpc.Accept(lis)

	var addrSet []string
	err = cli.Call("ServerKey.GetNodes", MinerPubKey{PubKey: pubKey}, &addrSet)
	HandleError(err)

	ConnectToMiners(addrSet)
}

func HandleError(err error) {
	if (err != nil) {
		fmt.Println(err)
	}
}