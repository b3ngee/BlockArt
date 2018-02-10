/*

Ink Miner.

Usage:
go run ink-miner.go [server ip:port] [pubKey] [privKey]
server ip:port: server IP addr
pubKey + privKey: key pair to validate connecting art nodes
*/

package main

import (
	"./blockartlib"

	"fmt"
	"net"
	"net/rpc"
	"os"
	"time"
	// "crypto/md5"
	"crypto/ecdsa"
	"crypto/elliptic"
	// "strconv"
	"crypto/x509"
	"encoding/gob"
	"encoding/hex"
)

type MinerKey int

type MinerInfo struct {
	Address net.Addr
	Key     ecdsa.PublicKey
}

type MinerPubKey struct {
	PubKey ecdsa.PublicKey
}

type Miner struct {
	Address net.Addr
	Key     ecdsa.PublicKey
	Cli     *rpc.Client
}

type ArtNodeInfo struct {
	PubKey ecdsa.PublicKey
}

type Block struct {
	PreviousHash string
	SetOPs       []Operation
	MinerPubKey  ecdsa.PublicKey
	Nonce        uint32
}

type Operation struct {
	ShapeType     blockartlib.ShapeType
	OPSignature   string
	ArtNodePubKey ecdsa.PublicKey
}

// Keeps track of all the keys & Miner Address so miner can send it to other miners.
// var privKey ecdsa.PrivateKey
var pubKey ecdsa.PublicKey

var minerAddr net.Addr

// Keeps track of all miners that are connected to this miner. (array/slice or map???)
var connectedMiners = make(map[string]Miner)

// Keeps track of all art nodes that are connected to this miner.
var connectedArtNodeMap = make(map[string]ArtNodeInfo)

// FUNCTION CALLS

// Registers incoming Miner that wants to connect.
func (minerKey *MinerKey) RegisterMiner(minerInfo *MinerInfo, reply *MinerInfo) error {
	cli, err := rpc.Dial("tcp", minerInfo.Address.String())

	miner := Miner{Address: minerInfo.Address, Key: minerInfo.Key, Cli: cli}
	connectedMiners[miner.Address.String()] = miner

	*reply = MinerInfo{Address: minerAddr, Key: pubKey}

	return err
}

// HELPER FUNCTIONS

// Initializes the heartbeat sends message to the server (message is the public key of miner so the server will remember it).
func InitHeartbeat(cli *rpc.Client, pubKey ecdsa.PublicKey, heartBeat uint32) {
	for {
		var reply bool
		err := cli.Call("RServer.HeartBeat", pubKey, &reply)
		HandleError(err)
		time.Sleep(time.Duration(int(heartBeat)/5) * time.Millisecond)
	}
}

// Connect to the miners that the server has given.
func ConnectToMiners(addrSet []net.Addr, currentAddress net.Addr, currentPubKey ecdsa.PublicKey) {
	for i := 0; i < len(addrSet); i++ {
		// if the address is not already set, dial and register the miner
		if _, ok := connectedMiners[addrSet[i].String()]; !ok {
			cli, err := rpc.Dial("tcp", addrSet[i].String())

			var reply MinerInfo
			err = cli.Call("MinerKey.RegisterMiner", MinerInfo{Address: currentAddress, Key: currentPubKey}, &reply)
			HandleError(err)

			connectedMiners[reply.Address.String()] = Miner{Address: reply.Address, Key: reply.Key, Cli: cli}
		}

	}
}

// This is goroutine to constantly call GetNodes() from the server every few seconds
func CheckConnectedMiners(cli *rpc.Client, MinNumMinerConnections uint8) {
	for {
		var addrSet []net.Addr
		err := cli.Call("RServer.GetNodes", pubKey, &addrSet)
		HandleError(err)

		ConnectToMiners(addrSet, minerAddr, pubKey)

		fmt.Println(connectedMiners)
		time.Sleep(10 * time.Second)
	}
}

// Returns the MD5 hash as a hex string for the OP Block (prev-hash + op + op-signature + pub-key + nonce) or No-OP Block (prev-hash + pub-key + nonce).
// Nonce is the secret for this assignment, keep increasing Nonce to find a hash with correct trailing number of zeroes.
// func ComputeBlockHash(block Block) string {
// 	h := md5.New()
// 	hash := block.PreviousHash
// 	// this states if it is an op block (has set of OPs) or not
// 	if (len(block.SetOPs) > 0) {
// 		for i := 0; i < len(block.SetOPs); i++ {
// 			hash = hash + string(block.SetOPs[i].ShapeType) + block.SetOPs[i].OPSignature
// 		}
// 	}
// 	h.Write([]byte(hash + block.MinerPubKey + strconv.Itoa(int(block.Nonce))))
// 	str := hex.EncodeToString(h.Sum(nil))
// 	return str
// }

func main() {
	gob.Register(&net.TCPAddr{})
	gob.Register(&elliptic.CurveParams{})

	serverAddr := os.Args[1]

	privateKeyBytesRestored, _ := hex.DecodeString(os.Args[3])
	privKey, _ := x509.ParseECPrivateKey(privateKeyBytesRestored)

	pubKey = privKey.PublicKey

	lis, err := net.Listen("tcp", ":0")
	minerAddr = lis.Addr()

	cli, _ := rpc.Dial("tcp", serverAddr)

	minerKey := new(MinerKey)
	rpc.Register(minerKey)

	var settings blockartlib.MinerNetSettings
	err = cli.Call("RServer.Register", MinerInfo{Address: minerAddr, Key: pubKey}, &settings)
	HandleError(err)
	// fmt.Println(settings)

	go InitHeartbeat(cli, pubKey, settings.HeartBeat)

	go rpc.Accept(lis)

	var addrSet []net.Addr
	err = cli.Call("RServer.GetNodes", pubKey, &addrSet)
	HandleError(err)

	ConnectToMiners(addrSet, minerAddr, pubKey)

	go CheckConnectedMiners(cli, settings.MinNumMinerConnections)
	for {

	}
}

func HandleError(err error) {
	if err != nil {
		fmt.Println(err)
	}
}
