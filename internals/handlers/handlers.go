package handlers

import (
	"fmt"
	"log"
	"net/http"
	"sort"

	"github.com/gorilla/websocket"
)

type WebSocketConnection struct {
	*websocket.Conn
}

type WsJsonResponse struct {
	Action        string
	Message       string
	Move          int
	FirstMove     bool
	ConnctedUsers []string
}

type WsPayload struct {
	Action   string
	Username string
	Message  string
	Move     int
	Conn     WebSocketConnection
}

var wsChan = make(chan WsPayload)
var clients = make(map[WebSocketConnection]string)
var opponentMappings = make(map[WebSocketConnection]WebSocketConnection)

var upgradeConnection = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func Home(w http.ResponseWriter, r *http.Request) {
	_, err := w.Write([]byte(("<h1>Hello World!</h1>")))
	if err != nil {
		log.Println("Had an error at rendering home page")
	}
}

func WsEndPoint(w http.ResponseWriter, r *http.Request) {
	ws, err := upgradeConnection.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
	}
	log.Println("Client connected to endpoint")
	conn := WebSocketConnection{Conn: ws}
	clients[conn] = ""
	if err != nil {
		log.Println(err, "error from hi tic toe message")
	}
	go ListenForWs(&conn)
}

// Listening for payload in connection
func ListenForWs(conn *WebSocketConnection) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Error", fmt.Sprintf("%v", r))
		}
	}()

	var payload WsPayload
	for {
		err := conn.ReadJSON(&payload)
		if err != nil {
			//do nothing
			log.Println(err)
		} else {
			payload.Conn = *conn
			wsChan <- payload
			log.Println(payload, "From paylod listener")
		}
	}
}

// Processing the messages in channel
func ListenToWsChannel() {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Error", fmt.Sprintf("%v", r))
		}
	}()

	var response WsJsonResponse
	for {
		e := <-wsChan
		switch e.Action {
		case "match":
			log.Println("request for match")
			users := getUserList()
			response.ConnctedUsers = users
			response.Move = -1
			foundMatch, firstMove := matchPlayer(e.Conn)
			if foundMatch {
				response.Action = "matchSuccess"
				response.Message = "Player found"
				response.FirstMove = firstMove
				e.Conn.WriteJSON(response)
			}
			if !foundMatch {
				response.Action = "matchFail"
				response.Message = "No available players found"
				e.Conn.WriteJSON(response)
			}
			log.Println(clients)
		case "over":
			delete(opponentMappings, e.Conn)
			delete(opponentMappings, opponentMappings[e.Conn])
		case "move":
			response.Action = "move"
			response.Move = e.Move
			response.Message = e.Message
			communicateWithPlayer(e.Conn, response)
		}
	}
}

func getUserList() []string {
	var userList []string
	for _, username := range clients {
		if username != "" {
			userList = append(userList, username)
		}
	}
	sort.Strings(userList)
	return userList
}

func communicateWithPlayer(conn WebSocketConnection, response WsJsonResponse) {
	opponentConn, ok := opponentMappings[conn]
	if !ok {
		log.Println("No opponent Conn found for conn")
	}
	err := opponentConn.WriteJSON(response)
	if err != nil {
		log.Println("Encountered an error communicating move")
		conn.Close()
		opponentConn.Close()
	}
}

func matchPlayer(conn WebSocketConnection) (bool, bool) {
	log.Println("Entered match player !")
	for client := range clients {
		_, ok := opponentMappings[client]
		if ok && opponentMappings[client] == conn {
			opponentMappings[conn] = client
			opponentMappings[client] = conn
			return true, false
		}
		if client == conn || ok {
			log.Println(clients, client)
			continue
		}
		opponentMappings[conn] = client
		opponentMappings[client] = conn
		return true, true
	}
	return false, false
}
