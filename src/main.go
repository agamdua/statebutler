//server.go

package main

import (
	"bytes"
	"fmt"
	"net"
	"os"
)

import (
	"github.com/jeffail/gabs"
	"github.com/jinzhu/gorm"
	_ "github.com/mattn/go-sqlite3"
)

const (
	CONN_HOST       = ""
	CONN_PORT       = "3333"
	CONN_TYPE       = "tcp"
	LOGIC_CONN_HOST = ""
	LOGIC_CONN_PORT = "5000"
	LOGIC_CONN_Type = CONN_TYPE
)

type GameState struct {
	UserID   int // TODO: generate UUID and send back to user
	GameTick int
	Action   int
	Data     string
}

func main() {
	// set up database
	db, err := gorm.Open("sqlite3", "/tmp/gorm.db")
	db.CreateTable(&GameState{})

	// Listen for incoming connections.
	l, err := net.Listen(CONN_TYPE, ":"+CONN_PORT)
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	// Close the listener when the application closes.
	defer l.Close()
	fmt.Println("Listening on " + CONN_HOST + ":" + CONN_PORT)

	// establishing connection to game logic server
	logic_server, err := net.ResolveTCPAddr("tcp", LOGIC_CONN_HOST+":"+LOGIC_CONN_PORT)
	if err != nil {
		println("Dial failed")
		os.Exit(1)
	}
	logic_conn, err := net.DialTCP("tcp", nil, logic_server)

	// event loop
	for {
		// Listen for an incoming connection.
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}

		//logs an incoming message
		fmt.Printf("Received message %s -> %s \n", conn.RemoteAddr(), conn.LocalAddr())

		// Handle connections in a new goroutine.
		go handleRequest(conn, logic_conn)
	}
}

// Handles incoming requests.
func handleRequest(conn net.Conn, logicConn net.Conn) {

	// Make a buffer to hold incoming data.
	buf := make([]byte, 1024)

	// Read the incoming connection into the buffer.
	reqLen, err := conn.Read(buf)
	println(string(reqLen))
	if err != nil {
		fmt.Println("Error reading:", err.Error())
	}

	n := bytes.Index(buf, []byte{0})

	// getting the json! up yours, rust
	jsonParsed, err := gabs.ParseJSON(buf[:n-1])
	println("This is a message: " + jsonParsed.String())

	conn.Close()
}
