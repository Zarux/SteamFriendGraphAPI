package SteamFriendGraph

import (
	"fmt"
	"sfgapi/internal/pkg/SteamFriendData"
)

type Graph struct {
	Nodes []*Node `json:"nodes"`
	Edges []*Edge `json:"edges"`
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

func New(friendData map[string]*SteamFriendData.SteamUser) *Graph {
	g := &Graph{}
	g.generate(friendData)
	return g
}

func (g *Graph) generate(friendData map[string]*SteamFriendData.SteamUser) {
	for id, user := range friendData {
		if user.Profile == nil || user.Profile.PersonaName == "" {
			continue
		}
		label := fmt.Sprintf("%s (%s) (%s)", user.Profile.PersonaName, user.Profile.RealName, user.Profile.LocCountryCode)
		node := Node{
			Id:    id,
			Label: label,
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
