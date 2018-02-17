/*

Public Key is:
3076301006072a8648ce3d020106052b8104002203620004ce35a39a7ea5c245d8464e7e84a161e77607e58b3a9e293b115b1fed706432e564be15b080f207358f8adca676f751be5fb21c1a4014af3a1b173c3357757ccc3d7402e36bd14107d4b33adf9d7277e7fd3cfd2030a001a909de97a7c91433b8

Private Key is:
3081a402010104307f18c67fdbec3bdb91e6ce08a951871a1941af1b82c6258ab8090193e6949aa3b39a730530c88bf2e2fec9c2339ef32ea00706052b81040022a16403620004ce35a39a7ea5c245d8464e7e84a161e77607e58b3a9e293b115b1fed706432e564be15b080f207358f8adca676f751be5fb21c1a4014af3a1b173c3357757ccc3d7402e36bd14107d4b33adf9d7277e7fd3cfd2030a001a909de97a7c91433b8

go run ink-miner.go 127.0.0.1:12345 3076301006072a8648ce3d020106052b8104002203620004ce35a39a7ea5c245d8464e7e84a161e77607e58b3a9e293b115b1fed706432e564be15b080f207358f8adca676f751be5fb21c1a4014af3a1b173c3357757ccc3d7402e36bd14107d4b33adf9d7277e7fd3cfd2030a001a909de97a7c91433b8 3081a402010104307f18c67fdbec3bdb91e6ce08a951871a1941af1b82c6258ab8090193e6949aa3b39a730530c88bf2e2fec9c2339ef32ea00706052b81040022a16403620004ce35a39a7ea5c245d8464e7e84a161e77607e58b3a9e293b115b1fed706432e564be15b080f207358f8adca676f751be5fb21c1a4014af3a1b173c3357757ccc3d7402e36bd14107d4b33adf9d7277e7fd3cfd2030a001a909de97a7c91433b8

go run art-app-2.go 3081a402010104307f18c67fdbec3bdb91e6ce08a951871a1941af1b82c6258ab8090193e6949aa3b39a730530c88bf2e2fec9c2339ef32ea00706052b81040022a16403620004ce35a39a7ea5c245d8464e7e84a161e77607e58b3a9e293b115b1fed706432e564be15b080f207358f8adca676f751be5fb21c1a4014af3a1b173c3357757ccc3d7402e36bd14107d4b33adf9d7277e7fd3cfd2030a001a909de97a7c91433b8

*/

package main

import (
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"os"

	"./blockartlib"
)

func main() {
	minerAddr := "[::]:61843"
	validateNum := uint8(3)

	privateKeyBytesRestored, _ := hex.DecodeString(os.Args[1])
	privKey, _ := x509.ParseECPrivateKey(privateKeyBytesRestored)

	// Open a canvas.
	canvas, settings, err := blockartlib.OpenCanvas(minerAddr, *privKey)
	if checkError(err) != nil {
		return
	}

	// Add a blue square.
	_, _, _, err1 := canvas.AddShape(validateNum, blockartlib.PATH, "M 0 30 L 20 30 L 20 50 L 0 50 Z", "transparent", "blue")
	checkError(err1)

	// Add a blue triangle
	sh2, _, _, derr := canvas.AddShape(validateNum, blockartlib.PATH, "M 30 30 L 50 30 L 50 50 Z", "purple", "blue")
	checkError(derr)

	// Insufficient ink
	_, _, _, err2 := canvas.AddShape(validateNum, blockartlib.PATH, "M 60 30 L 1000 30 L 1000 40 L 60 40 Z", "purple", "blue")
	checkError(err2)

	// Close the canvas.
	_, err3 := canvas.CloseCanvas()
	checkError(err3)
}

// If error is non-nil, print it out and return it.
func checkError(err error) error {
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return err
	}
	return nil
}
