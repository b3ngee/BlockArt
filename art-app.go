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
	minerAddr := "[::]:57174"
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
	shapeHash1, blockHash1, ink1, err1 := canvas.AddShape(validateNum, blockartlib.PATH, "M 0 0 L 10 0 L 10 10 L 0 10 Z", "transparent", "red")
	if checkError(err1) != nil {
		fmt.Print("First add error")
	}

	fmt.Println("ShapeHash: ", shapeHash1)
	fmt.Println("BlockHash: ", blockHash1)
	fmt.Println("Ink: ", ink1)

	time.Sleep(5 * time.Second)

	shapeHash2, _, _, err2 := canvas.AddShape(validateNum, blockartlib.PATH, "M 300 300 L 900 300 L 500 500 z", "blue", "red")
	if checkError(err2) != nil {
		fmt.Println("Should not have enough ink")
	}

	time.Sleep(5 * time.Second)

	_, _, _, err3 := canvas.AddShape(validateNum, blockartlib.PATH, "M 5 0 L 0 50", "transparent", "green")
	if checkError(err3) != nil {
		fmt.Println("Should intersect first shape")
	}

	_, err4 := canvas.DeleteShape(validateNum, shapeHash2)
	if checkError(err4) != nil {
		fmt.Println("Cannot delete second shape because it was never added")
	}

	ink5, err5 := canvas.DeleteShape(validateNum, shapeHash1)
	if checkError(err5) != nil {
		fmt.Println("Should have deleted first shape, something went wrong")
	}

	fmt.Println("Ink after delete: ", ink5)

	// // Close the canvas.
	// ink4, err := canvas.CloseCanvas()
	// if checkError(err) != nil {
	// 	return
	// }

	time.Sleep(2 * time.Minute)

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
