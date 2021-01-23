package server

import (
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"os"
	"sfgapi/internal/pkg/SteamFriendData"
	"time"
)

type Server struct {
	port     string
	steamApi *SteamFriendData.SteamApi
}

type messageStatus int

const (
	statusSuccess messageStatus = iota
	statusError
)

type responseMessage struct {
	Status messageStatus `json:"status"`
	Data   string        `json:"data"`
	Err    string        `json:"err"`
}

type requestMessage struct {
	Endpoint string `json:"endpoint"`
	Id       string `json:"id"`
}

func New(port string) *Server {
	apiKey := os.Getenv("STEAM_KEY")
	return &Server{
		port:     port,
		steamApi: SteamFriendData.New(apiKey, 15*time.Minute),
	}
}

var upgrader = websocket.Upgrader{}

var notFoundResponse = &responseMessage{
	Status: statusError,
	Err:    "endpoint not found",
}

func (s *Server) handshake(w http.ResponseWriter, r *http.Request) {
	socket, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Println(err)
		return
	}
	defer socket.Close()

	session := socketSession{
		socket:          socket,
		steamApiSession: s.steamApi.NewSession(),
	}

	for {
		incomingMessage := &requestMessage{}
		if err := socket.ReadJSON(incomingMessage); err != nil {
			errMsg := responseMessage{
				Status: statusError,
				Err:    err.Error(),
			}
			if err = socket.WriteJSON(errMsg); err != nil {
				fmt.Println(err)
				break
			}
		}
		response := session.handleMessage(incomingMessage)
		if response != nil {
			if err = socket.WriteJSON(response); err != nil {
				fmt.Println(err)
				break
			}
		}
	}
}

func (s *Server) Serve() {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	http.HandleFunc("/", s.handshake)
	fmt.Println("Running on", s.port)
	fmt.Println(http.ListenAndServe("localhost:"+s.port, nil))
}
