/*

This package specifies the application's interface to the the BlockArt
library (blockartlib) to be used in project 1 of UBC CS 416 2017W2.

*/

package blockartlib

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/gob"
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
	"unicode"
)

// Represents a type of shape in the BlockArt system.
type ShapeType int

const (
	// Path shape.
	PATH ShapeType = iota

	// Circle shape (extra credit).
	// CIRCLE
)

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
	ArtNodeID     int
	ShapeType     ShapeType
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
	DeleteUniqueID string
	PathShape      string
}

var canvasSettings CanvasSettings

// Settings for a canvas in BlockArt.
type CanvasSettings struct {
	// Canvas dimensions
	CanvasXMax uint32
	CanvasYMax uint32
}

// A CanvasObj will have the information about miners on the canvas
type CanvasObj struct {
	MinerAddress string
	PrivateKey   ecdsa.PrivateKey
	MinerCli     *rpc.Client
	ArtNodeID    int
}

type ArtNodeKey struct {
	ArtNodeID int
	R, S      *big.Int
	Hash      []byte
}

type Line struct {
	Start Point
	End   Point
}

type Point struct {
	X float64
	Y float64
}

// Settings for an instance of the BlockArt project/network.
type MinerNetSettings struct {
	// Hash of the very first (empty) block in the chain.
	GenesisBlockHash string

	// The minimum number of ink miners that an ink miner should be
	// connected to. If the ink miner dips below this number, then
	// they have to retrieve more nodes from the server using
	// GetNodes().
	MinNumMinerConnections uint8

	// Mining ink reward per op and no-op blocks (>= 1)
	InkPerOpBlock   uint32
	InkPerNoOpBlock uint32

	// Number of milliseconds between heartbeat messages to the server.
	HeartBeat uint32

	// Proof of work difficulty: number of zeroes in prefix (>=0)
	PoWDifficultyOpBlock   uint8
	PoWDifficultyNoOpBlock uint8

	// Canvas settings
	CanvasSettings CanvasSettings
}

////////////////////////////////////////////////////////////////////////////////////////////
// <ERROR DEFINITIONS>

// These type definitions allow the application to explicitly check
// for the kind of error that occurred. Each API call below lists the
// errors that it is allowed to raise.
//
// Also see:
// https://blog.golang.org/error-handling-and-go
// https://blog.golang.org/errors-are-values

// Contains address IP:port that art node cannot connect to.
type DisconnectedError string

func (e DisconnectedError) Error() string {
	return fmt.Sprintf("BlockArt: cannot connect to [%s]", string(e))
}

// Contains amount of ink remaining.
type InsufficientInkError uint32

func (e InsufficientInkError) Error() string {
	return fmt.Sprintf("BlockArt: Not enough ink to addShape [%d]", uint32(e))
}

// Contains the offending svg string.
type InvalidShapeSvgStringError string

func (e InvalidShapeSvgStringError) Error() string {
	return fmt.Sprintf("BlockArt: Bad shape svg string [%s]", string(e))
}

// Contains the offending svg string.
type ShapeSvgStringTooLongError string

func (e ShapeSvgStringTooLongError) Error() string {
	return fmt.Sprintf("BlockArt: Shape svg string too long [%s]", string(e))
}

// Contains the bad shape hash string.
type InvalidShapeHashError string

func (e InvalidShapeHashError) Error() string {
	return fmt.Sprintf("BlockArt: Invalid shape hash [%s]", string(e))
}

// Contains the bad shape hash string.
type ShapeOwnerError string

func (e ShapeOwnerError) Error() string {
	return fmt.Sprintf("BlockArt: Shape owned by someone else [%s]", string(e))
}

// Empty
type OutOfBoundsError struct{}

func (e OutOfBoundsError) Error() string {
	return fmt.Sprintf("BlockArt: Shape is outside the bounds of the canvas")
}

// Contains the hash of the shape that this shape overlaps with.
type ShapeOverlapError string

func (e ShapeOverlapError) Error() string {
	return fmt.Sprintf("BlockArt: Shape overlaps with a previously added shape [%s]", string(e))
}

// Contains the invalid block hash.
type InvalidBlockHashError string

func (e InvalidBlockHashError) Error() string {
	return fmt.Sprintf("BlockArt: Invalid block hash [%s]", string(e))
}

// SELF MADE: Contains Private/Public Key that is not validated by the Miner.
type InvalidKeyError string

func (e InvalidKeyError) Error() string {
	return fmt.Sprintf("BlockArt: Public Key is not validated", string(e))
}

// </ERROR DEFINITIONS>
////////////////////////////////////////////////////////////////////////////////////////////

// Represents a canvas in the system.
type Canvas interface {
	// Adds a new shape to the canvas.
	// Can return the following errors:
	// - DisconnectedError
	// - InsufficientInkError
	// - InvalidShapeSvgStringError
	// - ShapeSvgStringTooLongError
	// - ShapeOverlapError
	// - OutOfBoundsError
	AddShape(validateNum uint8, shapeType ShapeType, shapeSvgString string, fill string, stroke string) (shapeHash string, blockHash string, inkRemaining uint32, err error)

	// Returns the encoding of the shape as an svg string.
	// Can return the following errors:
	// - DisconnectedError
	// - InvalidShapeHashError
	GetSvgString(shapeHash string) (svgString string, err error)

	// Returns the amount of ink currently available.
	// Can return the following errors:
	// - DisconnectedError
	GetInk() (inkRemaining uint32, err error)

	// Removes a shape from the canvas.
	// Can return the following errors:
	// - DisconnectedError
	// - ShapeOwnerError
	DeleteShape(validateNum uint8, shapeHash string) (inkRemaining uint32, err error)

	// Retrieves hashes contained by a specific block.
	// Can return the following errors:
	// - DisconnectedError
	// - InvalidBlockHashError
	GetShapes(blockHash string) (shapeHashes []string, err error)

	// Returns the block hash of the genesis block.
	// Can return the following errors:
	// - DisconnectedError
	GetGenesisBlock() (blockHash string, err error)

	// Retrieves the children blocks of the block identified by blockHash.
	// Can return the following errors:
	// - DisconnectedError
	// - InvalidBlockHashError
	GetChildren(blockHash string) (blockHashes []string, err error)

	// Closes the canvas/connection to the BlockArt network.
	// - DisconnectedError
	CloseCanvas() (inkRemaining uint32, err error)
}

// The constructor for a new Canvas object instance. Takes the miner's
// IP:port address string and a public-private key pair (ecdsa private
// key type contains the public key). Returns a Canvas instance that
// can be used for all future interactions with blockartlib.
//
// The returned Canvas instance is a singleton: an application is
// expected to interact with just one Canvas instance at a time.
//
// Can return the following errors:
// - DisconnectedError
func OpenCanvas(minerAddr string, privKey ecdsa.PrivateKey) (canvas Canvas, setting CanvasSettings, err error) {
	gob.Register(&net.TCPAddr{})
	gob.Register(&elliptic.CurveParams{})

	mrand.Seed(time.Now().UnixNano())

	artNodeID := mrand.Intn(10000-1) + 1

	// pubKey := privKey.PublicKey

	cli, err := rpc.Dial("tcp", minerAddr)

	if err != nil {
		return nil, setting, DisconnectedError(minerAddr)
	}

	r, s, _ := ecdsa.Sign(rand.Reader, &privKey, []byte("This is the private key!"))

	err = cli.Call("ArtKey.ValidateKey", ArtNodeKey{ArtNodeID: artNodeID, R: r, S: s, Hash: []byte("This is the private key!")}, &setting)

	if err != nil {
		return nil, setting, DisconnectedError(minerAddr)
	}

	// provide canvas with a mineraddress and a privatekey
	canvasObj := CanvasObj{
		MinerAddress: minerAddr,
		PrivateKey:   privKey,
		MinerCli:     cli,
		ArtNodeID:    artNodeID,
	}

	canvasSettings = setting

	return canvasObj, setting, err
}

// Closes the canvas/connection to the BlockArt network.
// - DisconnectedError
func (canvasObj CanvasObj) CloseCanvas() (inkRemaining uint32, err error) {
	var reply uint32

	err = canvasObj.MinerCli.Call("ArtKey.GetInk", "", &reply)
	if err != nil {
		return uint32(0), DisconnectedError(canvasObj.MinerAddress)
	}

	canvasObj.MinerCli.Close()

	return reply, err
}

// Retrieves the children blocks of the block identified by blockHash.
// Can return the following errors:
// - DisconnectedError
// - InvalidBlockHashError
func (canvasObj CanvasObj) GetChildren(blockHash string) (blockHashes []string, err error) {
	address := canvasObj.MinerAddress

	var reply []string
	err = canvasObj.MinerCli.Call("ArtKey.GetChildren", blockHash, &reply)
	if err != nil {
		if err.Error() == "Hash does not exist" {
			return nil, InvalidBlockHashError(blockHash)
		}
		return nil, DisconnectedError(address)
	}

	return reply, nil
}

// Returns the block hash of the genesis block.
// Can return the following errors:
// - DisconnectedError
func (canvasObj CanvasObj) GetGenesisBlock() (blockHash string, err error) {
	address := canvasObj.MinerAddress

	var reply string
	err = canvasObj.MinerCli.Call("ArtKey.GetGenesisBlock", "", &reply)
	if err != nil {
		return "", DisconnectedError(address)
	}

	return reply, nil
}

func (canvasObj CanvasObj) AddShape(validateNum uint8, shapeType ShapeType, shapeSvgString string, fill string, stroke string) (shapeHash string, blockHash string, inkRemaining uint32, err error) {

	// - DisconnectedError
	// - InsufficientInkError

	// For parsing shapeSvgString:  https://piazza.com/class/jbyh5bsk4ez3cn?cid=416

	// address := canvasObj.MinerAddress

	// - ShapeSvgStringTooLongError
	if !HandleSvgStringLength(shapeSvgString) {
		return "", "", inkRemaining, ShapeSvgStringTooLongError(shapeSvgString)
	}

	svgArray := strings.Split(shapeSvgString, " ")

	// - InvalidShapeSvgStringError TODO: when fill or stroke is empty https://piazza.com/class/jbyh5bsk4ez3cn?cid=414
	if checkValidFillAndStroke(fill, stroke) == false {
		return "", "", inkRemaining, InvalidShapeSvgStringError(shapeSvgString)
	}

	if checkValidSvgArray(svgArray) == false {
		return "", "", inkRemaining, InvalidShapeSvgStringError(shapeSvgString)
	}

	// - OutOfBoundsError
	boundCheck := GetCoordinates(svgArray)

	if !BoundCheck(boundCheck) {
		boundsErr := OutOfBoundsError{}
		return "", "", inkRemaining, OutOfBoundsError(boundsErr)
	}

	// calculate amount of ink that this shape will use
	inkReq := CalcInkUsed(boundCheck, fill)

	nodePrivKey := canvasObj.PrivateKey

	r, s, _ := ecdsa.Sign(rand.Reader, &nodePrivKey, []byte("This is the private key!"))

	shapeHash = r.String() + s.String()

	// shape hash will only take on unique value for r, but for op-sig validation we should pass
	// in r and s but we will only need to look at r values for shapeHash validation?

	//set the coordinates

	pathShape := ConstructSvgString(shapeType, shapeSvgString, fill, stroke)

	linesToDraw := GetCoordinates(svgArray)

	var reply Block
	err = canvasObj.MinerCli.Call("ArtKey.AddShape", Operation{
		ArtNodeID:      canvasObj.ArtNodeID,
		UniqueID:       shapeHash,
		ArtNodePubKey:  canvasObj.PrivateKey.PublicKey,
		OpInkCost:      inkReq,
		OPSigR:         r,
		OPSigS:         s,
		OpType:         "Add",
		ValidateNum:    int(validateNum),
		ShapeType:      shapeType,
		ShapeSvgString: shapeSvgString,
		Fill:           fill,
		Stroke:         stroke,
		Lines:          linesToDraw,
		PathShape:      pathShape,
	}, &reply)

	if err != nil {

		if err.Error() == "connection is shut down" {
			return "", "", inkRemaining, DisconnectedError(canvasObj.MinerAddress)
		} else {
			fmt.Println("AddShape RPC: ", err.Error())
			return "", "", inkRemaining, err
		}
	}

	if reply.Hash == "" {
		return "", "", 0, errors.New("Timed out, operation not validated")
	}

	return shapeHash, reply.Hash, reply.TotalInkAmount, err
}

// Returns the encoding of the shape as an svg string.
// Can return the following errors:
// - DisconnectedError
// - InvalidShapeHashError
func (canvasObj CanvasObj) GetSvgString(shapeHash string) (svgString string, err error) {
	address := canvasObj.MinerAddress

	var reply Operation
	err = canvasObj.MinerCli.Call("ArtKey.GetOperationWithShapeHash", shapeHash, &reply)
	if err != nil {
		if err.Error() == "Does not exist" {
			return "", InvalidShapeHashError(shapeHash)
		}
		return "", DisconnectedError(address)
	}

	shapeType := reply.ShapeType
	fill := reply.Fill
	stroke := reply.Stroke
	dString := reply.ShapeSvgString

	svgString = ConstructSvgString(shapeType, dString, fill, stroke)

	return svgString, err
}

// Returns the amount of ink currently available.
// Can return the following errors:
// - DisconnectedError
func (canvasObj CanvasObj) GetInk() (inkRemaining uint32, err error) {
	var reply uint32
	err = canvasObj.MinerCli.Call("ArtKey.GetInk", "", &reply)
	if err != nil {
		return uint32(0), DisconnectedError(canvasObj.MinerAddress)
	}
	return reply, err
}

// Removes a shape from the canvas.
// Can return the following errors:
// - DisconnectedError
// - ShapeOwnerError
// - ShapeOwnerError is returned if this application did not create the shape with shapeHash (or if no shape exists with shapeHash).
func (canvasObj CanvasObj) DeleteShape(validateNum uint8, shapeHash string) (inkRemaining uint32, err error) {
	address := canvasObj.MinerAddress
	client := canvasObj.MinerCli

	client.Call("ArtKey.DeleteShape", shapeHash, &inkRemaining)
	if err != nil {
		if err.Error() == "Does not exist" || err.Error() == "Did not create" {
			return 0, ShapeOwnerError(shapeHash)
		}
		return 0, DisconnectedError(address)
	}

	r, s, _ := ecdsa.Sign(rand.Reader, &canvasObj.PrivateKey, []byte("This is the private key!"))

	var replyOp Operation
	err = canvasObj.MinerCli.Call("ArtKey.GetOperationWithShapeHash", shapeHash, &replyOp)
	if err != nil {
		if err.Error() == "connection is shut down" {
			return 0, DisconnectedError(address)
		} else {
			return 0, ShapeOwnerError(shapeHash)
		}
	}

	shapeType := replyOp.ShapeType
	dString := replyOp.ShapeSvgString
	cost := replyOp.OpInkCost

	deleteOperation := Operation{
		ArtNodeID:      canvasObj.ArtNodeID,
		UniqueID:       r.String() + s.String(),
		DeleteUniqueID: shapeHash,
		ArtNodePubKey:  canvasObj.PrivateKey.PublicKey,
		ValidateNum:    int(validateNum),
		OPSigR:         r,
		OPSigS:         s,
		OpType:         "Delete",
		Fill:           "white",
		Stroke:         "white",
		ShapeType:      shapeType,
		ShapeSvgString: dString,
		OpInkCost:      cost,
	}

	var reply bool
	client.Call("ArtKey.ValidateDelete", deleteOperation, &reply)

	if err != nil {

		if err.Error() == "connection is shut down" {
			return 0, DisconnectedError(address)
		} else {
			return 0, err
		}
	}

	if reply {
		return inkRemaining, nil
	}

	return 0, errors.New("Timed out, operation not validated")
}

// Retrieves hashes contained by a specific block.
// Can return the following errors:
// - DisconnectedError
// - InvalidBlockHashError
func (canvasObj CanvasObj) GetShapes(blockHash string) (shapeHashes []string, err error) {
	address := canvasObj.MinerAddress

	err = canvasObj.MinerCli.Call("ArtKey.GetShapes", blockHash, &shapeHashes)
	if err != nil {
		if err.Error() == "Invalid shape hash" {
			return shapeHashes, InvalidBlockHashError(blockHash)
		}
		return shapeHashes, DisconnectedError(address)
	}

	return shapeHashes, nil
}

///////////////////////////////// HELPER FUNCTIONS BELOW

// Retrieves all the PATH shapes from Ink Miner's local longest blockchain and creates an HTML file of the Canvas
func CreateCanvasHTML(paths []string, cSettings CanvasSettings) {

	f, err := os.Create("Canvas.html")
	HandleError(err)

	svgPath := "<svg height=\"" + strconv.Itoa(int(cSettings.CanvasYMax)) + "\" width=\"" + strconv.Itoa(int(cSettings.CanvasXMax)) + "\">"

	for i := 0; i < len(paths); i++ {
		svgPath = svgPath + paths[i]
	}
	svgPath = svgPath + "</svg>"

	f.Write([]byte(svgPath))
}

func ConstructSvgString(shapeType ShapeType, svgString string, fill string, stroke string) string {
	// <path d="M 0 0 L 20 20" stroke="red" fill="transparent"/>
	if shapeType == PATH {
		return fmt.Sprintf("<path d=\"%s\" stroke=\"%s\" fill=\"%s\" />", svgString, stroke, fill)
	}

	return ""
}

// svg string can be at most 128 characters in string length
func HandleSvgStringLength(svgstr string) bool {
	if len(svgstr) > 128 {
		return false
	}
	return true
}

// gets the corrdinates for the operation
func GetCoordinates(svgArray []string) []Line {
	// Uppercase = absolute
	// Lowercase = relative
	lines := []Line{}
	origin := Point{}
	startPt := Point{}
	endPt := Point{}

	for i := 0; i < len(svgArray); i++ {
		str := svgArray[i]

		switch str {
		case "M":
			xp := ToFloat64(svgArray[i+1])
			yp := ToFloat64(svgArray[i+2])
			startPt = Point{X: xp, Y: yp}
			origin = startPt
			i = i + 2
		case "m":
			xp := ToFloat64(svgArray[i+1]) + startPt.X
			yp := ToFloat64(svgArray[i+2]) + startPt.Y
			startPt = Point{X: xp, Y: yp}
			origin = startPt
			i = i + 2
		case "L":
			xp := ToFloat64(svgArray[i+1])
			yp := ToFloat64(svgArray[i+2])
			endPt = Point{X: xp, Y: yp}

			lines = append(lines, Line{Start: startPt, End: endPt})
			startPt = endPt
			i = i + 2
		case "l":
			xp := ToFloat64(svgArray[i+1]) + startPt.X
			yp := ToFloat64(svgArray[i+2]) + startPt.Y
			endPt = Point{X: xp, Y: yp}

			lines = append(lines, Line{Start: startPt, End: endPt})
			startPt = endPt
			i = i + 2
		case "H":
			xp := ToFloat64(svgArray[i+1])
			yp := startPt.Y
			endPt = Point{X: xp, Y: yp}

			lines = append(lines, Line{Start: startPt, End: endPt})
			startPt = endPt
			i = i + 1
		case "h":
			xp := ToFloat64(svgArray[i+1]) + startPt.X
			yp := startPt.Y
			endPt = Point{X: xp, Y: yp}

			lines = append(lines, Line{Start: startPt, End: endPt})
			startPt = endPt
			i = i + 1
		case "V":
			xp := startPt.X
			yp := ToFloat64(svgArray[i+1])
			endPt = Point{X: xp, Y: yp}

			lines = append(lines, Line{Start: startPt, End: endPt})
			startPt = endPt
			i = i + 1
		case "v":
			xp := startPt.X
			yp := ToFloat64(svgArray[i+1]) + startPt.Y
			endPt = Point{X: xp, Y: yp}

			lines = append(lines, Line{Start: startPt, End: endPt})
			startPt = endPt
			i = i + 1
		case "Z":
			endPt = origin
			lines = append(lines, Line{Start: startPt, End: endPt})
		case "z":
			endPt = origin
			lines = append(lines, Line{Start: startPt, End: endPt})
		}
	}

	return lines
}

func ToFloat64(str string) float64 {
	result, _ := strconv.ParseFloat(str, 64)
	return result
}

func CalcInkUsed(lines []Line, fill string) uint32 {

	var inkTotal float64

	inkTotal = 0

	for i := 0; i < len(lines); i++ {
		xstart := lines[i].Start
		xspos := xstart.X
		ystart := lines[i].Start
		yspos := ystart.Y
		xend := lines[i].End
		xepos := xend.X
		yend := lines[i].End
		yepos := yend.Y

		distance := math.Pow(float64(xspos)-float64(xepos), 2) + math.Pow(float64(yspos)-float64(yepos), 2)
		rootDis := math.Sqrt(distance)

		inkTotal = rootDis + inkTotal

	}

	if fill != "transparent" {
		points := []Point{}
		for i := 0; i < len(lines); i++ {
			points = append(points, lines[i].Start)

		}
		areaInk := PolygonArea(points)
		areaInk = math.Abs(areaInk)
		inkTotal = areaInk + inkTotal
	}

	inkTotal = round(inkTotal)

	return uint32(inkTotal)

}

func PolygonArea(points []Point) float64 {
	first := points[0]
	last := first
	var area float64

	for i, _ := range points {
		next := points[i]
		area = area + next.X*last.Y - last.X*next.Y
		last = next
	}
	return area / 2
}

func round(a float64) float64 {
	if a < 0 {
		return math.Ceil(a - 0.5)
	}
	return math.Floor(a + 0.5)
}

func checkValidFillAndStroke(fill string, stroke string) bool {
	if fill == "transparent" && stroke == "transparent" {
		return false
	}

	if fill == "" || stroke == "" {
		return false
	}

	return true
}

func checkValidSvgArray(svgArray []string) bool {

	// Check if "M" is always the first element in string array
	if svgArray[0] != "M" {
		return false
	}

	// M or m --> 2 ints
	// L or l --> 2 ints
	// V or v --> 1 int
	// H or h --> 1 int
	// Z or z --> 0 int
	restrictionInt := 0
	for i := 0; i < len(svgArray); i++ {

		if restrictionInt > 0 {
			if isNum(svgArray[i]) {
				restrictionInt = restrictionInt - 1
			} else {
				return false
			}
		} else {
			if isLetter(svgArray[i]) {
				switch svgArray[i] {
				case "M":
					restrictionInt = 2
				case "m":
					restrictionInt = 2
				case "L":
					restrictionInt = 2
				case "l":
					restrictionInt = 2
				case "V":
					restrictionInt = 1
				case "v":
					restrictionInt = 1
				case "H":
					restrictionInt = 1
				case "h":
					restrictionInt = 1
				case "Z":
					restrictionInt = 0
				case "z":
					restrictionInt = 0
				default:
					return false
				}
			} else {
				return false
			}
		}
	}

	if restrictionInt > 0 {
		return false
	} else {
		return true
	}
}

func isLetter(svgLetter string) bool {
	for _, letter := range svgLetter {
		if !unicode.IsLetter(letter) {
			return false
		}
	}
	return true
}

func isNum(svgNum string) bool {
	for _, num := range svgNum {
		if !unicode.IsNumber(num) {
			return false
		}
	}
	return true
}

// checks the boundary settings for the position of shape, EX "M 0 10 H 20" checks 0 and 10
func BoundCheck(lines []Line) bool {
	for i := 0; i < len(lines); i++ {
		xstart := lines[i].Start
		xspos := xstart.X
		ystart := lines[i].Start
		yspos := ystart.Y
		xend := lines[i].End
		xepos := xend.X
		yend := lines[i].End
		yepos := yend.Y

		if xspos > float64(canvasSettings.CanvasXMax) {
			return false
		}
		if yspos > float64(canvasSettings.CanvasYMax) {
			return false
		}
		if xepos > float64(canvasSettings.CanvasXMax) {
			return false
		}
		if yepos > float64(canvasSettings.CanvasYMax) {
			return false
		}
		if xspos < float64(0) {
			return false
		}
		if yspos < float64(0) {
			return false
		}
		if xepos < float64(0) {
			return false
		}
		if yepos < float64(0) {
			return false
		}
	}
	return true
}

func FindPaths(canvas Canvas, blockHash string, path []string, res *[][]string) error {

	children, err := canvas.GetChildren(blockHash)
	HandleError(err)
	if len(children) == 0 {
		*res = append(*res, path)
		return nil
	} else {
		pathTemp := path
		for _, c := range children {
			pathTemp = append(path, c)
			FindPaths(canvas, c, pathTemp, res)
		}
	}
	return err
}

func GetAllSVGs(canvas Canvas) ([]string, error) {
	res := [][]string{}
	var resPtr *[][]string
	resPtr = &res
	path := []string{}

	genHash, err := canvas.GetGenesisBlock()
	HandleError(err)

	// fills resPtr with arrays of all paths
	findPathError := FindPaths(canvas, genHash, path, resPtr)

	longestPath := []string{}
	for _, tempPath := range res {
		if len(tempPath) > len(longestPath) {
			longestPath = tempPath
		}
	}

	shapeHashes := []string{}
	for _, blockHash := range longestPath {
		currBlockHashes, err := canvas.GetShapes(blockHash)
		HandleError(err)
		shapeHashes = append(shapeHashes, currBlockHashes...)
	}

	SVGs := []string{}
	for _, sHash := range shapeHashes {
		currSVG, err := canvas.GetSvgString(sHash)
		HandleError(err)
		SVGs = append(SVGs, currSVG)
	}
	return SVGs, findPathError
}

func HandleError(err error) {
	if err != nil {
		fmt.Println(err)
	}
}
