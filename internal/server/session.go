package server

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"sfgapi/internal/pkg/SteamFriendData"
	"sfgapi/internal/pkg/SteamFriendGraph"
)

type socketSession struct {
	socket          *websocket.Conn
	steamApiSession *SteamFriendData.Session
}

func (ss *socketSession) generateGraphDataJson(id string) ([]byte, error) {
	ss.steamApiSession.Clear()
	profiles, err := ss.steamApiSession.GenerateFriendData(id, 1)
	if err != nil {
		return nil, err
	}
	graph := SteamFriendGraph.New(profiles)
	data, err := json.Marshal(graph)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (ss *socketSession) generateFriendProfilesJson(id string) ([]byte, error) {
	profiles, err := ss.steamApiSession.GetFriendProfiles(id)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(profiles)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (ss *socketSession) handleMessage(msg *requestMessage) *responseMessage {
	var (
		data []byte
		err  error
	)
	switch msg.Endpoint {
	case "generateGraphData":
		data, err = ss.generateGraphDataJson(msg.Id)
		if err != nil {
			return &responseMessage{
				Status: statusError,
				Err:    err.Error(),
			}
		}
	case "getFriendProfiles":
		data, err = ss.generateFriendProfilesJson(msg.Id)
		if err != nil {
			return &responseMessage{
				Status: statusError,
				Err:    err.Error(),
			}
		}
	default:
		return notFoundResponse
	}

	return &responseMessage{
		Status: statusSuccess,
		Data:   string(data),
	}
}
