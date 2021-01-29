package server

import (
	"fmt"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
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
	Status   messageStatus `json:"status"`
	Endpoint string        `json:"endpoint"`
	Data     string        `json:"data"`
	Err      string        `json:"err"`
}

type requestMessage struct {
	Endpoint string `json:"endpoint"`
	Id       string `json:"id"`
}

func New(port string) *Server {
	apiKey := os.Getenv("STEAM_KEY")
	return &Server{
		port:     port,
		steamApi: SteamFriendData.New(apiKey, 3*time.Hour),
	}
}

var upgrader = websocket.Upgrader{}

var notFoundResponse = &responseMessage{
	Status: statusError,
	Err:    "endpoint not found",
}

func (s *Server) handshake(w http.ResponseWriter, r *http.Request) {
	requestLogger := log.WithFields(log.Fields{"user_ip": r.RemoteAddr})
	log.WithField("client ip", r.RemoteAddr).Info("Handshake request")
	socket, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		requestLogger.Error(err)
		return
	}
	defer socket.Close()

	session := socketSession{
		socket:          socket,
		steamApiSession: s.steamApi.NewSession(),
		requestLogger:   requestLogger,
	}
	requestLogger.WithField("client_ip", r.RemoteAddr).Info("Socket session created")
	for {
		if s.steamApi.CallCounter() > 90000 {
			errMsg := responseMessage{
				Status: statusError,
				Err:    "calls to high",
			}
			if err = socket.WriteJSON(errMsg); err != nil {
				requestLogger.Error(err)
				break
			}
		}

		incomingMessage := &requestMessage{}
		if err := socket.ReadJSON(incomingMessage); err != nil {
			errMsg := responseMessage{
				Status: statusError,
				Err:    err.Error(),
			}
			if err = socket.WriteJSON(errMsg); err != nil {
				requestLogger.Error(err)
				break
			}
		}
		response := session.handleMessage(incomingMessage)
		if response.Status == statusError {
			requestLogger.WithField("endpoint", incomingMessage.Endpoint).Error(response.Err)
		}
		if response != nil {
			if err = socket.WriteJSON(response); err != nil {
				requestLogger.Error(err)
				break
			}
		}
		requestLogger.WithField("callCounter", s.steamApi.CallCounter()).Info("Current calls")
	}
}

func (s *Server) Serve() {
	log.SetReportCaller(true)
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	http.HandleFunc("/", s.handshake)
	log.Info("Running on ", s.port)
	fmt.Println(http.ListenAndServe(":"+s.port, nil))
}
