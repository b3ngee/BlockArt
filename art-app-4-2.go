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
	validateNum := uint8(1)

	privateKeyBytesRestored, _ := hex.DecodeString(os.Args[1])
	privKey, _ := x509.ParseECPrivateKey(privateKeyBytesRestored)

	// Open a canvas.
	canvas, settings, err := blockartlib.OpenCanvas(minerAddr, *privKey)
	if checkError(err) != nil {
		return
	}

	fmt.Println("ValidateNum for this app: ", validateNum)

	// Overlaps a black square with orange square from art-app-4-1
	_, _, _, err1 := canvas.AddShape(validateNum, blockartlib.PATH, "M 0 90 L 20 90 L 20 110 L 0 110 Z", "transparent", "black")
	checkError(err1)

	// Adds black square with yellow fill
	sh, _, _, err2 := canvas.AddShape(validateNum, blockartlib.PATH, "M 0 120 L 20 120 L 20 140 L 0 140 Z", "yellow", "black")
	checkError(err2)

	time.Sleep(5 * time.Second)

	// Delete previous shape
	_, derr := canvas.DeleteShape(validateNum, sh)
	checkError(derr)

	// Delete previous shape
	_, derr2 := canvas.DeleteShape(validateNum, sh)
	checkError(derr2)

	time.Sleep(90 * time.Second)

	svgs, _ := blockartlib.GetAllSVGs(canvas)
	blockartlib.CreateCanvasHTML(svgs, "4-2", settings)
	fmt.Println("Svg strings: ", svgs)

	// Close the canvas.
	_, err3 := canvas.CloseCanvas()
	checkError(err3)

	fmt.Println("Successful art-app-4-2")
}

// If error is non-nil, print it out and return it.
func checkError(err error) error {
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return err
	}
	return nil
}
