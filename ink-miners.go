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

	"os"
	"net"
	"net/rpc"
	"time"
	"fmt"
	"crypto/md5"
	"crypto/rand"
	"crypto/elliptic"
	"crypto/ecdsa"
	"encoding/hex"
	"strconv"
	"encoding/gob"
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
	MinerAddr string
	PubKey string
	Cli *rpc.Client
}

type ArtNodeInfo struct {
	PubKey string
}

type Block struct {
	PreviousHash string
	SetOPs []Operation
	MinerPubKey string
	Nonce uint32
}

type Operation struct {
	ShapeType blockartlib.ShapeType
	OPSignature string
	ArtNodePubKey string
}

// Keeps track of all miners that are connected to this miner. (array/slice or map???)
var connectedMiners []Miner

// Keeps track of all art nodes that are connected to this miner.
var connectedArtNodeMap = make(map[string]ArtNodeInfo)


// FUNCTION CALLS

// Registers incoming Miner that wants to connect.
func (minerKey *MinerKey) RegisterMiner(minerInfo *MinerInfo, reply *string) error {
	// TODO: Add the Miner Info to the map or array.
	// cli, _ := rpc.Dial("tcp", minerInfo.MinerAddr)

	return nil
}	


// HELPER FUNCTIONS

// Initializes the heartbeat sends message to the server (message is the public key of miner so the server will remember it).
func InitHeartbeat(cli *rpc.Client, pubKey ecdsa.PublicKey, heartBeat uint32) {
	for {
		var reply string
		err := cli.Call("RServer.Heartbeat", MinerPubKey{PubKey: pubKey}, &reply)
		HandleError(err)
		time.Sleep(time.Duration(heartBeat) * time.Millisecond)
	}
}

// Connect to the miners that the server has given.
func ConnectToMiners(addrSet []string) {
	// TODO: Traverse through list, dial the miner's address given in list, call RegisterMiner to notify the other miner.
}

// Returns the MD5 hash as a hex string for the OP Block (prev-hash + op + op-signature + pub-key + nonce) or No-OP Block (prev-hash + pub-key + nonce).
// Nonce is the secret for this assignment, keep increasing Nonce to find a hash with correct trailing number of zeroes.
func ComputeBlockHash(block Block) string {
	h := md5.New()
	hash := block.PreviousHash
	// this states if it is an op block (has set of OPs) or not
	if (len(block.SetOPs) > 0) {
		for i := 0; i < len(block.SetOPs); i++ {
			hash = hash + string(block.SetOPs[i].ShapeType) + block.SetOPs[i].OPSignature
		}
	}
	h.Write([]byte(hash + block.MinerPubKey + strconv.Itoa(int(block.Nonce))))
	str := hex.EncodeToString(h.Sum(nil))
	return str
}

func main() {
	gob.Register(&net.TCPAddr{})
	gob.Register(&elliptic.CurveParams{})

	serverAddr := os.Args[1]
	// pubKey := os.Args[2]
	// privKey := os.Args[3]

	priv, _ := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	pubKey := priv.PublicKey
	// privKey := *priv

	lis, err := net.Listen("tcp", ":0")
	minerAddr := lis.Addr()

	cli, _ := rpc.Dial("tcp", serverAddr)

	minerKey := new(MinerKey)
	rpc.Register(minerKey)

	var settings blockartlib.MinerNetSettings

	err = cli.Call("RServer.Register", MinerInfo{Address: minerAddr, Key: pubKey}, &settings)
	HandleError(err)
	fmt.Println(settings)

	go InitHeartbeat(cli, pubKey, settings.HeartBeat)

	go rpc.Accept(lis)

	var addrSet []net.Addr
	err = cli.Call("RServer.GetNodes", pubKey, &addrSet)
	HandleError(err)

	// ConnectToMiners(addrSet)
}

func HandleError(err error) {
	if (err != nil) {
		fmt.Println(err)
	}
}