/*

A trivial application to illustrate how the blockartlib library can be
used from an application in project 1 for UBC CS 416 2017W2.

Usage:
go run art-app.go
*/

package main

// Expects blockartlib.go to be in the ./blockartlib/ dir, relative to
// this art-app.go file
import (
		"./blockartlib"
		"net"
		"time"
		"bufio"
		"fmt"
		"os"
 		"crypto/ecdsa")

func main() {
	minerAddr := "127.0.0.1:8080"
	privKey := // TODO: use crypto/ecdsa to read pub/priv keys from a file argument.

	// function listens in to ink miners that may want to connect to art node
	listenerForMiner(":8080")

	// Open a canvas.
	canvas, settings, err := blockartlib.OpenCanvas(minerAddr, privKey)
	if checkError(err) != nil {
		return
	}

    validateNum := 2

	// Add a line.
	shapeHash, blockHash, ink, err := canvas.AddShape(validateNum, blockartlib.PATH, "M 0 0 L 0 5", "transparent", "red")
	if checkError(err) != nil {
		return
	}

	// Add another line.
	shapeHash2, blockHash2, ink2, err := canvas.AddShape(validateNum, blockartlib.PATH, "M 0 0 L 5 0", "transparent", "blue")
	if checkError(err) != nil {
		return
	}

	// Delete the first line.
	ink3, err := canvas.DeleteShape(validateNum, shapeHash)
	if checkError(err) != nil {
		return
	}

	// assert ink3 > ink2

	// Close the canvas.
	ink4, err := canvas.CloseCanvas()
	if checkError(err) != nil {
		return
	}
}

// If error is non-nil, print it out and return it.
func checkError(err error) error {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error ", err.Error())
		return err
	}
	return nil
}



// handles the tcp connection to listen into oncoming ink miners trying to connect to this art node
// addr is the port used
func listenerForMiner(addr string){

	listener, err:= net.Listen("tcp", addr)
	if err != nil {
		fmt.Println(err)
		return
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println(err)
			break
		}
		go registerInkMiner(conn)
	}

}


// function for the art node to take in information about the ink miner
func registerInkMiner(conn net.Conn) {

	// timeout the register of the ink miner if it fails to connect within some amount of seconds
	timeoutDuration := 2 * time.Second
	bufReader := bufio.NewReader(conn)

	for {
		conn.SetReadDeadline(time.Now().Add(timeoutDuration))

		// TODO:
		// INCOMPLETE
		// depending the the structure of the ink miner add on to this to get the necessary information regarding the ink miner

		bytes, err := bufReader.ReadBytes('\n')
		if err != nil {
			fmt.Println(err)
			return
		}
	}

}