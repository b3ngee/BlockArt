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
	validateNum := uint8(2)

	privateKeyBytesRestored, _ := hex.DecodeString(os.Args[1])
	privKey, _ := x509.ParseECPrivateKey(privateKeyBytesRestored)

	// Open a canvas.
	canvas, settings, err := blockartlib.OpenCanvas(minerAddr, *privKey)
	if checkError(err) != nil {
		return
	}

	fmt.Println("ValidateNum for this app: ", validateNum)

	// Add a blue square.
	_, _, _, err1 := canvas.AddShape(validateNum, blockartlib.PATH, "M 0 30 L 20 30 L 20 50 L 0 50 Z", "transparent", "blue")
	checkError(err1)

	// Add a blue triangle
	_, _, _, derr := canvas.AddShape(validateNum, blockartlib.PATH, "M 30 30 L 50 30 L 50 50 Z", "purple", "blue")
	checkError(derr)

	// Insufficient ink
	_, _, _, err2 := canvas.AddShape(validateNum, blockartlib.PATH, "M 0 150 L 1023 150 L 1023 1023 L 0 1023 Z", "purple", "blue")
	checkError(err2)

	time.Sleep(90 * time.Second)

	svgs, _ := blockartlib.GetAllSVGs(canvas)
	blockartlib.CreateCanvasHTML(svgs, "2", settings)
	fmt.Println("Svg Strings: ", svgs)

	// Close the canvas.
	_, err3 := canvas.CloseCanvas()
	checkError(err3)

	fmt.Println("Successful art-app-2")
}

// If error is non-nil, print it out and return it.
func checkError(err error) error {
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return err
	}
	return nil
}
