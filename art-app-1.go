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

	// Add a red square.
	_, _, _, err1 := canvas.AddShape(validateNum, blockartlib.PATH, "M 0 0 L 20 0 L 20 20 L 0 20 Z", "transparent", "red")
	checkError(err1)

	// Add a red triangle, fill purple
	_, _, _, derr := canvas.AddShape(validateNum, blockartlib.PATH, "M 30 0 L 50 0 L 50 20 Z", "purple", "red")
	checkError(derr)

	// Delete non-existent thing
	_, derr2 := canvas.DeleteShape(validateNum, "This doesn't exist")
	checkError(derr2)

	time.Sleep(120 * time.Second)

	svgs, _ := blockartlib.GetAllSVGs(canvas)
	blockartlib.CreateCanvasHTML(svgs, "1", settings)
	fmt.Println("Svg Strings: ", svgs)

	// Close the canvas.
	_, err4 := canvas.CloseCanvas()
	checkError(err4)

	fmt.Println("Successful art-app-1")
}

// If error is non-nil, print it out and return it.
func checkError(err error) error {
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return err
	}
	return nil
}
