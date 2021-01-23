package SteamFriendData

import (
	"encoding/json"
	"sync"
)

type Session struct {
	steamApi  *SteamApi
	profiles  map[string]*SteamUser
	profileMu sync.Mutex
}

func (sa *SteamApi) NewSession() *Session {
	s := &Session{
		steamApi: sa,
		profiles: make(map[string]*SteamUser),
	}
	return s
}

func (s *Session) addToProfile(user *SteamUser) {
	s.profileMu.Lock()
	if _, ok := s.profiles[user.Profile.SteamId]; !ok {
		s.profiles[user.Profile.SteamId] = user
	}
	s.profileMu.Unlock()
}

func (s *Session) updateProfile(info SteamProfile) {
	s.profileMu.Lock()
	s.profiles[info.SteamId].Profile = &info
	s.profileMu.Unlock()
}

func (s *Session) getUserSlice() []*SteamUser {
	var userSlice []*SteamUser
	for _, user := range s.profiles {
		userSlice = append(userSlice, user)
	}
	return userSlice
}

func (s *Session) fillUserInfo(users []*SteamUser) error {
	var (
		chunks [][]*SteamUser
		wg     sync.WaitGroup
	)
	errChan := make(chan error)

	n := 100
	for i := 0; i < len(users); i += n {
		chunk := users[i:min(i+n, len(users))]
		chunks = append(chunks, chunk)
	}
	for _, chunk := range chunks {
		wg.Add(1)
		go func(chunk []*SteamUser) {
			defer wg.Done()
			var ids []string
			for _, user := range chunk {
				ids = append(ids, user.Profile.SteamId)
			}
			body, err := s.steamApi.get(s.steamApi.buildUserUrl(ids))
			if err != nil {
				errChan <- err
				return
			}
			userInfo := UserInfo{}
			err = json.Unmarshal(body, &userInfo)
			if err != nil {
				errChan <- err
				return
			}
			for _, info := range userInfo.Response.Players {
				s.updateProfile(info)
			}
		}(chunk)
	}
	wg.Wait()
	return nil
}

func (s *Session) getFriends(user *SteamUser) error {
	body, err := s.steamApi.get(s.steamApi.buildFriendUrl(user.Profile.SteamId))
	if err != nil {
		return err
	}
	friendData := FriendData{}
	err = json.Unmarshal(body, &friendData)
	if err != nil {
		return err
	}

	user.Friends = &friendData.FriendsList

	for _, friend := range friendData.FriendsList.Friends {
		friendUser := &SteamUser{
			Profile: &SteamProfile{
				SteamId: friend.SteamId,
			},
		}
		s.addToProfile(friendUser)
	}
	return nil
}

func (s *Session) multiGetFriends() {
	var wg sync.WaitGroup
	errChan := make(chan error)
	userChan := make(chan *SteamUser, len(s.profiles))

	for _, user := range s.profiles {
		userChan <- user
	}
	close(userChan)

	for user := range userChan {
		if user.Friends != nil {
			continue
		}
		wg.Add(1)
		go func(user *SteamUser) {
			defer wg.Done()
			err := s.getFriends(user)
			if err != nil {
				errChan <- err
				return
			}
		}(user)
	}
	wg.Wait()
}

func (s *Session) GetFriendProfiles(id string) ([]*SteamProfile, error) {
	id, err := s.steamApi.validateId(id)
	if err != nil {
		return nil, err
	}
	if _, ok := s.profiles[id]; !ok {
		return nil, ErrProfileNotFound
	}
	var friends []*SteamProfile
	for _, friend := range s.profiles[id].Friends.Friends {
		if f, ok := s.profiles[friend.SteamId]; ok {
			friends = append(friends, f.Profile)
		}
	}
	return friends, nil
}

func (s *Session) GenerateFriendData(rootId string, depth int) (map[string]*SteamUser, error) {
	rootId, err := s.steamApi.validateId(rootId)
	if err != nil {
		return nil, err
	}
	rootUser := SteamUser{
		Profile: &SteamProfile{
			SteamId: rootId,
		},
	}
	s.addToProfile(&rootUser)
	err = s.getFriends(&rootUser)
	if err != nil {
		return nil, err
	}
	s.multiGetFriends()

	err = s.fillUserInfo(s.getUserSlice())
	if err != nil {
		return nil, err
	}
	return s.profiles, nil
}

func (s *Session) Clear() {
	s.profiles = make(map[string]*SteamUser)
}
