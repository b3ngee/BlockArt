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
	"fmt"
	"math"
	"math/big"
	"net"
	"net/rpc"
	"strconv"
	"strings"
)

// Represents a type of shape in the BlockArt system.
type ShapeType int

const (
	// Path shape.
	PATH ShapeType = iota

	// Circle shape (extra credit).
	// CIRCLE
)

type Operation struct {
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
	xStart         float64
	xEnd           float64
	yStart         float64
	yEnd           float64
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
}

type ArtNodeKey struct {
	R, S *big.Int
	Hash []byte
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

	// pubKey := privKey.PublicKey

	cli, err := rpc.Dial("tcp", minerAddr)

	if err != nil {
		return nil, setting, DisconnectedError(minerAddr)
	}

	r, s, _ := ecdsa.Sign(rand.Reader, &privKey, []byte("This is the Private Key."))

	err = cli.Call("ArtKey.ValidateKey", ArtNodeKey{R: r, S: s, Hash: []byte("This is the Private Key.")}, &setting)
	if err != nil {
		return nil, setting, DisconnectedError(minerAddr)
	}

	// provide canvas with a mineraddress and a privatekey
	canvasObj := CanvasObj{
		MinerAddress: minerAddr,
		PrivateKey:   privKey,
		MinerCli:     cli}

	return canvasObj, setting, err
}

// Closes the canvas/connection to the BlockArt network.
// - DisconnectedError
func (canvasObj CanvasObj) CloseCanvas() (inkRemaining uint32, err error) {
	return inkRemaining, err
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
		return nil, DisconnectedError(address)
	}
	if len(reply) > 0 && reply[0] == "INVALID" {
		return nil, InvalidBlockHashError(blockHash)
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
	// - InvalidShapeSvgStringError TODO: when fill or stroke is empty https://piazza.com/class/jbyh5bsk4ez3cn?cid=414

	// For parsing shapeSvgString:  https://piazza.com/class/jbyh5bsk4ez3cn?cid=416

	address := canvasObj.MinerAddress

	// - ShapeSvgStringTooLongError
	if !HandleSvgStringLength(shapeSvgString) {
		return "", "", inkRemaining, ShapeSvgStringTooLongError(shapeSvgString)
	}

	// - OutOfBoundsError
	svgArray := strings.Split(shapeSvgString, " ")
	if !BoundCheck(svgArray) {
		boundsErr := OutOfBoundsError{}
		return "", "", inkRemaining, OutOfBoundsError(boundsErr)
	}

	// calculate amount of ink that this shape will use
	inkReq := uint32(CalcInkUsed(svgArray))
	fmt.Println(inkReq)

	nodePrivKey := canvasObj.PrivateKey
	var reply string

	r, s, _ := ecdsa.Sign(rand.Reader, &nodePrivKey, []byte("This is the Private Key."))

	shapeHash = r.String() + s.String()

	// shape hash will only take on unique value for r, but for op-sig validation we should pass
	// in r and s but we will only need to look at r values for shapeHash validation?

	//set the coordinates

	x, xe, y, ye := GetCoordinates(svgArray)

	err = canvasObj.MinerCli.Call("ArtKey.AddShape", Operation{
		UniqueID:       shapeHash,
		OpInkCost:      inkReq,
		OPSigR:         r,
		OPSigS:         s,
		OpType:         "Add",
		ValidateNum:    int(validateNum),
		ShapeType:      shapeType,
		ShapeSvgString: shapeSvgString,
		Fill:           fill,
		Stroke:         stroke,
		xStart:         x,
		xEnd:           xe,
		yStart:         y,
		yEnd:           ye,
	}, &reply)
	if err != nil {
		return "", "", inkRemaining, DisconnectedError(address)
	}

	return shapeHash, blockHash, inkRemaining, err
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
		return "", DisconnectedError(address)
	}
	if reply.UniqueID == "" {
		return "", InvalidShapeHashError(shapeHash)
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

	return inkRemaining, err
}

// Removes a shape from the canvas.
// Can return the following errors:
// - DisconnectedError
// - ShapeOwnerError
func (canvasObj CanvasObj) DeleteShape(validateNum uint8, shapeHash string) (inkRemaining uint32, err error) {
	return inkRemaining, err
}

// Retrieves hashes contained by a specific block.
// Can return the following errors:
// - DisconnectedError
// - InvalidBlockHashError
func (canvasObj CanvasObj) GetShapes(blockHash string) (shapeHashes []string, err error) {
	return shapeHashes, err
}

///////////////////////////////// HELPER FUNCTIONS BELOW

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
func GetCoordinates(svgArray []string) (float64, float64, float64, float64) {
	hor := svgArray[3]
	vert := svgArray[5]

	xstart, err := strconv.ParseInt(svgArray[1], 0, 32)
	HandleError(err)
	xend := int64(0)
	ystart, err := strconv.ParseInt(svgArray[2], 0, 32)
	HandleError(err)
	yend := int64(0)

	if hor == "H" && vert == "V" {
		hendstr := svgArray[4]
		hend, err := strconv.ParseInt(hendstr, 0, 32)
		HandleError(err)
		xend = hend
		vendstr := svgArray[6]
		vend, err := strconv.ParseInt(vendstr, 0, 32)
		HandleError(err)
		yend = vend

	} else {
		if hor == "H" {
			hendstr := svgArray[4]
			hend, err := strconv.ParseInt(hendstr, 0, 32)
			HandleError(err)
			xend = hend
			yend = -1
		} else {
			if hor == "V" {
				vendstr := svgArray[4]
				vend, err := strconv.ParseInt(vendstr, 0, 32)
				HandleError(err)
				yend = vend
				xend = -1
			}
		}
	}

	// then we know its a line
	hendstr := svgArray[4]
	hend, err := strconv.ParseInt(hendstr, 0, 32)
	HandleError(err)
	xend = hend

	vendstr := svgArray[5]
	vend, err := strconv.ParseInt(vendstr, 0, 32)
	HandleError(err)
	yend = vend

	return float64(xstart), float64(xend), float64(ystart), float64(yend)
}

func CalcInkUsed(svgArray []string) int64 {
	fill := svgArray[3]
	vert := svgArray[5]
	var totalInk int64 = 0

	if fill == "H" && vert == "V" {
		hfill := svgArray[4]
		hink, err := strconv.ParseInt(hfill, 0, 32)
		HandleError(err)
		vfill := svgArray[6]
		vink, err := strconv.ParseInt(vfill, 0, 32)
		HandleError(err)

		totalInk = vink * hink
		return totalInk
	} else {
		anyfill := svgArray[4]
		hOrVInk, err := strconv.ParseInt(anyfill, 0, 32)
		HandleError(err)
		return hOrVInk
	}

	// or the fill will be the line L

	startlinex := svgArray[1]
	x, err := strconv.ParseInt(startlinex, 0, 32)
	HandleError(err)
	startliney := svgArray[2]
	y, err := strconv.ParseInt(startliney, 0, 32)
	HandleError(err)
	endlinex := svgArray[4]
	xend, err := strconv.ParseInt(endlinex, 0, 32)
	HandleError(err)
	endliney := svgArray[5]
	yend, err := strconv.ParseInt(endliney, 0, 32)
	HandleError(err)
	distance := math.Pow(float64(x)-float64(xend), 2) + math.Pow(float64(y)-float64(yend), 2)
	rootDis := int64(math.Sqrt(distance))

	return rootDis

}

// checks the boundary settings for the position of shape, EX "M 0 10 H 20" checks 0 and 10
func BoundCheck(svgArray []string) bool {

	xInt := svgArray[1]
	yInt := svgArray[2]
	fill := svgArray[3]

	x, err := strconv.ParseInt(xInt, 0, 32)
	HandleError(err)
	y, err := strconv.ParseInt(yInt, 0, 32)
	HandleError(err)

	if x < 0 {
		return false
	}
	if y < 0 {
		return false
	}
	if x > int64(canvasSettings.CanvasXMax) {
		return false
	}
	if y > int64(canvasSettings.CanvasYMax) {
		return false
	}

	if fill == "H" {
		hDis := svgArray[4]
		xDis, err := strconv.ParseInt(hDis, 0, 32)
		HandleError(err)
		xEnd := xDis + x
		if xEnd > int64(canvasSettings.CanvasXMax) {
			return false
		} else {
			moreFill := svgArray[5]
			if moreFill == "V" {
				hvDis := svgArray[6]
				yxDis, err := strconv.ParseInt(hvDis, 0, 32)
				HandleError(err)
				yxEnd := yxDis + y
				if yxEnd > int64(canvasSettings.CanvasYMax) {
					return false
				}
			}
		}
	}

	if fill == "V" {
		vDis := svgArray[4]
		yDis, err := strconv.ParseInt(vDis, 0, 32)
		HandleError(err)
		yEnd := yDis + y
		if yEnd > int64(canvasSettings.CanvasYMax) {
			return false
		}
	}

	if fill == "L" {
		xline := svgArray[4]
		yline := svgArray[5]
		xlend, err := strconv.ParseInt(xline, 0, 32)
		HandleError(err)
		ylend, err := strconv.ParseInt(yline, 0, 32)
		HandleError(err)

		if ylend > int64(canvasSettings.CanvasYMax) {
			return false
		}
		if xlend > int64(canvasSettings.CanvasYMax) {
			return false
		}
	}

	return true
}

func HandleError(err error) {
	if err != nil {
		fmt.Println(err)
	}
}
