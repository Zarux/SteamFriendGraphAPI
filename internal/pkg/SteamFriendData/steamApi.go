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

var (
	ErrProfileNotFound = &SAError{ErrorString: "profile not found"}
)

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
	key              string
	counter          uint64
	urlCache         map[string][]byte
	urlCacheInvChan  chan string
	urlCacheMu       sync.Mutex
	profileCache     map[string]SteamProfile
	profileCacheChan chan SteamProfile
	profileCacheMu   sync.Mutex
}

type SteamProfile struct {
	SteamId        string `json:"steamid"`
	PersonaName    string `json:"personaname"`
	ProfileUrl     string `json:"profileurl"`
	AvatarMedium   string `json:"avatarmedium"`
	PersonaState   int    `json:"personastate"`
	TimeCreated    int    `json:"timecreated,omitempty"`
	RealName       string `json:"realname,omitempty"`
	LocCountryCode string `json:"loccountrycode,omitempty"`
}

type SteamUser struct {
	Profile *SteamProfile
	Friends *FriendsList
}

func New(key string, cacheDuration time.Duration) *SteamApi {
	api := &SteamApi{
		key:              key,
		urlCache:         make(map[string][]byte),
		urlCacheInvChan:  make(chan string),
		profileCache:     make(map[string]SteamProfile),
		profileCacheChan: make(chan SteamProfile),
	}

	go func(api *SteamApi) {
		for {
			select {
			case profile := <-api.profileCacheChan:
				api.profileCacheMu.Lock()
				api.profileCache[profile.SteamId] = profile
				api.profileCacheMu.Unlock()
				time.AfterFunc(cacheDuration, func() {
					api.profileCacheMu.Lock()
					delete(api.profileCache, profile.SteamId)
					api.profileCacheMu.Unlock()
				})
			case key := <-api.urlCacheInvChan:
				time.AfterFunc(cacheDuration, func() {
					api.urlCacheMu.Lock()
					delete(api.urlCache, key)
					api.urlCacheMu.Unlock()
				})
			}
		}
	}(api)

	return api
}

func (sa *SteamApi) CallCounter() int {
	return int(sa.counter)
}

func (sa *SteamApi) cacheUrlResult(key string, data []byte) {
	sa.urlCacheMu.Lock()
	sa.urlCache[key] = data
	sa.urlCacheMu.Unlock()
	sa.urlCacheInvChan <- key
}

func (sa *SteamApi) getUrlCache(key string) ([]byte, bool) {
	sa.urlCacheMu.Lock()
	cache, ok := sa.urlCache[key]
	sa.urlCacheMu.Unlock()
	return cache, ok
}

func (sa *SteamApi) getProfileCache(key string) (SteamProfile, bool) {
	sa.profileCacheMu.Lock()
	cache, ok := sa.profileCache[key]
	sa.profileCacheMu.Unlock()
	return cache, ok
}

func (sa *SteamApi) get(url string) ([]byte, error) {
	if cache, ok := sa.getUrlCache(url); ok {
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
	sa.cacheUrlResult(url, body)
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
