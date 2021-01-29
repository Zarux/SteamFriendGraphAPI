package server

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"sfgapi/internal/pkg/SteamFriendData"
	"sfgapi/internal/pkg/SteamFriendGraph"
)

type socketSession struct {
	socket          *websocket.Conn
	steamApiSession *SteamFriendData.Session
	requestLogger   *log.Entry
}

func (ss *socketSession) generateGraphDataJson(id string) ([]byte, error) {
	ss.steamApiSession.Clear()
	profiles, rootId, err := ss.steamApiSession.GenerateFriendData(id, 1)
	ss.requestLogger.Info("Friends generated")
	if err != nil {
		return nil, err
	}
	graph := SteamFriendGraph.New(profiles, rootId)
	ss.requestLogger.Info("Graph generated")
	data, err := json.Marshal(graph)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (ss *socketSession) generateFriendProfilesJson(id string) ([]byte, error) {
	profiles, root, err := ss.steamApiSession.GetFriendProfiles(id)
	ss.requestLogger.Info("Friend profiles got")
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
	ss.requestLogger.Info("Profile data got")
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
	ss.requestLogger.WithField("endpoint", msg.Endpoint).Info("Handling BEGIN")
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
	ss.requestLogger.WithField("endpoint", msg.Endpoint).Info("Handling COMPLETE")
	return &responseMessage{
		Endpoint: msg.Endpoint,
		Status:   statusSuccess,
		Data:     string(data),
	}
}
