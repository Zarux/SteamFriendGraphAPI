package SteamFriendGraph

import (
	"fmt"
	"sfgapi/internal/pkg/SteamFriendData"
)

type Graph struct {
	Nodes  []*Node `json:"nodes"`
	Edges  []*Edge `json:"edges"`
	RootId string  `json:"rootId"`
}

type Node struct {
	Id    string `json:"id"`
	Label string `json:"label"`
}

type Edge struct {
	Id     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
}

func New(friendData map[string]*SteamFriendData.SteamUser, rootId string) *Graph {
	g := &Graph{}
	g.generate(friendData)
	g.RootId = rootId
	return g
}

func (g *Graph) generate(friendData map[string]*SteamFriendData.SteamUser) {
	for id, user := range friendData {
		node := Node{
			Id:    id,
			Label: id,
		}
		g.Nodes = append(g.Nodes, &node)

		if user.Friends == nil {
			continue
		}
		for _, friend := range user.Friends.Friends {
			edge := Edge{
				Id:     fmt.Sprintf("%s-%s", id, friend.SteamId),
				Source: id,
				Target: friend.SteamId,
			}
			g.Edges = append(g.Edges, &edge)
		}
	}
}

func GenerateLabels(profiles []*SteamFriendData.SteamProfile) map[string]string {
	labels := make(map[string]string)
	for _, profile := range profiles {
		label := profile.PersonaName
		if profile.RealName != "" {
			label += fmt.Sprintf(" (%s)", profile.RealName)
		}
		if profile.LocCountryCode != "" {
			label += fmt.Sprintf(" (%s)", profile.LocCountryCode)
		}
		labels[profile.SteamId] = label
	}
	return labels
}
