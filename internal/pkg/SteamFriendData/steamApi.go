package SteamFriendData

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	friendApiUrl = "http://api.steampowered.com/ISteamUser/GetFriendList/v0001/"
	userApiUrl   = "http://api.steampowered.com/ISteamUser/GetPlayerSummaries/v0002/"
	vanityApiUrl = "http://api.steampowered.com/ISteamUser/ResolveVanityURL/v0001/"
)

type SAError struct {
	ErrorString string
}

func (sae *SAError) Error() string {
	return sae.ErrorString
}

var ErrProfileNotFound = &SAError{ErrorString: "profile not found"}

type VanityResolution struct {
	Response struct {
		SteamId string `json:"steamid"`
		Success int    `json:"success"`
	} `json:"response"`
}

type FriendData struct {
	FriendsList `json:"friendslist"`
}

type FriendsList struct {
	Friends []struct {
		SteamId      string `json:"steamid"`
		Relationship string `json:"relationship"`
		FriendSince  int    `json:"friend_since"`
	} `json:"friends"`
}

type UserInfo struct {
	Response struct {
		Players []SteamProfile `json:"players"`
	} `json:"response"`
}

type SteamApi struct {
	key          string
	counter      uint64
	cache        map[string][]byte
	cacheInvChan chan string
	mu           sync.Mutex
}

type SteamProfile struct {
	SteamId                  string `json:"steamid"`
	CommunityVisibilityState int    `json:"communityvisibilitystate"`
	ProfileState             int    `json:"profilestate"`
	PersonaName              string `json:"personaname"`
	ProfileUrl               string `json:"profileurl"`
	Avatar                   string `json:"avatar"`
	AvatarMedium             string `json:"avatarmedium"`
	AvatarFull               string `json:"avatarfull"`
	AvatarHash               string `json:"avatarhash"`
	LastLogoff               int    `json:"lastlogoff,omitempty"`
	PersonaState             int    `json:"personastate"`
	PrimaryClanId            string `json:"primaryclanid,omitempty"`
	TimeCreated              int    `json:"timecreated,omitempty"`
	PersonaStateFlags        int    `json:"personastateflags"`
	CommentPermission        int    `json:"commentpermission,omitempty"`
	RealName                 string `json:"realname,omitempty"`
	LocCountryCode           string `json:"loccountrycode,omitempty"`
	GameExtraInfo            string `json:"gameextrainfo,omitempty"`
	GameId                   string `json:"gameid,omitempty"`
	LocStateCode             string `json:"locstatecode,omitempty"`
	LocCityId                int    `json:"loccityid,omitempty"`
	GameServerIp             string `json:"gameserverip,omitempty"`
	GameServerSteamId        string `json:"gameserversteamid,omitempty"`
}

type SteamUser struct {
	Profile *SteamProfile
	Friends *FriendsList
}

func New(key string, cacheDuration time.Duration) *SteamApi {
	api := &SteamApi{
		key:          key,
		cache:        make(map[string][]byte),
		cacheInvChan: make(chan string),
	}

	go func(api *SteamApi) {
		for {
			select {
			case key := <-api.cacheInvChan:
				time.AfterFunc(cacheDuration, func() {
					delete(api.cache, key)
				})
			}
		}
	}(api)

	return api
}

func (sa *SteamApi) CallCounter() int {
	return int(sa.counter)
}

func (sa *SteamApi) cacheResult(key string, data []byte) {
	sa.mu.Lock()
	sa.cache[key] = data
	sa.mu.Unlock()
	sa.cacheInvChan <- key
}

func (sa *SteamApi) getCache(key string) ([]byte, bool) {
	sa.mu.Lock()
	cache, ok := sa.cache[key]
	sa.mu.Unlock()
	return cache, ok
}

func (sa *SteamApi) get(url string) ([]byte, error) {
	if cache, ok := sa.getCache(url); ok {
		return cache, nil
	}
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	atomic.AddUint64(&sa.counter, 1)
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	sa.cacheResult(url, body)
	return body, nil
}

func (sa *SteamApi) buildUserUrl(ids []string) string {
	idString := strings.Join(ids, ",")
	return fmt.Sprintf("%s?key=%s&steamids=%s", userApiUrl, sa.key, idString)
}

func (sa *SteamApi) buildFriendUrl(id string) string {
	return fmt.Sprintf("%s?key=%s&steamid=%s&relationship=friend", friendApiUrl, sa.key, id)
}

func (sa *SteamApi) buildVanityUrl(id string) string {
	return fmt.Sprintf("%s?key=%s&vanityurl=%s", vanityApiUrl, sa.key, id)
}

func (sa *SteamApi) resolveVanityUrl(id string) (string, error) {
	body, err := sa.get(sa.buildVanityUrl(id))
	if err != nil {
		return "", err
	}
	vanityResolution := VanityResolution{}
	err = json.Unmarshal(body, &vanityResolution)
	if err != nil {
		return "", err
	}
	if vanityResolution.Response.Success == 42 {
		return "", errors.New("couldn't find vanity url")
	}
	return vanityResolution.Response.SteamId, nil
}

func (sa *SteamApi) validateId(id string) (string, error) {
	if strings.HasPrefix(id, "765611") && len(id) == 17 {
		return id, nil
	}
	_, err := strconv.Atoi(id)
	if err != nil {
		id, err = sa.resolveVanityUrl(id)
		if err != nil {
			return "", err
		}
		return id, nil
	}
	id, err = sa.resolveVanityUrl(id)
	if err != nil {
		return "", err
	}
	return id, nil
}

func min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}
