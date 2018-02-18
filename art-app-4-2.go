/*

Public Key is:
3076301006072a8648ce3d020106052b810400220362000494074f7e5aeba2082d33a1023f155f73e704777e1efb08cab4ce237db520f5503a9b21fc761b5dff45e5d7ea26c3ced92e369f5d0e89f61a95400131cb27e8db8ef98ce90c143f1e48e966a18c61eb07af5a171220f71e42f7ac06d008a72290

Private Key is:
3081a4020101043071628d6563274256e60b0cc3be066d4bc617709c3c0299919496e1fbae6a918d8a0784c29700060c31c969de5415830ea00706052b81040022a1640362000494074f7e5aeba2082d33a1023f155f73e704777e1efb08cab4ce237db520f5503a9b21fc761b5dff45e5d7ea26c3ced92e369f5d0e89f61a95400131cb27e8db8ef98ce90c143f1e48e966a18c61eb07af5a171220f71e42f7ac06d008a72290

go run ink-miner.go 127.0.0.1:12345 3076301006072a8648ce3d020106052b810400220362000494074f7e5aeba2082d33a1023f155f73e704777e1efb08cab4ce237db520f5503a9b21fc761b5dff45e5d7ea26c3ced92e369f5d0e89f61a95400131cb27e8db8ef98ce90c143f1e48e966a18c61eb07af5a171220f71e42f7ac06d008a72290 3081a4020101043071628d6563274256e60b0cc3be066d4bc617709c3c0299919496e1fbae6a918d8a0784c29700060c31c969de5415830ea00706052b81040022a1640362000494074f7e5aeba2082d33a1023f155f73e704777e1efb08cab4ce237db520f5503a9b21fc761b5dff45e5d7ea26c3ced92e369f5d0e89f61a95400131cb27e8db8ef98ce90c143f1e48e966a18c61eb07af5a171220f71e42f7ac06d008a72290

go run art-app-4-2.go 3081a4020101043071628d6563274256e60b0cc3be066d4bc617709c3c0299919496e1fbae6a918d8a0784c29700060c31c969de5415830ea00706052b81040022a1640362000494074f7e5aeba2082d33a1023f155f73e704777e1efb08cab4ce237db520f5503a9b21fc761b5dff45e5d7ea26c3ced92e369f5d0e89f61a95400131cb27e8db8ef98ce90c143f1e48e966a18c61eb07af5a171220f71e42f7ac06d008a72290

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
	minerAddr := os.Args[2]
	validateNum := uint8(3)

	privateKeyBytesRestored, _ := hex.DecodeString(os.Args[1])
	privKey, _ := x509.ParseECPrivateKey(privateKeyBytesRestored)

	// Open a canvas.
	canvas, settings, err := blockartlib.OpenCanvas(minerAddr, *privKey)
	if checkError(err) != nil {
		return
	}

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
