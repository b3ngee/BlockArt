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
	"crypto/x509"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
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
	ShapeType     blockartlib.ShapeType
	UniqueID      string
	ArtNodePubKey ecdsa.PublicKey
	OPSigR        *big.Int
	OPSigS        *big.Int

	//Adding some new fields that could come in handy trying to validate
	ValidateNum    int
	ShapeSvgString string
	Fill           string
	Stroke         string
	OpInkCost      uint32
	OpType         string
	Lines          []Line
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
var blockList = []Block{}

// Queue of incoming operations
var operations = []Operation{}

// Operations that are seen already (consists of unique shape hash)
var operationsHistory = make([]string, 0)

// FUNCTION CALLS

// Registers incoming Miner that wants to connect.
func (minerKey *MinerKey) RegisterMiner(minerInfo *MinerInfo, reply *MinerInfo) error {
	cli, err := rpc.Dial("tcp", minerInfo.Address.String())

	miner := Miner{Address: minerInfo.Address, Key: minerInfo.Key, Cli: cli}
	connectedMiners[miner.Address.String()] = miner

	*reply = MinerInfo{Address: minerAddr, Key: pubKey}

	return err
}

// Updates the longest block chain in the Miner Network.
// If a miner receives this call, it will compare the length of blockchain received with their own longest blockchain:
//		length of blockchain received > length of own longest blockchain -> replace own blockchain with longer and send that to neighbours
//		length of blockchain received < length of own longest blockchain -> send own blockchain to neighbours
//		length of blockchain received = length of own longest blockchain -> check if its exactly same as own blockchain
// If length of blockchain received = length of own longest blockchain:
// 		exactly the same blockchain -> do not send it to neighbours anymore
//		not the same blockchain -> keep the blockchain
func (minerKey *MinerKey) UpdateLongestBlockChain(longestBlockChain *LongestBlockChain, reply *string) error {
	mrand.Seed(time.Now().UnixNano())

	ownLongestBlockChain := FindLongestBlockChain()
	var err error

	// replace own blockList and send to neighbours
	if len(longestBlockChain.BlockChain) > len(ownLongestBlockChain) {
		blockList = longestBlockChain.BlockChain

		for _, miner := range connectedMiners {
			err = miner.Cli.Call("Minerkey.UpdateLongestBlockChain", LongestBlockChain{BlockChain: longestBlockChain.BlockChain}, &reply)
		}
	} else if len(longestBlockChain.BlockChain) < len(ownLongestBlockChain) {

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
func (minerKey *MinerKey) ReceiveOperation(operation *Operation, reply *string) error {
	exists := false
	for i := 0; i < len(operationsHistory); i++ {

		if operation.UniqueID == operationsHistory[i] {
			exists = true
		}
	}

	if exists == false {
		operationsHistory = append(operationsHistory, operation.UniqueID)
		operations = append(operations, *operation)

		for key, miner := range connectedMiners {

			err := miner.Cli.Call("MinerKey.ReceiveOperation", operation, reply)

			if err != nil {
				delete(connectedMiners, key)
			}
		}
	}

	return nil
}

func (artkey *ArtKey) ValidateKey(artNodeKey *blockartlib.ArtNodeKey, canvasSettings *blockartlib.CanvasSettings) error {

	if ecdsa.Verify(&pubKey, artNodeKey.Hash, artNodeKey.R, artNodeKey.S) == true {
		*canvasSettings = settings.CanvasSettings
	}

	return nil
}

func (artkey *ArtKey) AddShape(operation *Operation, reply *bool) error {

	return nil
}

func (artkey *ArtKey) GetInk(_ *struct{}, inkAmount *uint32) error {
	for i := len(longestBlockChain) - 1; i >= 0; i-- {
		if IsPublicKeySame(blockList[i].MinerPubKey) {
			*inkAmount = blockList[i].TotalInkAmount
			break
		}
	}
	return nil
}

func GetInkAmount(prevBlock *Block) uint32 {
	temp := *prevBlock
	for i := prevBlock.PathLength; i > 1; i-- {
		if IsPublicKeySame(temp.MinerPubKey) {
			return temp.TotalInkAmount
			break
		}
		temp = *temp.PreviousBlock
	}
	return 0
}

func GenerateBlock(settings blockartlib.MinerNetSettings) {
	for {
		var difficulty int
		var isNoOp bool
		var prevBlock *Block

		newBlock := Block{Nonce: 0, MinerPubKey: pubKey}

		if len(operations) > 0 {
			isNoOp = false
			copyOfOps := make([]Operation, len(operations))
			copy(copyOfOps, operations)
			operations = operations[:0]

			newBlock.SetOPs = copyOfOps
			difficulty = int(settings.PoWDifficultyOpBlock)
		} else {
			isNoOp = true
			difficulty = int(settings.PoWDifficultyNoOpBlock)
		}

		zeroString := strings.Repeat("0", difficulty)

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

		prevBlockHash := ComputeBlockHash(*prevBlock)

		for {
			if isNoOp && len(operations) > 0 {
				break
			}

			hash := ComputeBlockHash(newBlock)
			subString := hash[len(hash)-difficulty:]
			if zeroString == subString {
				newBlock.Hash = hash
				newBlock.PreviousHash = prevBlockHash
				newBlock.IsEndBlock = true

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
	for _, block := range endBlocks {
		currPath := FindBlockChainPath(block)
		isValid := ValidateOperationAgainstPath(newBlock.SetOPs, currPath)
		if isValid {
			validEndBlocks = append(validEndBlocks, block)
		}
	}

	if len(validEndBlocks) > 1 {
		return SelectRandomBranch(validEndBlocks)
	} else if len(validEndBlocks) == 1 {
		return validEndBlocks[0]
	}
	return nil
}

// Checks if given branch causes any validation error for given set of operations
func ValidateOperationAgainstPath(ops []Operation, block []Block) bool {
	// TODO
	return true
}

// Selects random node in the given array
func SelectRandomBranch(endBlocks []*Block) *Block {
	length := len(endBlocks)
	mrand.Seed(time.Now().Unix())
	randIndex := mrand.Intn(length - 1)

	return endBlocks[randIndex]
}

// Checks for any possible intersection between set of operations and operations in
// the longest block chain
func CheckIntersection(operation Operation) bool {
	deletes := []Operation{}
	blockChain := FindLongestBlockChain()
	point1 := Point{x: operation.xStart, y: operation.yStart}
	point2 := Point{x: operation.xEnd, y: operation.yEnd}
	for k := len(blockChain); k > 0; k-- {
		for _, op := range blockChain[k].SetOPs {
			point3 := Point{x: op.xStart, y: op.yStart}
			point4 := Point{x: op.xEnd, y: op.yEnd}
			if CheckIntersectionLines(point1, point2, point3, point4) {
				if op.OpType == "Add" {
					exists := false
					for i, delOp := range deletes {
						if delOp.xStart == op.xStart && delOp.xEnd == op.xEnd &&
							delOp.xEnd == op.xEnd && delOp.yEnd == op.yEnd {
							deletes = append(deletes[:i], deletes[i+1:]...)
							exists = true
							break
						}
					}
					if !exists {
						return false
					}
				} else {
					deletes = append(deletes, op)
				}
			}
		}
	}
	return false
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

	if o1 == 0 && OnSegment(p1, p3, p2) {
		return true
	}
	if o2 == 0 && OnSegment(p1, p4, p2) {
		return true
	}
	if o3 == 0 && OnSegment(p3, p1, p4) {
		return true
	}
	if o4 == 0 && OnSegment(p3, p2, p4) {
		return true
	}
	return false
}

// Find orientation of the triplet points
// 1 -> c1, c2, c3 are colinear
// 2 -> Clockwise
// 3 -> Counterclockwise
func Orientation(c1 Point, c2 Point, c3 Point) int {
	val := (c2.y-c1.y)*(c3.x-c2.x) - (c2.x-c1.x)*(c3.y-c2.y)

	if val == 0 {
		return 0
	} else if val > 0 {
		return 1
	} else {
		return 2
	}
}

// Checks if point c2 lies on the line segment between c1 and c3
func OnSegment(c1 Point, c2 Point, c3 Point) bool {
	if c2.x <= math.Max(c1.x, c3.x) && c2.x >= math.Min(c1.x, c3.x) &&
		c2.y <= math.Max(c1.y, c3.y) && c2.y >= math.Min(c1.y, c3.y) {
		return true
	}
	return false
}

func FindLongestBlockChain() []Block {
	endNodes := FindLastBlockOfLongestChain()
	if len(endNodes) > 1 {
		// TODO : for edge case when multiple longest branches are present
		// we will select at random for now
		randNode := SelectRandomBranch(endNodes)
		return FindBlockChainPath(randNode)
	}
	return FindBlockChainPath(endNodes[0])
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
	return path
}

// TODO: INCOMPLETE?
// send out the block information to peers in the connected network of miners
func SendBlockInfo(block Block) error {

	replyStr := ""

	for key, miner := range connectedMiners {

		err := miner.Cli.Call("MinerKey.ReceiveBlock", block, replyStr)

		if err != nil {
			delete(connectedMiners, key)
		}

	}
	return errors.New("Parse error")
}

// once information about a block is received unpack that message and update ink-miner
func (minerKey *MinerKey) ReceiveBlock(block *Block, reply *string) error {
	receivedBlock := *block
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
		fmt.Println("Block exists within the blockchain")
	} else {
		return errors.New("failed to validate hash of a previous block")
	}

	// Check if received block is a No-Op or Op block based on length of operations
	// 1 = No-Op
	// 2 = Op
	var i int = 0
	if len(operations) == 0 {
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
			for _, operation := range receivedBlock.SetOPs {
				ValidateOperation(operation)
			}
		} else {
			return errors.New("No-op block proof of work does not match the zeroes of nonce")
		}
	}

	// After all validations pass, we set properties of block, append to blockchain and send to network
	receivedBlock.PathLength = previousBlock.PathLength + 1
	receivedBlock.PreviousBlock = previousBlock
	receivedBlock.InkBank = previousBlock.TotalInkAmount
	receivedBlock.TotalInkAmount = receivedBlock.InkBank // TODO: minus any operations performed for the generator of this block
	previousBlock.IsEndBlock = false

	if i == 1 {
		receivedBlock.TotalInkAmount = receivedBlock.TotalInkAmount + settings.InkPerNoOpBlock
	} else {
		receivedBlock.TotalInkAmount = receivedBlock.TotalInkAmount + settings.InkPerOpBlock
	}

	blockList = append(blockList, receivedBlock)
	err := SendBlockInfo(receivedBlock)

	return err
}

// Floods the network of miners with Operations
func SendOperation(operation Operation) {

	operationsHistory = append(operationsHistory, operation.UniqueID)
	operations = append(operations, operation)

	reply := ""

	for key, miner := range connectedMiners {

		err := miner.Cli.Call("MinerKey.ReceiveOperation", operation, reply)

		if err != nil {
			delete(connectedMiners, key)
		}

	}
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
func ComputeTrailingZeroes(hash string, num uint8) bool {
	var numZeroesStr = ""
	for i := 1; i <= int(num); i++ {
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
func CheckPreviousBlock(hash string) (*Block, bool) {
	var blockPtr *Block
	for i, block := range blockList {
		if block.Hash == hash {
			blockPtr = &blockList[i]
			return blockPtr, true
		}
		continue
	}
	return blockPtr, false
}

// call this for op-blocks to validate the op-block
func ValidateOperation(operation Operation) error {

	// made a dummy private key but it should correspond to blockartlib shape added?
	// need clarification from you guys
	// Check that each operation in the block has a valid signature

	if ecdsa.Verify(&operation.ArtNodePubKey, []byte(GlobalHash), operation.OPSigR, operation.OPSigS) {
		fmt.Println("op-sig is valid .... continuing validation")
	} else {
		return errors.New("failed to validate operation signature")
	}

	if operation.OpType == "Delete" {
		for _, doneOp := range operationsHistory {
			if operation.UniqueID == doneOp {
				fmt.Println("Delete operation validation sucess, shape to be deleted exists")
			} else {
				return errors.New("Delete operation could not find a shape previously added")
			}
		}
	}

	if operation.OpType == "Add" {
		for _, doneOp := range operationsHistory {
			if operation.UniqueID == doneOp {
				return errors.New("Duplicate add operation of same shape")
			} else {
				fmt.Println("identical signature could not be found, add shape validation sucess")
			}
		}
	}

	// checking ink amount
	for _, block := range blockList {
		if block.MinerPubKey == operation.ArtNodePubKey {
			minerCurrentInk := GetInkAmount(&block)
			minerCurrentInk = minerCurrentInk - operation.OpInkCost
			if minerCurrentInk < 0 {
				return errors.New("the total operation cost exceeds ink-miner supply")
			} else {
				continue
			}
		} else {
			return errors.New("ArtNodePubKey's associated MinerPubKey could not be found")
		}
	}
	return nil
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

		if _, ok := connectedMiners[addrSet[i].String()]; !ok {
			fmt.Println("hello")
			cli, err := rpc.Dial("tcp", addrSet[i].String())

			var reply MinerInfo
			err = cli.Call("MinerKey.RegisterMiner", MinerInfo{Address: currentAddress, Key: currentPubKey}, &reply)
			HandleError(err)

			connectedMiners[reply.Address.String()] = Miner{Address: reply.Address, Key: reply.Key, Cli: cli}
		}

	}
}

// Goroutine to get nodes (if number of connected miners go below min-num-miner-connections, get more from server)
func GetNodes(cli *rpc.Client) {
	for {
		var addrSet []net.Addr
		err := cli.Call("RServer.GetNodes", pubKey, &addrSet)
		HandleError(err)

		ConnectToMiners(addrSet, minerAddr, pubKey)

		time.Sleep(30000 * time.Millisecond)
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

// Updates everyone's blockchain to only include the LONGEST blockchain
func SyncMinersLongestChain() {
	for {
		if len(connectedMiners) > 0 {

			longestBlockChain := FindLongestBlockChain()
			blockList = longestBlockChain
			var reply string

			for _, miner := range connectedMiners {
				miner.Cli.Call("Minerkey.UpdateLongestBlockChain", LongestBlockChain{BlockChain: longestBlockChain}, &reply)
			}
		}

		time.Sleep(30 * time.Second)
	}
}

// Gets the Max Validate Number for a block using set of operations inside it.
func GetMaxValidateNum(operations []Operation) int {
	maxValidateNum := 0
	for i := 0; i < len(operations); i++ {
		if operations[i].ValidateNum > maxValidateNum {
			maxValidateNum = operations[i].ValidateNum
		}
	}
	return maxValidateNum
}

// Checks whether or not operations are validated or not (check validateNum against the block)
func CheckOperationValidation(uniqueID string) {
	for {
		// blockToCheck is the block that contains the checked Operation
		blockToCheck := Block{}
		opToCheck := Operation{}

		// Get the block that we need (where the operation is in)
		for i := 0; i < len(blockList); i++ {
			opList := blockList[i].SetOPs

			for j := 0; j < len(blockList[i].SetOPs); j++ {

				if blockList[i].SetOPs[j].UniqueID == uniqueID {
					blockToCheck = blockList[i]
					opToCheck = blockList[i].SetOPs[j]
					break
				}
			}
		}

		// Tail block of the blockChain that consists of blockToCheck
		endBlock := Block{}

		// Goes through each end blocks to find the one that consists blockToCheck
		for k := len(blockList); k > 0; k-- {

			if blockList[k].IsEndBlock == true {

				blockChain := FindBlockChainPath(blockList[k])
				for l := 0; l < len(blockChain); l++ {

					if blockChain[l].Hash == blockToCheck.Hash {
						endBlock = blockList[k]
						break
					}
				}
			}
		}

		if endBlock.Hash != "" {
			// Stop the infinite for loop when we find something to send back to the Art Node
			if endBlock.PathLength-blockToCheck.PathLength >= opToCheck.ValidateNum {
				break
			}
		}

		time.Sleep(5 * time.Second)
	}
}

// Used to compare public keys
func IsPublicKeySame(incomingPubKey ecdsa.PublicKey) bool {
	data := []byte("This is private key")
	r, s, _ := ecdsa.Sign(rand.Reader, &privKey, data)

	if ecdsa.Verify(&incomingPubKey, data, r, s) {
		fmt.Println("This is the same")
		return true
	}

	fmt.Println("This is not the same")
	return false
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
	for _, op := range operationsHistory {
		if op.UniqueID == shapeHash {
			*operation = op
			return nil
		}
	}
	*operation = Operation{}
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

	fmt.Println(minerAddr)

	cli, _ := rpc.Dial("tcp", serverAddr)

	minerKey := new(MinerKey)
	rpc.Register(minerKey)

	artKey := new(ArtKey)
	rpc.Register(artKey)

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

	fmt.Println(blockList)

	go InitHeartbeat(cli, pubKey, settings.HeartBeat)

	go rpc.Accept(lis)

	var addrSet []net.Addr
	err = cli.Call("RServer.GetNodes", pubKey, &addrSet)
	HandleError(err)

	ConnectToMiners(addrSet, minerAddr, pubKey)

	go GetNodes(cli)

	GenerateBlock(settings)
}

func HandleError(err error) {
	if err != nil {
		fmt.Println(err)
	}
}
