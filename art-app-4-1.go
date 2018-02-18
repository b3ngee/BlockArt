package main

import (
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"./blockartlib"
)

func main() {
	minerAddr := os.Args[2]
	validateNum := uint8(3)

	privateKeyBytesRestored, _ := hex.DecodeString(os.Args[1])
	privKey, _ := x509.ParseECPrivateKey(privateKeyBytesRestored)

	// Open a canvas.
	canvas, settings, err := blockartlib.OpenCanvas(minerAddr, *privKey)
	if checkError(err) != nil {
		return
	}

	// Add a orange square.
	_, _, _, err1 := canvas.AddShape(validateNum, blockartlib.PATH, "M 0 90 L 20 90 L 20 110 L 0 110 Z", "transparent", "orange")
	checkError(err1)

	// Overlaps with triangle in art-app-3
	_, _, _, derr := canvas.AddShape(validateNum, blockartlib.PATH, "M 30 60 L 50 60 L 50 80 Z", "transparent", "orange")
	checkError(derr)

	// Add a orange square inside the first one
	_, _, _, err2 := canvas.AddShape(validateNum, blockartlib.PATH, "M 15 95 L 25 95 L 25 105 L 15 105 Z", "transparent", "orange")
	checkError(err2)

	time.Sleep(90 * time.Second)

	svgs, _ := blockartlib.GetAllSVGs(canvas)
	blockartlib.CreateCanvasHTML(svgs, "4-1", settings)
	fmt.Println("Svg strings: ", svgs)

	// Close the canvas.
	_, err3 := canvas.CloseCanvas()
	checkError(err3)

	fmt.Println("Successful art-app-4-1")
}

// If error is non-nil, print it out and return it.
func checkError(err error) error {
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return err
	}
	return nil
}
