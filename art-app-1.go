/*

Public Key is:
3076301006072a8648ce3d020106052b8104002203620004dcd436bc7524d3c4b3019b3bca44e74002c2499f02a8a98b50a967354037d69430e198c8722806e9eb3b01bbd73bc5c94b5acbe1110b4575cf0bb0c2220d1b92bc2f541e230f098bca1d0d283b4f3ca0ca3a8f78e4badaea873db4800d6b3174

Private Key is:
3081a402010104308d5427e6d61e48fcf464bd8942ba3432dc9dbd50c0316c92a895838a430657db853676039067e48019684db086821dd3a00706052b81040022a16403620004dcd436bc7524d3c4b3019b3bca44e74002c2499f02a8a98b50a967354037d69430e198c8722806e9eb3b01bbd73bc5c94b5acbe1110b4575cf0bb0c2220d1b92bc2f541e230f098bca1d0d283b4f3ca0ca3a8f78e4badaea873db4800d6b3174

go run ink-miner.go 127.0.0.1:12345 3076301006072a8648ce3d020106052b8104002203620004dcd436bc7524d3c4b3019b3bca44e74002c2499f02a8a98b50a967354037d69430e198c8722806e9eb3b01bbd73bc5c94b5acbe1110b4575cf0bb0c2220d1b92bc2f541e230f098bca1d0d283b4f3ca0ca3a8f78e4badaea873db4800d6b3174 3081a402010104308d5427e6d61e48fcf464bd8942ba3432dc9dbd50c0316c92a895838a430657db853676039067e48019684db086821dd3a00706052b81040022a16403620004dcd436bc7524d3c4b3019b3bca44e74002c2499f02a8a98b50a967354037d69430e198c8722806e9eb3b01bbd73bc5c94b5acbe1110b4575cf0bb0c2220d1b92bc2f541e230f098bca1d0d283b4f3ca0ca3a8f78e4badaea873db4800d6b3174

go run art-app-1.go 3081a402010104308d5427e6d61e48fcf464bd8942ba3432dc9dbd50c0316c92a895838a430657db853676039067e48019684db086821dd3a00706052b81040022a16403620004dcd436bc7524d3c4b3019b3bca44e74002c2499f02a8a98b50a967354037d69430e198c8722806e9eb3b01bbd73bc5c94b5acbe1110b4575cf0bb0c2220d1b92bc2f541e230f098bca1d0d283b4f3ca0ca3a8f78e4badaea873db4800d6b3174

*/

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
	minerAddr := "[::]:61729"
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

	// Add a line that intersects square (allowed)
	sh, _, _, err2 := canvas.AddShape(validateNum, blockartlib.PATH, "M 0 0 L 25 25", "transparent", "red")
	checkError(err2)

	// Add a red triangle, fill purple
	_, _, _, derr := canvas.AddShape(validateNum, blockartlib.PATH, "M 30 0 L 50 0 L 50 20 Z", "purple", "red")
	checkError(derr)

	// Add a red line
	_, _, _, err3 := canvas.AddShape(validateNum, blockartlib.PATH, "M 60 0 L 160 0", "transparent", "red")
	checkError(err3)

	// Delete red triangle
	_, derr2 := canvas.DeleteShape(validateNum, sh)
	checkError(derr2)

	time.Sleep(120 * time.Second) // 2 minute timeout

	svgs, _ := blockartlib.GetAllSVGs(canvas)
	blockartlib.CreateCanvasHTML(svgs, settings)
	fmt.Println("HERE ARE THE SVG: ", svgs)

	// Close the canvas.
	_, err4 := canvas.CloseCanvas()
	checkError(err4)
}

// If error is non-nil, print it out and return it.
func checkError(err error) error {
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return err
	}
	return nil
}
