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

	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/md5"
	"crypto/rand"
	"crypto/x509"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/rpc"
	"os"
	"strconv"
	"strings"
	"time"
	//"math/big"
)

var settings blockartlib.MinerNetSettings

type MinerKey int

type ArtKey int

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
	PreviousBlock *Block
	PreviousHash  string
	Hash          string
	SetOPs        []Operation
	MinerPubKey   ecdsa.PublicKey
	Nonce         uint32
	InkAmount     uint32
	PathLength    int
	IsEndBlock    bool
}

type Operation struct {
	ShapeType     blockartlib.ShapeType
	OPSignature   string
	ArtNodePubKey ecdsa.PublicKey

	//Adding some new fields that could come in handy trying to validate
	OpInkCost uint32
	OpType    string
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
		prevBlock := FindLastBlockOfLongestChain()
		prevBlockHash := ComputeBlockHash(*prevBlock)

		var newBlock Block
		var difficulty int
		var isNoOp bool

		if len(operations) > 0 {
			isNoOp = false
			copyOfOps := make([]Operation, len(operations))
			copy(copyOfOps, operations)
			operations = operations[:0]

			newBlock = Block{PreviousHash: prevBlockHash, SetOPs: copyOfOps, Nonce: 0, MinerPubKey: pubKey, InkAmount: settings.InkPerOpBlock}
			difficulty = int(settings.PoWDifficultyOpBlock)
		} else {
			isNoOp = true
			newBlock = Block{PreviousHash: prevBlockHash, Nonce: 0, MinerPubKey: pubKey, InkAmount: settings.InkPerNoOpBlock}
			difficulty = int(settings.PoWDifficultyNoOpBlock)
		}

		zeroString := strings.Repeat("0", difficulty)

		for {
			if isNoOp && len(operations) > 0 {
				break
			}

			hash := ComputeBlockHash(newBlock)
			subString := hash[len(hash)-difficulty:]
			if zeroString == subString {
				// validate secret with other miners?
				if validatedWithNetwork(newBlock) {
					prevBlock.IsEndBlock = false

					newBlock.PathLength = prevBlock.PathLength + 1
					newBlock.PreviousBlock = prevBlock
					newBlock.Hash = hash
					newBlock.IsEndBlock = true

					blockList = append(blockList, newBlock)

					SendBlockInfo(&newBlock)
				}
				break
			}
			newBlock.Nonce = newBlock.Nonce + 1
		}
	}
}

func FindLastBlockOfLongestChain() []*Block {
	tempMaxLength := -1
	lastBlocks := []*Block{}

	for i := len(blockList) - 1; i >= 0; i-- {
		currBlock := blockList[i]

		if currBlock.IsEndBlock && currBlock.PathLength >= tempMaxLength {
			if currBlock.PathLength > tempMaxLength {
				lastBlocks = []*Block{}
				tempMaxLength = currBlock.PathLength
			}
			lastBlocks = append(lastBlocks, &blockList[i])
		}
	}

	return lastBlocks
}

func FindBlockChainPath(block *Block) []Block {
	tempBlock := *block
	path := []Block{}

	for i := block.PathLength; i > 1; i-- {
		path = append(path, tempBlock)
		tempBlock = *tempBlock.PreviousBlock
	}

	path = append(path, tempBlock)
	return path
}

// TODO: INCOMPLETE?
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

// TODO: INCOMPLETE?
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
func (minerKey *MinerKey) ReceiveBlock(block Block, reply *string) error {
	blockType := block.SetOPs
	var i int = 0
	var hash string = ComputeBlockHash(block)

	if CheckPreviousBlock(block.PreviousHash) {
		fmt.Println("Block exists within the blockchain")
	} else {
		return errors.New("failed to validate hash of a previous block")
	}

	if len(blockType) == 0 {
		i = 1
	} else {
		i = 2
	}

	switch i {
	case 1:
		if ComputeTrailingZeroes(hash, settings.PoWDifficultyNoOpBlock) {
		} else {
			return errors.New("No-op block proof of work does not match the zeroes of nonce")
		}
	case 2:
		if ComputeTrailingZeroes(hash, settings.PoWDifficultyOpBlock) {
			ValidateOperation(block)
		} else {
			return errors.New("No-op block proof of work does not match the zeroes of nonce")
		}
	}
	return errors.New("failed to validate block")
}

// returns a boolean true if hash contains specified number of zeroes num at the end
func ComputeTrailingZeroes(hash string, num uint8) bool {
	var i uint8 = 0
	var numZeroesStr = ""
	for i = 1; i < num; i++ {
		numZeroesStr += "0"
	}
	// HARDCODED FOR NOW NEED TO FIX IT REAL EASY JUST NEED TO GET MINER SETTINGS
	numZeroesStr = "00000"
	if strings.HasSuffix(hash, numZeroesStr) {
		return true
	}
	return false
}

// checks that the previousHash in the block struct points to a previous generated block's Hash
func CheckPreviousBlock(hash string) bool {
	for _, block := range blockList {
		if block.Hash == hash {
			return true
		}
		continue
	}
	return false
}

// call this for op-blocks to validate the op-block
func ValidateOperation(block Block) error {

	// made a dummy private key but it should correspond to blockartlib shape added?
	// need clarification from you guys
	// Check that each operation in the block has a valid signature
	privateKeyBytesRestored, _ := hex.DecodeString("yoloswag")
	privKey, _ := x509.ParseECPrivateKey(privateKeyBytesRestored)

	r, s, err := ecdsa.Sign(rand.Reader, privKey, []byte(block.Hash))
	HandleError(err)

	if ecdsa.Verify(&block.MinerPubKey, []byte(block.Hash), r, s) {
		fmt.Println("op-sig is valid .... continuing validation")
	} else {
		return errors.New("failed to validate operation signature")
	}

	// Check that each operation has sufficient ink associated with the public key that generated the operation.
	for _, operation := range block.SetOPs {
		totalInk := block.InkAmount
		currentOpCost := operation.OpInkCost

		if totalInk >= currentOpCost {
			totalInk = totalInk - currentOpCost
			continue
		} else {
			return errors.New("not enough ink to perform operation")
		}

		if operation.OpType == "Delete" {
			// TODO: make sure that delete operation deletes a shape that exists, how are we keeping track
			// of the shapes?
		}

		// Check that the operation with an identical signature has not been previously added to the blockchain
		for _, finishedop := range operations {
			if finishedop.OPSignature == operation.OPSignature {
				return errors.New("operation with same signature exists")
			}
			continue
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
		time.Sleep(time.Duration(int(heartBeat)/5) * time.Millisecond)
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

func (artkey *ArtKey) GetChildren(blockHash string, children *[]string) error {
	hashExists := false
	result := []string{}

	for _, block := range blockList {

		if block.Hash == blockHash {
			hashExists = true
		}
		if block.PreviousHash == blockHash {
			result = append(result, block.Hash)
		}
	}

	if !hashExists {
		result = append(result, "INVALID")
	}

	*children = result

	return nil
}

func (artkey *ArtKey) GetGenesisBlock(doNotUse string, genesisHash *string) error {
	*genesisHash = blockList[0].Hash
	return nil
}

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

	err = cli.Call("RServer.Register", MinerInfo{Address: minerAddr, Key: pubKey}, &settings)
	HandleError(err)

	// fmt.Println(settings)

	blockList = append(blockList, Block{Hash: settings.GenesisBlockHash, PathLength: 1, IsEndBlock: true})

	/////////////////////////////////////////////
	// VALIDATION TEXTING
	// checking block validation
	// blockList = append(blockList, Block{Hash: "345", PathLength: 1, IsEndBlock: true})
	// blockList = append(blockList, Block{Hash: "1234", PathLength: 1, IsEndBlock: true})
	// var singleop Operation = Operation{ShapeType: 5, OPSignature: "yolo", ArtNodePubKey: pubKey}
	// var operationsCheck []Operation
	// operationsCheck = append(operationsCheck, singleop)
	// previousTestBlock := Block{PreviousHash: "345", Hash: "1234"}
	// blocktocheck := Block{PreviousBlock: &previousTestBlock, PreviousHash: "1234", Hash: "yee",
	// SetOPs: operationsCheck, MinerPubKey: pubKey, Nonce: 5, InkAmount: 6}

	///////////////////////////////////////////////

	//GenerateBlock(settings)

	go InitHeartbeat(cli, pubKey, settings.HeartBeat)

	go rpc.Accept(lis)

	var addrSet []net.Addr
	err = cli.Call("RServer.GetNodes", pubKey, &addrSet)
	HandleError(err)

	ConnectToMiners(addrSet, minerAddr, pubKey)

	for {

	}
}

func HandleError(err error) {
	if err != nil {
		fmt.Println(err)
	}
}
