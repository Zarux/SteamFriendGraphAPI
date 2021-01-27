package server

import (
	"encoding/json"
	"fmt"
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
	profiles, rootId, err := ss.steamApiSession.GenerateFriendData(id, 1)
	fmt.Println("Got profiles")
	if err != nil {
		return nil, err
	}
	graph := SteamFriendGraph.New(profiles, rootId)
	fmt.Println("Got graph")
	data, err := json.Marshal(graph)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (ss *socketSession) generateFriendProfilesJson(id string) ([]byte, error) {
	profiles, root, err := ss.steamApiSession.GetFriendProfiles(id)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(struct {
		Friends []*SteamFriendData.SteamProfile `json:"friends"`
		Profile *SteamFriendData.SteamProfile   `json:"profile"`
	}{
		Friends: profiles,
		Profile: root,
	})
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (ss *socketSession) generateLabelsJson() ([]byte, error) {
	profileData, err := ss.steamApiSession.GetProfileData()
	if err != nil {
		return nil, err
	}
	labels := SteamFriendGraph.GenerateLabels(profileData)
	data, err := json.Marshal(labels)
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
	fmt.Println("Handling", msg.Endpoint)
	switch msg.Endpoint {
	case "ping":
		data = []byte("pong")
	case "generateGraphData":
		data, err = ss.generateGraphDataJson(msg.Id)
		if err != nil {
			return &responseMessage{
				Endpoint: msg.Endpoint,
				Status:   statusError,
				Err:      err.Error(),
			}
		}
	case "generateLabels":
		data, err = ss.generateLabelsJson()
		if err != nil {
			return &responseMessage{
				Endpoint: msg.Endpoint,
				Status:   statusError,
				Err:      err.Error(),
			}
		}
	case "getFriendProfiles":
		data, err = ss.generateFriendProfilesJson(msg.Id)
		if err != nil {
			return &responseMessage{
				Endpoint: msg.Endpoint,
				Status:   statusError,
				Err:      err.Error(),
			}
		}
	default:
		return notFoundResponse
	}
	fmt.Println("Handling", msg.Endpoint, "DONE")
	return &responseMessage{
		Endpoint: msg.Endpoint,
		Status:   statusSuccess,
		Data:     string(data),
	}
}
