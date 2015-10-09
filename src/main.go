//server.go

package main

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"time"
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
	Actions  string
	Stale    bool
}

type GameState struct {
	State string
}

// map json from player to logicserver
func mapPlayerToGlobal(db gorm.DB) *gabs.Container {
	var player_actions []PlayerAction

	// fetch all active entries to memory
	db.Where("Stale = ?", false).Find(&player_actions)
	// update all the entries as stale in a batch
	// TODO: this is stupid, I might lost a player action if something happens
	// I really need to update just the ones which I have got back
	db.Table("player_action").Where("Stale = ?", false).Updates(PlayerAction{Stale: true})

	jsonObj := gabs.New()

	// ref: https://github.com/Jeffail/gabs#generating-json
	for _, player_state := range player_actions {
		jsonObj.Set(player_state.Actions, player_state.Username, "Actions")
	}

	return jsonObj
}

func savePlayerAction(jsonObj gabs.Container, db gorm.DB) bool {
	println("saving player action")
	username := jsonObj.Path("Username").Data().(string)
	userID := username
	actions := jsonObj.Path("Actions").Data().(string)

	playerAction := PlayerAction{Username: username, UserID: userID, Actions: actions, Stale: false}

	return db.NewRecord(&playerAction)
}

func main() {
	// set up database
	db, err := gorm.Open("sqlite3", "/tmp/gorm.db")
	db.CreateTable(&PlayerAction{})
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

	println("logic 1")
	// establishing connection to game logic server
	logic_server, err := net.ResolveTCPAddr("tcp", LOGIC_CONN_HOST+":"+LOGIC_CONN_PORT)
	if err != nil {
		println("Dial failed")
		os.Exit(1)
	}
	logicConn, err := net.DialTCP("tcp", nil, logic_server)
	println("logic 2")

	// event loop
	for {
		println("accepting")
		conn, _ := l.Accept()
		// logs an incoming message
		fmt.Printf("Received message %s -> %s \n", conn.RemoteAddr(), conn.LocalAddr())

		// Handle connections in a new goroutine.
		go handleRequest(conn, logicConn, db)
	}
}

// Handles incoming requests.
func handleRequest(conn net.Conn, logicConn net.Conn, db gorm.DB) {
	println("handling")

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

	savePlayerAction(*jsonParsed, db)

	// go sendGameStateToServer(logicConn, db)

	println("game state shit")
	// get last game state

	// TODO: kirt -- dunno
	gameState := GameState{}
	var lastGameState []byte
	db.Model(&GameState{}).Last(&gameState).Pluck("state", &lastGameState)
	// send last game state to client (singular)
	conn.Write([]byte(lastGameState))

	conn.Close()
}

func sendGameStateToServer(logicConn net.Conn, db gorm.DB) {
	time.Sleep(time.Millisecond * 1500)

	data := mapPlayerToGlobal(db)
	println(data)
	// TODO: kirt -- dunno
}
