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

	// Adds a purple square
	sh, _, _, err1 := canvas.AddShape(validateNum, blockartlib.PATH, "M 0 150 L 20 150 L 20 170 L 0 170 Z", "transparent", "purple")
	checkError(err1)

	time.Sleep(5 * time.Second)

	// Delete previous shape
	_, derr := canvas.DeleteShape(validateNum, sh)
	checkError(derr)

	time.Sleep(90 * time.Second)

	svgs, _ := blockartlib.GetAllSVGs(canvas)
	blockartlib.CreateCanvasHTML(svgs, "4-3", settings)
	fmt.Println("Svg strings: ", svgs)

	// Close the canvas.
	_, err3 := canvas.CloseCanvas()
	checkError(err3)

	fmt.Println("Successful art-app-4-3")
}

// If error is non-nil, print it out and return it.
func checkError(err error) error {
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return err
	}
	return nil
}
