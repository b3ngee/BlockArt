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
	"crypto/md5"
	"crypto/rand"
	"fmt"
	"net"
	"net/rpc"
	"os"
	"strconv"
	"strings"
	"time"

	"./blockartlib"
	//"encoding/hex"
	"encoding/hex"
	"encoding/json"
	// "strconv"
	// "strings"
	// "crypto/md5"
	"encoding/gob"
	"errors"
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
	InkAmount    uint32
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

// Keeps track of all blocks generated
var blockList = []Block{}

// Array of incoming operations
var operations = []Operation{}

// FUNCTION CALLS

// Registers incoming Miner that wants to connect.
func (minerKey *MinerKey) RegisterMiner(minerInfo *MinerInfo, reply *MinerInfo) error {
	cli, err := rpc.Dial("tcp", minerInfo.Address.String())

	miner := Miner{Address: minerInfo.Address, Key: minerInfo.Key, Cli: cli}
	connectedMiners[miner.Address.String()] = miner

	*reply = MinerInfo{Address: minerAddr, Key: pubKey}

	return err
}

// TODO
func validatedWithNetwork(block Block) bool {
	// TODO
	return true
}

func GenerateBlock(settings blockartlib.MinerNetSettings) {
	for {
		prevBlock := blockList[len(blockList)-1]
		prevBlockHash := ComputeBlockHash(prevBlock)

		var newBlock Block
		var difficulty int

		if len(operations) > 0 {
			copyOfOps := make([]Operation, len(operations))
			copy(copyOfOps, operations)
			operations = operations[:0]

			newBlock = Block{PreviousHash: prevBlockHash, SetOPs: copyOfOps, Nonce: 0, MinerPubKey: pubKey, InkAmount: settings.InkPerOpBlock}
			difficulty = int(settings.PoWDifficultyOpBlock)
		} else {
			newBlock = Block{PreviousHash: prevBlockHash, Nonce: 0, MinerPubKey: pubKey, InkAmount: settings.InkPerNoOpBlock}
			difficulty = int(settings.PoWDifficultyNoOpBlock)
		}

		zeroString := strings.Repeat("0", difficulty)

		for {
			hash := ComputeBlockHash(newBlock)
			subString := hash[len(hash)-difficulty:]
			if zeroString == subString {
				// validate secret with other miners?
				if validatedWithNetwork(newBlock) {
					blockList = append(blockList, newBlock)

					// TODO
					// once we compute whatever and the block gets generated, call the function
					// SendBlock in order to create the block information and then broadcast the new block
					// to all the other miners in the network of connectedMiners[]
					SendBlockInfo(&newBlock)
				}
				fmt.Println("this is block ", newBlock)
				break
			}
			//fmt.Println(newBlock.Nonce)
			newBlock.Nonce = newBlock.Nonce + 1
		}
	}
}

// placeholder for what this function must do
func (minerKey *MinerKey) WriteBlock(block *Block, miner *Miner) error {

	// minerIPPort := miner.minerAddr
	// not sure how to connect to net.Addr type
	// for each miner there is a minerAddr, connect to that and send the block

	conn, err := net.Dial("udp", "127.0.0.0.1:80")
	payload, err := json.Marshal(block)
	conn.Write(payload)

	defer conn.Close()

	return err
}

// send out the block information to peers in the connected network of miners
func SendBlockInfo(block *Block) error {

	for key, miner := range connectedMiners {

		err := miner.Cli.Call("MinerKey.WriteBlock", block, miner)

		if err != nil {
			delete(connectedMiners, key)
		}

		// cannot connect to said miner then delete from connectedMiners

	}
	return errors.New("Parse error")
}

// once information about a block is received unpack that message and update ink-miner
func (minerKey *MinerKey) ReceiveBlock(block *Block, reply *string) error {

	// get the settings in config file to check for specific validation
	var settings blockartlib.MinerNetSettings

	//Block validations:
	// Check that the nonce for the block is valid: PoW is correct and has the right difficulty.
	blockType := block.SetOPs
	var i int = 0

	if len(blockType) == 0 {
		i = 1
	} else {
		i = 2
	}

	switch i {
	case 1:
		if settings.PoWDifficultyNoOpBlock != uint8(block.Nonce) {
			return errors.New("No-op block proof of work does not match the zeroes of nonce")
		} else {
			//TODO
			// continue validation
			fmt.Println("No-op Block has the same zeroes as nonce")
		}
	case 2:
		if settings.PoWDifficultyOpBlock != uint8(block.Nonce) {
			return errors.New("op block proof of work does not match the zeroes of nonce")
		} else {
			//TODO
			// continue validation
			fmt.Println("op Block has the same zeroes as nonce")

		}
	}

	return errors.New("failed to validate block")
}

// HELPER FUNCTIONS

// Initializes the heartbeat sends message to the server (message is the public key of miner so the server will remember it).
func InitHeartbeat(cli *rpc.Client, pubKey ecdsa.PublicKey, heartBeat uint32) {
	for {
		var reply bool
		err := cli.Call("RServer.HeartBeat", pubKey, &reply)
		HandleError(err)
		time.Sleep(10 * time.Millisecond)
	}
}

// Connect to the miners that the server has given.
func ConnectToMiners(addrSet []net.Addr, currentAddress net.Addr, currentPubKey ecdsa.PublicKey) {
	for i := 0; i < len(addrSet); i++ {

		cli, err := rpc.Dial("tcp", addrSet[i].String())

		var reply MinerInfo
		err = cli.Call("MinerKey.RegisterMiner", MinerInfo{Address: currentAddress, Key: currentPubKey}, &reply)
		HandleError(err)

		connectedMiners[reply.Address.String()] = Miner{Address: reply.Address, Key: reply.Key, Cli: cli}
	}
}

// Returns the MD5 hash as a hex string for the OP Block (prev-hash + op + op-signature + pub-key + nonce) or No-OP Block (prev-hash + pub-key + nonce).
// Nonce is the secret for this assignment, keep increasing Nonce to find a hash with correct trailing number of zeroes.
func ComputeBlockHash(block Block) string {
	h := md5.New()
	hash := block.PreviousHash
	// this states if it is an op block (has set of OPs) or not
	if len(block.SetOPs) > 0 {
		for i := 0; i < len(block.SetOPs); i++ {
			hash = hash + string(block.SetOPs[i].ShapeType) + block.SetOPs[i].OPSignature
		}
	}
	minerPubKey, _ := json.Marshal(block.MinerPubKey)
	h.Write([]byte(hash + string(minerPubKey) + strconv.Itoa(int(block.Nonce))))
	str := hex.EncodeToString(h.Sum(nil))
	return str
}

func main() {
	gob.Register(&net.TCPAddr{})
	gob.Register(&elliptic.CurveParams{})

	serverAddr := os.Args[1]
	// pubKey := os.Args[2]
	// privKey := os.Args[3]

	privKey, _ := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
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

	blockList = append(blockList, Block{PreviousHash: settings.GenesisBlockHash})

	GenerateBlock(settings)

	go InitHeartbeat(cli, pubKey, settings.HeartBeat)

	go rpc.Accept(lis)

	var addrSet []net.Addr
	err = cli.Call("RServer.GetNodes", pubKey, &addrSet)
	HandleError(err)

	ConnectToMiners(addrSet, minerAddr, pubKey)

	// for {

	// }
}

func HandleError(err error) {
	if err != nil {
		fmt.Println(err)
	}
}
