/*

A trivial application to illustrate how the blockartlib library can be
used from an application in project 1 for UBC CS 416 2017W2.

Usage:
go run art-app.go [miner priv-key]
*/

package main

// Expects blockartlib.go to be in the ./blockartlib/ dir, relative to
// this art-app.go file
import (
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"./blockartlib"
)

func main() {
	minerAddr := "[::]:63206"
	// privKey := // TODO: use crypto/ecdsa to read pub/priv keys from a file argument.

	privateKeyBytesRestored, _ := hex.DecodeString(os.Args[1])
	privKey, _ := x509.ParseECPrivateKey(privateKeyBytesRestored)

	// Open a canvas.
	canvas, settings, err := blockartlib.OpenCanvas(minerAddr, *privKey)
	if checkError(err) != nil {
		return
	}

	validateNum := uint8(2)

	// Add a line.
	shapeHash, blockHash, ink, err := canvas.AddShape(validateNum, blockartlib.PATH, "M 0 0 L 1000 0 L 0 1000 Z", "blue", "red")
	if checkError(err) != nil {
		return
	}

	fmt.Println("ShapeHash: ", shapeHash)
	fmt.Println("BlockHash: ", blockHash)
	fmt.Println("Ink: ", ink)

	// time.Sleep(30 * time.Second)

	// shapeHash1, blockHash1, ink1, err := canvas.AddShape(validateNum, blockartlib.PATH, "M 300 300 L 500 300 L 500 500 z", "blue", "red")
	// if checkError(err) != nil {
	// 	return
	// }

	// fmt.Println("ShapeHash: ", shapeHash1)
	// fmt.Println("BlockHash: ", blockHash1)
	// fmt.Println("Ink: ", ink1)

	// // Add another line.
	// shapeHash2, blockHash2, ink2, err := canvas.AddShape(validateNum, blockartlib.PATH, "M 0 0 L 5 0", "transparent", "blue")
	// if checkError(err) != nil {
	// 	return
	// }

	// // Delete the first line.
	// ink3, err := canvas.DeleteShape(validateNum, shapeHash)
	// if checkError(err) != nil {
	// 	return
	// }

	// // assert ink3 > ink2

	// // Close the canvas.
	// ink4, err := canvas.CloseCanvas()
	// if checkError(err) != nil {
	// 	return
	// }

	time.Sleep(1 * time.Minute)

	svgs, _ := blockartlib.GetAllSVGs(canvas)
	blockartlib.CreateCanvasHTML(svgs, settings)
	fmt.Println("HERE ARE THE SVG: ", svgs)
}

// If error is non-nil, print it out and return it.
func checkError(err error) error {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error ", err.Error())
		return err
	}
	return nil
}
