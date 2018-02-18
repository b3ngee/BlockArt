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

	// Add a green square.
	sh, _, _, err1 := canvas.AddShape(validateNum, blockartlib.PATH, "M 0 60 L 20 60 L 20 80 L 0 80 Z", "transparent", "green")
	checkError(err1)

	// Add a green triangle
	_, _, _, derr := canvas.AddShape(validateNum, blockartlib.PATH, "M 30 60 L 50 60 L 50 80 Z", "transparent", "green")
	checkError(derr)

	time.Sleep(5 * time.Second)

	// Delete green square
	_, derr2 := canvas.DeleteShape(validateNum, sh)
	checkError(derr2)

	// Out of bound error
	_, _, _, err2 := canvas.AddShape(validateNum, blockartlib.PATH, "M 60 60 L 1025 60", "transparent", "green")
	checkError(err2)

	time.Sleep(90 * time.Second)

	svgs, _ := blockartlib.GetAllSVGs(canvas)
	blockartlib.CreateCanvasHTML(svgs, "3", settings)
	fmt.Println("Svg Strings: ", svgs)

	// Close the canvas.
	_, err3 := canvas.CloseCanvas()
	checkError(err3)

	fmt.Println("Successful art-app-3")
}

// If error is non-nil, print it out and return it.
func checkError(err error) error {
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return err
	}
	return nil
}
