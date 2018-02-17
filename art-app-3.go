/*

Public Key is:
3076301006072a8648ce3d020106052b8104002203620004046e2495dd079f59b4dd7189f2a42872c4dc0d604adf56187d06dffa630299735e57696551c0fb2800a75f2464c67429651e70df8efa6acdb6e033f0166527a0e0fa44fe723563d26538aca2bf12677f8085a3583174e5134add65eb50b31b7e

Private Key is:
3081a4020101043067bd39ef861ce1ba30bc77d34211d91d4d0b26796607cd6f56f39fd4b988b054939b6adec3941f22aaf7d16e660797f4a00706052b81040022a16403620004046e2495dd079f59b4dd7189f2a42872c4dc0d604adf56187d06dffa630299735e57696551c0fb2800a75f2464c67429651e70df8efa6acdb6e033f0166527a0e0fa44fe723563d26538aca2bf12677f8085a3583174e5134add65eb50b31b7e

go run ink-miner.go 127.0.0.1:12345 3076301006072a8648ce3d020106052b8104002203620004046e2495dd079f59b4dd7189f2a42872c4dc0d604adf56187d06dffa630299735e57696551c0fb2800a75f2464c67429651e70df8efa6acdb6e033f0166527a0e0fa44fe723563d26538aca2bf12677f8085a3583174e5134add65eb50b31b7e 3081a4020101043067bd39ef861ce1ba30bc77d34211d91d4d0b26796607cd6f56f39fd4b988b054939b6adec3941f22aaf7d16e660797f4a00706052b81040022a16403620004046e2495dd079f59b4dd7189f2a42872c4dc0d604adf56187d06dffa630299735e57696551c0fb2800a75f2464c67429651e70df8efa6acdb6e033f0166527a0e0fa44fe723563d26538aca2bf12677f8085a3583174e5134add65eb50b31b7e

go run art-app-3.go 3081a4020101043067bd39ef861ce1ba30bc77d34211d91d4d0b26796607cd6f56f39fd4b988b054939b6adec3941f22aaf7d16e660797f4a00706052b81040022a16403620004046e2495dd079f59b4dd7189f2a42872c4dc0d604adf56187d06dffa630299735e57696551c0fb2800a75f2464c67429651e70df8efa6acdb6e033f0166527a0e0fa44fe723563d26538aca2bf12677f8085a3583174e5134add65eb50b31b7e

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
	minerAddr := "[::]:60660"
	validateNum := uint8(2)

	privateKeyBytesRestored, _ := hex.DecodeString(os.Args[1])
	privKey, _ := x509.ParseECPrivateKey(privateKeyBytesRestored)

	time.Sleep(10 * time.Millisecond)

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

	time.Sleep(500 * time.Millisecond)

	// Delete green square
	_, derr2 := canvas.DeleteShape(validateNum, sh)
	checkError(derr2)

	// Out of bound error
	_, _, _, err2 := canvas.AddShape(validateNum, blockartlib.PATH, "M 60 60 L 1025 60", "transparent", "green")
	checkError(err2)

	time.Sleep(3 * time.Minute)

	svgs, _ := blockartlib.GetAllSVGs(canvas)
	blockartlib.CreateCanvasHTML(svgs, "3", settings)
	fmt.Println("HERE ARE THE SVG: ", svgs)

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
