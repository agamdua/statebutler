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
	LOGIC_CONN_HOST = "" // TODO: fill in
	LOGIC_CONN_PORT = "5000"
	LOGIC_CONN_Type = CONN_TYPE
)

type PlayerAction struct {
	Username string
	UserID   string // TODO: generate UUID and send back to user
	GameTick int
	Actions  string
	Stale    bool
}

type GameState struct {
	GameTick int // TODO: needs to be a FK
	State    string
}

// map json from player to logicserver
func mapPlayerToGlobal(db gorm.DB) *gabs.Container {
	// TODO: IMPORTANT THIS RELIES ON GAME TICK LOGIC
	var player_actions []PlayerAction

	// fetch all active entries to memory
	// ordering by game tick so only recent ones are taken
	db.Where("Stale = ?", false).Order("GameTick").Find(&player_actions)

	// update all the entries as stale in a batch
	// TODO: ideally the above two should be in a transaction
	db.Table("player_action").Where("Stale = ?", false).Updates(PlayerAction{Stale: true})

	jsonObj := gabs.New()

	// ref: https://github.com/Jeffail/gabs#generating-json
	for _, player_state := range player_actions {
		jsonObj.Set(player_state.Actions, player_state.Username, "Actions")
		jsonObj.Set(player_state.GameTick, player_state.Username, "GameTick")
	}

	return jsonObj
}

func savePlayerAction(jsonObj gabs.Container, db gorm.DB) bool {
	// parse message from player and dump in db
	// TODO: what is th equivalent of `for key in dict.keys()` ??
	// TODO: once I figure that out I need to save this in PlayerAction model
	return true
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
	logicConn, err := net.DialTCP("tcp", nil, logic_server)

	// event loop
	for {
		// TODO: we need to maintain game ticks here
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}

		//logs an incoming message
		fmt.Printf("Received message %s -> %s \n", conn.RemoteAddr(), conn.LocalAddr())

		// Handle connections in a new goroutine.
		go handleRequest(conn, logicConn, db)
	}
}

// Handles incoming requests.
func handleRequest(conn net.Conn, logicConn net.Conn, db gorm.DB) {

	// Make a buffer to hold incoming data.
	buf := make([]byte, 1024)

	// Read the incoming connection into the buffer.
	reqLen, err := conn.Read(buf)
	println(string(reqLen))
	if err != nil {
		fmt.Println("Error reading:", err.Error())
	}

	n := bytes.Index(buf, []byte{0})

	// getting the json! up yours, rust.
	jsonParsed, err := gabs.ParseJSON(buf[:n-1])
	println("This is a message: " + jsonParsed.String())

	// save in db
	savePlayerAction(*jsonParsed, db)

	// write to logic server
	// TODO: check if that will communicate over sockets
	// TODO: can this be its goroutine?
	_, err = logicConn.Write([]byte(jsonParsed.String()))
	if err != nil {
		println("Write to server failed: ", err.Error())
		os.Exit(1)
	}

	conn.Close()
}
