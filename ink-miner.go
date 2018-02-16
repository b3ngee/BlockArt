/*

Ink Miner.

Usage:
go run ink-miner.go [server ip:port] [pubKey] [privKey]
server ip:port: server IP addr
pubKey + privKey: key pair to validate connecting art nodes
*/

package main

import (
	"reflect"

	"./blockartlib"

	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/md5"
	"crypto/x509"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	mrand "math/rand"
	"net"
	"net/rpc"
	"os"
	"strconv"
	"strings"
	"time"
	//"math/big"
)

var settings blockartlib.MinerNetSettings

var artNodeID int

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

type Block struct {
	PreviousBlock  *Block
	PreviousHash   string
	Hash           string
	SetOPs         []Operation
	MinerPubKey    ecdsa.PublicKey
	Nonce          uint32
	TotalInkAmount uint32 // Total = Bank - Opcost (+ Reward when you mine the block)
	InkBank        uint32 // Bank  = Previous block total (with same pubkey as yourself)
	PathLength     int
	IsEndBlock     bool
}

type Operation struct {
	ArtNodeID      int
	ShapeType      blockartlib.ShapeType
	UniqueID       string
	ArtNodePubKey  ecdsa.PublicKey
	OPSigR         *big.Int
	OPSigS         *big.Int
	ValidateNum    int
	ShapeSvgString string
	Fill           string
	Stroke         string
	OpInkCost      uint32
	OpType         string
	Lines          []Line
	DeleteUniqueID string
	PathShape      string
}

type Line struct {
	Start Point
	End   Point
}

type Point struct {
	X float64
	Y float64
}

type LongestBlockChain struct {
	BlockChain []Block
}

// Keeps track of all the keys & Miner Address so miner can send it to other miners.
var privKey ecdsa.PrivateKey
var pubKey ecdsa.PublicKey

var minerAddr net.Addr

// Keeps track of all miners that are connected to this miner. (array/slice or map???)
var connectedMiners = make(map[string]Miner)

// Keeps track of all art nodes that are connected to this miner.
// var connectedArtNodeMap = make(map[string]ArtNodeInfo)

// Keeps track of all blocks generated
// GenesisBlock is at the start of the block
var blockList = []Block{}

// Queue of incoming operations
var operations = []Operation{}

// Operations that are seen already (consists of unique shape hash)
var operationsHistory = make([]string, 0)

// GenesisBlock is at the end of the block
var globalChain []Block

// FUNCTION CALLS

// Registers incoming Miner that wants to connect.
func (minerKey *MinerKey) RegisterMiner(minerInfo *MinerInfo, reply *MinerInfo) error {
	cli, err := rpc.Dial("tcp", minerInfo.Address.String())

	miner := Miner{Address: minerInfo.Address, Key: minerInfo.Key, Cli: cli}
	connectedMiners[miner.Address.String()] = miner

	*reply = MinerInfo{Address: minerAddr, Key: pubKey}

	return err
}

// Checks blocklist against longest chain and find unvalidated operations in the shorter chain.
// Moves the unvalidated operation to the operations queue when it finds them.
func checkUnvalidatedOperation() {
	for i := len(blockList); i > 0; i-- {

		if blockList[i].IsEndBlock && (blockList[i].Hash != globalChain[0].Hash) {

			shorterChain := FindBlockChainPath(blockList[i])

			for j := len(shorterChain); j > 1; j-- {

				for k := 0; k < len(shorterChain[j].SetOPs); k++ {

					// Compares the end block path length and the path length of the block that has the current operation
					if (shorterChain[len(shorterChain)-1].PathLength - shorterChain[j].PathLength) < shorterChain[j].SetOPs[k].ValidateNum {

						// If it is validated, append to operations queue. Otherwise, drop it and let timeout handle the case.
						err := ValidateOperationForLongestChain(shorterChain[j].SetOPs[k], globalChain)
						if err == nil {
							operations = append(operations, shorterChain[j].SetOPs[k])
						}
					}
				}
			}
		}
	}
}

// Updates the longest block chain in the Miner Network.
// If a miner receives this call, it will compare the length of blockchain received with their own longest blockchain:
//		length of blockchain received > length of own longest blockchain -> replace own blockchain with longer and send that to neighbours
//		length of blockchain received < length of own longest blockchain -> send own blockchain to neighbours
//		length of blockchain received = length of own longest blockchain -> check if its exactly same as own blockchain
// If length of blockchain received = length of own longest blockchain:
// 		exactly the same blockchain -> do not send it to neighbours anymore
//		not the same blockchain -> keep the blockchain
func (minerKey *MinerKey) UpdateLongestBlockChain(longestBlockChain LongestBlockChain, reply *string) error {
	mrand.Seed(time.Now().UnixNano())

	ownLongestBlockChain := globalChain
	var err error

	// replace own blockList and send to neighbours
	if len(longestBlockChain.BlockChain) > len(ownLongestBlockChain) {

		blockList = longestBlockChain.BlockChain
		globalChain = blockList

		for _, miner := range connectedMiners {
			err = miner.Cli.Call("Minerkey.UpdateLongestBlockChain", LongestBlockChain{BlockChain: longestBlockChain.BlockChain}, &reply)
		}
	} else if len(longestBlockChain.BlockChain) < len(ownLongestBlockChain) {

		// Calls Helper function to take care of edge case where there are unvalidated OP blocks in shorter chain
		// checkUnValidatedOperation()

		// send own longest blockchain to neighbours
		for _, miner := range connectedMiners {
			err = miner.Cli.Call("Minerkey.UpdateLongestBlockChain", LongestBlockChain{BlockChain: ownLongestBlockChain}, &reply)
		}
	} else {

		// equal length, check whether blockchain are duplicates
		isDuplicate := true

		for i := 0; i < len(ownLongestBlockChain); i++ {
			if ownLongestBlockChain[i].Hash != longestBlockChain.BlockChain[i].Hash {
				isDuplicate = false
				break
			}
		}
		// not the same blockchain
		if isDuplicate != true {

			// if rand = 1, use the longest blockchain received and update the neighbours. Otherwise, keep own longest blockchain.
			if mrand.Intn(2-1)+1 == 1 {
				// TODO
				blockList = longestBlockChain.BlockChain
				for _, miner := range connectedMiners {
					err = miner.Cli.Call("Minerkey.UpdateLongestBlockChain", LongestBlockChain{BlockChain: longestBlockChain.BlockChain}, &reply)
				}
			} else {
				for _, miner := range connectedMiners {
					err = miner.Cli.Call("Minerkey.UpdateLongestBlockChain", LongestBlockChain{BlockChain: ownLongestBlockChain}, &reply)
				}
			}
		} else {
			err = nil
		}
	}

	return err
}

// Miner receives operation from other miner in the network and will add it into the Operations History Array & Operations Queue
func (minerKey *MinerKey) ReceiveOperation(operation Operation, reply *bool) error {
	exists := false
	for i := 0; i < len(operationsHistory); i++ {
		if operation.UniqueID == operationsHistory[i] {
			exists = true
			break
		}
	}

	if exists == false {
		longestBlockChain := globalChain

		err := ValidateOperationForLongestChain(operation, longestBlockChain)
		if err != nil {
			return err
		}

		operationsHistory = append(operationsHistory, operation.UniqueID)
		operations = append(operations, operation)

		for key, miner := range connectedMiners {
			err := miner.Cli.Call("MinerKey.ReceiveOperation", operation, &reply)

			if err != nil {
				delete(connectedMiners, key)
			}
		}
	}

	return nil
}

func (artkey *ArtKey) ValidateKey(artNodeKey blockartlib.ArtNodeKey, canvasSettings *blockartlib.CanvasSettings) error {

	if ecdsa.Verify(&pubKey, artNodeKey.Hash, artNodeKey.R, artNodeKey.S) == true {
		*canvasSettings = settings.CanvasSettings
	}

	artNodeID = artNodeKey.ArtNodeID

	return nil
}

func (artkey *ArtKey) AddShape(operation Operation, reply *Block) error {
	longestBlockChain := globalChain

	err := ValidateOperationForLongestChain(operation, longestBlockChain)
	if err != nil {
		return err
	}

	operationsHistory = append(operationsHistory, operation.UniqueID)
	operations = append(operations, operation)

	// Floods the network of miners with Operations
	for key, miner := range connectedMiners {

		err := miner.Cli.Call("MinerKey.ReceiveOperation", operation, &reply)

		if err != nil {
			delete(connectedMiners, key)
		}
	}

	// put reply as return type of below
	block, valid := CheckOperationValidation(operation.UniqueID)

	if valid {
		*reply = block
	}

	return nil
}

func (artkey *ArtKey) GetInk(empty string, inkAmount *uint32) error {
	longestBlockChain := globalChain
	for i := len(longestBlockChain) - 1; i >= 0; i-- {
		if reflect.DeepEqual(blockList[i].MinerPubKey, pubKey) {
			*inkAmount = blockList[i].TotalInkAmount
			break
		}
	}
	return nil
}

// UNUSED
func GetInkAmount(prevBlock *Block) uint32 {
	temp := *prevBlock
	for i := prevBlock.PathLength; i > 1; i-- {
		if reflect.DeepEqual(temp.MinerPubKey, pubKey) {
			return temp.TotalInkAmount
			break
		}
		temp = *temp.PreviousBlock
	}
	return 0
}

func GenerateBlock() {
	// FOR TESTING
	go printBlockChain()
	for {
		var difficulty int
		var isNoOp bool
		var prevBlock *Block
		var copyOfOps []Operation

		newBlock := Block{Nonce: 0, MinerPubKey: pubKey}

		if len(operations) > 0 {
			if len(operations) > 1 && CheckIntersectionWithinOp(operations) {
				copyOfOps = make([]Operation, 1)
				copy(copyOfOps, []Operation{operations[0]})
				operations = append(operations[:0], operations[1:]...)
			} else {
				copyOfOps = make([]Operation, len(operations))
				copy(copyOfOps, operations)
				operations = operations[:0]
			}
			isNoOp = false

			newBlock.SetOPs = copyOfOps
			difficulty = int(settings.PoWDifficultyOpBlock)
		} else {
			isNoOp = true
			difficulty = int(settings.PoWDifficultyNoOpBlock)
		}

		endBlocks := FindLastBlockOfLongestChain()

		if len(endBlocks) > 1 {
			if !isNoOp {
				prevBlock = SelectValidBranch(endBlocks, newBlock)
			} else {
				prevBlock = SelectRandomBranch(endBlocks)
			}
		} else {
			prevBlock = endBlocks[0]
		}

		globalChain = FindBlockChainPath(*prevBlock)

		prevBlockHash := (*prevBlock).Hash
		newBlock.PreviousHash = prevBlockHash

		zeroString := strings.Repeat("0", difficulty)

		for {
			if isNoOp && len(operations) > 0 {
				break
			}

			if (*prevBlock).Hash != globalChain[len(globalChain)-1].Hash {
				break
			}

			hash := ComputeBlockHash(newBlock)
			subString := hash[len(hash)-difficulty:]
			if zeroString == subString {
				newBlock.Hash = hash
				newBlock.IsEndBlock = true
				newBlock.PathLength = prevBlock.PathLength + 1
				newBlock.PreviousBlock = prevBlock
				newBlock.InkBank = prevBlock.TotalInkAmount
				newBlock.TotalInkAmount = newBlock.InkBank + ComputeOpCostForMiner(newBlock.MinerPubKey, copyOfOps)
				prevBlock.IsEndBlock = false

				if len(copyOfOps) == 0 {
					newBlock.TotalInkAmount = newBlock.TotalInkAmount + settings.InkPerNoOpBlock
				} else {
					newBlock.TotalInkAmount = newBlock.TotalInkAmount + settings.InkPerOpBlock
				}

				blockList = append(blockList, newBlock)
				globalChain = FindLongestBlockChain()

				SendBlockInfo(newBlock)
				break
			}
			newBlock.Nonce = newBlock.Nonce + 1
		}
	}
}

// Select valid branch among > 1 longest nodes
// If all branches are valid, then choose one at random
func SelectValidBranch(endBlocks []*Block, newBlock Block) *Block {
	var validEndBlocks []*Block
	var err error
	for _, block := range endBlocks {
		currPath := FindBlockChainPath(*block)
		for _, op := range newBlock.SetOPs {
			err = ValidateOperationForLongestChain(op, currPath)
			if err != nil {
				break
			}
		}
		validEndBlocks = append(validEndBlocks, block)
	}

	if len(validEndBlocks) > 1 {
		return SelectRandomBranch(validEndBlocks)
	} else if len(validEndBlocks) == 1 {
		return validEndBlocks[0]
	}
	return nil
}

// Selects random node in the given array
func SelectRandomBranch(endBlocks []*Block) *Block {
	length := len(endBlocks)
	mrand.Seed(time.Now().Unix())
	randIndex := mrand.Intn(length - 1)

	return endBlocks[randIndex]
}

// returns true if there is an intersection within operation set
func CheckIntersectionWithinOp(operations []Operation) bool {
	length := len(operations)
	// for every op in operations, for every line segment in the op, compare to next operation's every line segment.
	for i := 0; i < length; i++ {
		for j := 0; j < length; j++ {
			if i != j {
				for _, line1 := range operations[i].Lines {
					//each line of op1 is compared against op2's lines
					for _, line2 := range operations[j].Lines {
						if CheckIntersectionLines(line1.Start, line1.End, line2.Start, line2.End) && operations[i].ArtNodeID != operations[j].ArtNodeID {
							return true
						}
					}
				}
			}
		}
	}
	return false
}

// Checks for any possible intersection between one operation and the operations in
// the longest block chain
// returns error if there is an intersection, nil if there isn't
func CheckIntersection(operation Operation) error {
	blockChain := globalChain
	for _, line := range operation.Lines {
		deletes := []Operation{}
		start := line.Start
		end := line.End
		for k := len(blockChain) - 1; k > 0; k-- {
			for _, op := range blockChain[k].SetOPs {
				for _, opLine := range op.Lines {
					opStart := opLine.Start
					opEnd := opLine.End
					if CheckIntersectionLines(start, end, opStart, opEnd) && operation.ArtNodeID != op.ArtNodeID {
						if op.OpType == "Add" {
							exists := false
							for i, delOp := range deletes {
								if delOp.DeleteUniqueID == operation.UniqueID {
									deletes = append(deletes[:i], deletes[i+1:]...)
									exists = true
									break
								}
							}
							if !exists {
								return blockartlib.ShapeOverlapError(op.UniqueID)
							}
						} else {
							deletes = append(deletes, op)
						}
						break
					}
				}
			}
		}
	}
	return nil
}

// Checks if given set of points intersect
// Logic sourced from www.geeksforgeeks.org/check-if-two-given-line-segments-intersect/
func CheckIntersectionLines(p1 Point, p2 Point, p3 Point, p4 Point) bool {
	o1 := Orientation(p1, p2, p3)
	o2 := Orientation(p1, p2, p4)
	o3 := Orientation(p3, p4, p1)
	o4 := Orientation(p3, p4, p2)

	if o1 != o2 && o3 != o4 {
		return true
	}

	return false
}

// Find orientation of the triplet points
// 1 -> c1, c2, c3 are colinear
// 2 -> Clockwise
// 3 -> Counterclockwise
func Orientation(c1 Point, c2 Point, c3 Point) int {
	val := (c2.Y-c1.Y)*(c3.X-c2.X) - (c2.X-c1.X)*(c3.Y-c2.Y)

	if val == 0 {
		return 0
	} else if val > 0 {
		return 1
	} else {
		return 2
	}
}

func FindLongestBlockChain() []Block {
	endNodes := FindLastBlockOfLongestChain()

	if len(endNodes) > 1 {
		randNode := SelectRandomBranch(endNodes)
		return FindBlockChainPath(*randNode)
	}
	return FindBlockChainPath(*endNodes[0])
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

func FindBlockChainPath(block Block) []Block {
	tempBlock := block
	path := []Block{}

	for i := block.PathLength; i > 1; i-- {
		path = append(path, tempBlock)
		tempBlock = *tempBlock.PreviousBlock
	}

	path = append(path, tempBlock)
	ReverseArray(path)

	return path
}

// send out the block information to peers in the connected network of miners
func SendBlockInfo(block Block) error {
	replyStr := ""
	for key, miner := range connectedMiners {
		err := miner.Cli.Call("MinerKey.ReceiveBlock", block, &replyStr)
		if err != nil {
			fmt.Println("Connection error in SendBlockInfo: ", err.Error())
			delete(connectedMiners, key)
		}
	}
	return nil
}

// once information about a block is received unpack that message and update ink-miner
func (minerKey *MinerKey) ReceiveBlock(receivedBlock Block, reply *string) error {
	hash := receivedBlock.Hash
	previousHash := receivedBlock.PreviousHash
	operations := receivedBlock.SetOPs

	// Already exists in local blockchain, do nothing
	if ExistInLocalBlockchain(hash) {
		return nil
	}

	// Check if previous hash is a block that exists in the block chain
	var previousBlock *Block
	if prevBlock, exists := CheckPreviousBlock(previousHash); exists {
		previousBlock = prevBlock
	} else {
		return errors.New("Failed to validate hash of a previous block")
	}

	// Check if received block is a No-Op or Op block based on length of operations
	if len(operations) == 0 {
		if !ComputeTrailingZeroes(hash, settings.PoWDifficultyNoOpBlock) {
			return errors.New("No-op block proof of work does not match the zeroes")
		}
	} else {
		if ComputeTrailingZeroes(hash, settings.PoWDifficultyOpBlock) {
			for _, op := range operations {
				err := ValidateOperationForLongestChain(op, globalChain)
				if err != nil {
					fmt.Println("Error after VOFLC: ", err.Error())
					return errors.New("Block contains operations that failed to validate")
				}
			}
		} else {
			return errors.New("Op block proof of work does not match the zeroes")
		}
	}

	// After all validations pass, we set properties of block, append to blockchain and send to network
	receivedBlock.PathLength = previousBlock.PathLength + 1
	receivedBlock.PreviousBlock = previousBlock
	receivedBlock.InkBank = previousBlock.TotalInkAmount
	receivedBlock.TotalInkAmount = receivedBlock.InkBank + ComputeOpCostForMiner(receivedBlock.MinerPubKey, operations) // add any operations performed by the miner that generated this block
	previousBlock.IsEndBlock = false

	if len(operations) == 0 {
		receivedBlock.TotalInkAmount = receivedBlock.TotalInkAmount + settings.InkPerNoOpBlock
	} else {
		receivedBlock.TotalInkAmount = receivedBlock.TotalInkAmount + settings.InkPerOpBlock
	}

	blockList = append(blockList, receivedBlock)
	globalChain = FindLongestBlockChain()

	SendBlockInfo(receivedBlock)

	return nil
}

func ExistInLocalBlockchain(blockHash string) bool {
	for _, block := range blockList {
		if blockHash == block.Hash {
			return true
		}
	}

	return false
}

// returns a boolean true if hash contains specified number of zeroes num at the end
func ComputeTrailingZeroes(hash string, numOfZero uint8) bool {
	zeroString := strings.Repeat("0", int(numOfZero))

	if strings.HasSuffix(hash, zeroString) {
		return true
	}
	return false
}

// checks that the previousHash in the block struct points to a previous generated block's Hash
func CheckPreviousBlock(hash string) (*Block, bool) {
	var blockPtr *Block
	for i, block := range blockList {
		if block.Hash == hash {
			blockPtr = &blockList[i]
			return blockPtr, true
		}
	}
	return blockPtr, false
}

// compute the total cost of operations for the current miner
func ComputeOpCostForMiner(publicKey ecdsa.PublicKey, operations []Operation) uint32 {
	var cost uint32
	for _, op := range operations {
		if reflect.DeepEqual(publicKey, op.ArtNodePubKey) {
			if op.OpType == "Add" {
				cost = cost - op.OpInkCost
			}
			// Fix delete by fixing DeleteShape's OpInkCost
		}
	}

	return cost
}

// call this for op-blocks to validate the op-block
func ValidateOperationForLongestChain(operation Operation, longestChain []Block) error {

	// made a dummy private key but it should correspond to blockartlib shape added?
	// need clarification from you guys
	// Check that each operation in the block has a valid signature

	// CHECK THIS: DON'T THINK IT'S RIGHT
	if !ecdsa.Verify(&operation.ArtNodePubKey, []byte("This is the private key!"), operation.OPSigR, operation.OPSigS) {
		return errors.New("Failed to validate operation signature")
	}

	// Checks for DeleteShape
	if operation.OpType == "Delete" {

		DeleteConfirmed := false
		for i := 0; i < len(longestChain); i++ {

			for m := 0; m < len(longestChain[i].SetOPs); m++ {
				if operation.DeleteUniqueID == longestChain[i].SetOPs[m].UniqueID {
					DeleteConfirmed = true
					return nil
				}
			}
		}

		if DeleteConfirmed == false {
			return blockartlib.ShapeOwnerError(operation.DeleteUniqueID)
		}
	}

	// Checks for AddShape
	if operation.OpType == "Add" {
		// Validates the operation against duplicate signatures (UniqueID)
		for j := 0; j < len(longestChain); j++ {

			for k := 0; k < len(longestChain[j].SetOPs); k++ {

				if operation.UniqueID == longestChain[j].SetOPs[k].UniqueID {
					fmt.Println("Failed on Add")
					return blockartlib.ShapeOverlapError(operation.UniqueID)
				}
			}
		}

		// Validates the operation against the Ink Amount Check
		for l := len(longestChain) - 1; l >= 0; l-- {
			if reflect.DeepEqual(longestChain[l].MinerPubKey, operation.ArtNodePubKey) {
				fmt.Println("found a match, opcost: ", operation.OpInkCost)
				fmt.Println("Longest chain ink bank: ", longestChain[l].InkBank)
				difference := int(longestChain[l].InkBank) - int(operation.OpInkCost)
				if difference < 0 {
					fmt.Println("Failed on Ink")
					return blockartlib.InsufficientInkError(longestChain[l].InkBank)
				}
				break
			}
		}
	}

	return CheckIntersection(operation)
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
// Checks if the address already exists in ConnectedMiners map, it will skip connecting to them again.
func ConnectToMiners(addrSet []net.Addr, currentAddress net.Addr, currentPubKey ecdsa.PublicKey) {
	for i := 0; i < len(addrSet); i++ {
		addr := addrSet[i].String()

		if _, ok := connectedMiners[addr]; !ok {
			cli, err := rpc.Dial("tcp", addr)

			var reply MinerInfo
			err = cli.Call("MinerKey.RegisterMiner", MinerInfo{Address: currentAddress, Key: currentPubKey}, &reply)
			HandleError(err)

			connectedMiners[reply.Address.String()] = Miner{Address: reply.Address, Key: reply.Key, Cli: cli}
		}
	}
}

// Goroutine to get nodes (if number of connected miners go below min-num-miner-connections, get more from server)
func GetNodes(cli *rpc.Client, minNumberConnections int) {
	for {
		if len(connectedMiners) < minNumberConnections {

			var addrSet []net.Addr

			err := cli.Call("RServer.GetNodes", pubKey, &addrSet)
			HandleError(err)

			ConnectToMiners(addrSet, minerAddr, pubKey)
		}

		time.Sleep(7000 * time.Millisecond)
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
			hash = hash + string(block.SetOPs[i].ShapeType) + block.SetOPs[i].UniqueID
		}
	}
	minerPubKey, _ := json.Marshal(block.MinerPubKey)
	h.Write([]byte(hash + string(minerPubKey) + strconv.Itoa(int(block.Nonce))))
	str := hex.EncodeToString(h.Sum(nil))
	return str
}

// Updates everyone's blockchain to only include the LONGEST blockchain
func SyncMinersLongestChain() {
	for {
		if len(connectedMiners) > 0 {

			longestBlockChain := globalChain
			blockList = longestBlockChain
			var reply string

			for _, miner := range connectedMiners {
				miner.Cli.Call("MinerKey.UpdateLongestBlockChain", LongestBlockChain{BlockChain: longestBlockChain}, &reply)
			}
		}

		time.Sleep(10 * time.Second)
	}
}

// CALL THIS TO REVERSE AFTER FINDING LONGEST CHAIN USING END NODE
// [E ... G] -> [G ... E] TO MATCH BLOCKLIST ORDERING
func ReverseArray(reverseThis []Block) {
	for i, j := 0, len(reverseThis)-1; i < j; i, j = i+1, j-1 {
		reverseThis[i], reverseThis[j] = reverseThis[j], reverseThis[i]
	}
}

// Checks whether or not operations are validated or not and returns block where op is in (check validateNum against the block)
func CheckOperationValidation(uniqueID string) (Block, bool) {
	//timeOut := 0

	// blockToCheck is the block that contains the checked Operation
	blockToCheck := Block{}
	opToCheck := Operation{}
	foundBlock := false

	for {
		// Times out, sends reply back (3 mins currently)
		// if timeOut == 36 {
		// 	return Block{}, false
		// }

		// Get the block that we need (where the operation is in)
		if !foundBlock {
			for i := 0; i < len(blockList); i++ {
				opList := blockList[i].SetOPs

				for j := 0; j < len(opList); j++ {

					if opList[j].UniqueID == uniqueID {
						foundBlock = true
						blockToCheck = blockList[i]
						opToCheck = opList[j]
						break
					}
				}
			}
		}

		if foundBlock {
			// Goes through each end blocks to find the one that consists blockToCheck
			for k := len(blockList) - 1; k > 0; k-- {

				if blockList[k].IsEndBlock == true {

					blockChain := FindBlockChainPath(blockList[k])
					for l := 0; l < len(blockChain); l++ {

						if blockChain[l].Hash == blockToCheck.Hash {
							if blockList[k].PathLength-blockChain[l].PathLength >= opToCheck.ValidateNum {
								return blockToCheck, true
							}
							break
						}
					}
				}
			}
		}

		time.Sleep(5 * time.Second)
		//timeOut = timeOut + 1
	}
}

func FindOperationInLongestChain(shapeHash string) Operation {
	longestBlockChain := globalChain

	for _, block := range longestBlockChain {
		for _, op := range block.SetOPs {
			if op.UniqueID == shapeHash {
				return op
			}
		}
	}

	return Operation{}
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
		return errors.New("Hash does not exist")
	}

	*children = result
	return nil
}

func (artkey *ArtKey) GetGenesisBlock(doNotUse string, genesisHash *string) error {
	*genesisHash = blockList[0].Hash
	return nil
}

func (artkey *ArtKey) GetShapes(blockHash string, shapeHashes *[]string) error {
	result := []string{}
	for _, block := range blockList {
		if block.Hash == blockHash {
			operations := block.SetOPs

			for _, op := range operations {
				result = append(result, op.UniqueID)
			}

			*shapeHashes = result
			return nil
		}
	}

	return errors.New("Invalid shape hash")
}

func (artKey *ArtKey) GetOperationWithShapeHash(shapeHash string, operation *Operation) error {
	op := FindOperationInLongestChain(shapeHash)

	if op.UniqueID == "" {
		return errors.New("Does not exist")
	}

	*operation = op
	return nil
}

func (artKey *ArtKey) DeleteShape(shapeHash string, inkRemaining *uint32) error {
	op := FindOperationInLongestChain(shapeHash)

	if op.UniqueID == "" {
		return errors.New("Does not exist")
	}
	if pubKey != op.ArtNodePubKey {
		return errors.New("Did not create")
	}

	longestBlockChain := globalChain
	for i := len(longestBlockChain) - 1; i >= 0; i-- {
		block := longestBlockChain[i]
		if reflect.DeepEqual(block.MinerPubKey, pubKey) {
			*inkRemaining = block.TotalInkAmount + op.OpInkCost
			break
		}
	}

	return nil
}

func (artkey *ArtKey) ValidateDelete(operation Operation, reply *bool) error {
	longestBlockChain := globalChain

	err := ValidateOperationForLongestChain(operation, longestBlockChain)
	if err != nil {
		return err
	}

	operationsHistory = append(operationsHistory, operation.UniqueID)
	operations = append(operations, operation)

	// Floods the network of miners with Operations
	for key, miner := range connectedMiners {

		err := miner.Cli.Call("MinerKey.ReceiveOperation", operation, &reply)

		if err != nil {
			delete(connectedMiners, key)
		}
	}

	_, valid := CheckOperationValidation(operation.UniqueID)
	*reply = valid

	for i := len(longestBlockChain) - 1; i >= 0; i-- {
		block := longestBlockChain[i]
		if reflect.DeepEqual(block.MinerPubKey, pubKey) {
			block.TotalInkAmount = operation.OpInkCost
			break
		}
	}

	return nil
}

func main() {
	gob.Register(&net.TCPAddr{})
	gob.Register(&elliptic.CurveParams{})

	serverAddr := os.Args[1]

	privateKeyBytesRestored, _ := hex.DecodeString(os.Args[3])
	priv, _ := x509.ParseECPrivateKey(privateKeyBytesRestored)

	privKey = *priv
	pubKey = privKey.PublicKey

	lis, err := net.Listen("tcp", ":0")
	minerAddr = lis.Addr()

	cli, _ := rpc.Dial("tcp", serverAddr)

	minerKey := new(MinerKey)
	rpc.Register(minerKey)

	artKey := new(ArtKey)
	rpc.Register(artKey)

	err = cli.Call("RServer.Register", MinerInfo{Address: minerAddr, Key: pubKey}, &settings)
	HandleError(err)

	blockList = append(blockList, Block{Hash: settings.GenesisBlockHash, PathLength: 1, IsEndBlock: true})
	globalChain = FindLongestBlockChain()

	go InitHeartbeat(cli, pubKey, settings.HeartBeat)

	go rpc.Accept(lis)

	var addrSet []net.Addr
	err = cli.Call("RServer.GetNodes", pubKey, &addrSet)
	HandleError(err)

	ConnectToMiners(addrSet, minerAddr, pubKey)

	go GetNodes(cli, int(settings.MinNumMinerConnections))

	go SyncMinersLongestChain()

	GenerateBlock()
}

// FOR TESTING
func printBlockChain() {
	for {
		time.Sleep(90 * time.Second)
		fmt.Println(globalChain)
	}
}

func HandleError(err error) {
	if err != nil {
		fmt.Println(err)
	}
}
