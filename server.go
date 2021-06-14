package main

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
)

func homePage(w http.ResponseWriter, r *http.Request) {
	html, err := ioutil.ReadFile("index.html")
	if err != nil {
		log.Fatal(err)
	}

	b := bytes.NewBuffer(html)

	// stream straight to client(browser)
	w.Header().Set("Content-type", "text/html")

	if _, err := b.WriteTo(w); err != nil { // <----- here!
		log.Fatal(err)
	}
}

func tokenGenerator() string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

var LoggedIn = make(map[string]string)
var chatRoom = make(map[string]*websocket.Conn)

func authenticate(w http.ResponseWriter, r *http.Request) {
	var login struct {
		User     string `json:"username"`
		Password string `json:"password"`
		Token    string `json:"token,omitempty"`
	}
	log.Println("Authenticating...")
	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&login); err != nil {
		log.Fatal(err)
		return
	}
	// Needed to get rid of pesky front-end error.
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if login.User == "root" && login.Password == "password" {
		login.Token = tokenGenerator()

		LoggedIn[login.Token] = login.User

		data, err := json.Marshal(login)
		if err != nil {
			log.Fatalf("JSON marshaling failed: %s", err)
			return
		}
		w.Write(data)
	} else {
		w.WriteHeader(http.StatusForbidden)
	}
}

// This upgrader defines the Read and Write buffer size.
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func wsEndpoint(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	if len(r.Form["access-token"]) < 1 {
		log.Println("No access token provided.")
		return
	}
	token := r.Form["access-token"][0]

	user, ok := LoggedIn[token]
	if !ok {
		log.Println("Trying to access with invalid token")
		return
	}

	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	// upgrade this connection to a WebSocket connection
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
	}

	// helpful log statement to show connections
	log.Printf("User: %s logged in.\n", user)
	if err != nil {
		log.Println(err)
	}

	chatRoom[token] = ws

	reader(ws, token)

	delete(chatRoom, token)
}

func reader(conn *websocket.Conn, token string) {
	// The reader will log all of the messages received to a file
	f, err := os.Create("./data/" + token)
	if err != nil {
		log.Println(err)
		return
	}

	msgLogger := log.New(f, "", log.LstdFlags)

	for {
		// read in a message
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}
		// Print out that message for clarity
		fmt.Println(messageType)
		fmt.Println(string(p))

		msgLogger.Println(string(p))

		for _, ws := range chatRoom {
			if err := ws.WriteMessage(messageType, p); err != nil {
				log.Println(err)
				return
			}
		}
	}
}

func setupRoutes() {
	http.HandleFunc("/", homePage)
	http.HandleFunc("/chat", wsEndpoint)
	http.HandleFunc("/auth", authenticate)
}

func main() {
	fmt.Println("Running server on port 8080...")
	setupRoutes()
	log.Fatal(http.ListenAndServe(":8080", nil))
}
