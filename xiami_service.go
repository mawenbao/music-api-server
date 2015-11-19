package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

const (
	gXiamiSongSplitter = ","
	gXiamiRetOK        = 0
	gXiamiRetFail      = "failed"
	gXiamiTokenName    = "_xiamitoken"
	gXiamiProvider     = "http://www.xiami.com/"
	gXiamiAPIUrlBase   = "http://api.xiami.com/web?v=2.0&app_key=1&r="
	gXiamiSongUrl      = "song/detail&id="
	gXiamiAlbumUrl     = "album/detail&id="
	gXiamiCollectUrl   = "collect/detail&type=collectId&id="
)

var (
	gXiamiTokenVal    = "_xiamitoken="
	gXiamiClient      = &http.Client{}
	gXiamiHttpHeaders = map[string]string{
		"User-Agent":       "Mozilla/5.0 (iPhone; CPU iPhone OS 7_1_2 like Mac OS X) AppleWebKit/537.51.2 (KHTML, like Gecko) Version/7.0 Mobile/11D257 Safari/9537.53",
		"Referer":          "http://m.xiami.com/",
		"Host":             "m.xiami.com",
		"Proxy-Connection": "keep-alive",
		"X-Requested-With": "XMLHttpRequest",
		"X-FORWARDED-FOR":  "42.156.140.238",
		"CLIENT-IP":        "42.156.140.238",
	}
)

type XiamiRetStatus struct {
	Status  int    `json:"state"`
	Message string `json:"message"`
}

type XiamiSong struct {
	Name   string `json:"song_name"`
	Url    string `json:"listen_file"`
	Artist string `json:"singers"`
}

type XiamiRetData struct {
	Songs []XiamiSong `json:"songs"`
	Song  XiamiSong   `json:"song"`
}

type XiamiRet struct {
	XiamiRetStatus
	XiamiRetData `json:"data"`
}

func init() {
	if !setXiamiToken() {
		log.Fatalln("failed to set xiami token")
	}
}

func isXiamiTokenSet() bool {
	return (gXiamiTokenName + "=") != gXiamiTokenVal
}

func setXiamiToken() bool {
	if isXiamiTokenSet() {
		return true
	}
	tokenUrl := "http://m.xiami.com"
	req, err := http.NewRequest("HEAD", tokenUrl, nil)
	if nil != err {
		log.Printf("failed to create http request for url %s: %s", tokenUrl, err)
		return false
	}
	for k, v := range gXiamiHttpHeaders {
		req.Header.Add(k, v)
	}
	resp, err := gXiamiClient.Do(req)
	defer resp.Body.Close()
	if nil != err {
		log.Printf("error get url %s: %s", tokenUrl, err)
		return false
	}

	// parse xiami token from cookies
	for _, c := range resp.Cookies() {
		if gXiamiTokenName == c.Name {
			gXiamiTokenVal += c.Value
		}
	}
	if !isXiamiTokenSet() {
		log.Printf("error get xiami token from response cookies")
		return false
	}
	return true
}

func getXiamiUrl(client *http.Client, url string) []byte {
	cacheKey := GenUrlCacheKey(url)
	if "" == cacheKey {
		return nil
	}
	// try to load from cache first
	body := GetCache(cacheKey, true)
	if nil != body {
		return body
	}

	// do the real request
	urlWithToken := url + "&" + gXiamiTokenVal
	req, err := http.NewRequest("GET", urlWithToken, nil)
	if nil != err {
		log.Printf("failed to create http request for url %s: %s", urlWithToken, err)
		return nil
	}
	for k, v := range gXiamiHttpHeaders {
		req.Header.Add(k, v)
	}
	req.Header.Add("Cookie", gXiamiTokenVal)
	resp, err := gXiamiClient.Do(req)
	if nil != err {
		log.Printf("error get url %s: %s")
		return nil
	}
	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("error getting response body from url %s: %s", url, err)
		return nil
	}

	// update cache, with compression
	SetCache(cacheKey, body, gCacheExpiryTime, true)
	return body
}

func parseXiamiRetData(url string, data []byte) *SongList {
	sl := NewSongList()
	if nil == data {
		return sl.SetAndLogErrorf("error accessing url %s", url)
	}

	var songret XiamiRet
	err := json.Unmarshal(data, &songret)
	if nil != err {
		return sl.SetAndLogErrorf("error parsing xiami return data from url %s: %s", url, err)
	}

	if gXiamiRetOK != songret.Status {
		return sl.SetAndLogErrorf("error getting url %s: %s", url, songret.Message)
	}

	if 0 != len(songret.Songs) {
		// for album and collect
		for i, _ := range songret.Songs {
			song := &songret.Songs[i]
			sl.AddSong(&Song{
				Name:     song.Name,
				Url:      song.Url,
				Artists:  song.Artist,
				Provider: gXiamiProvider,
			})
		}
		return sl
	} else if "" != songret.Song.Url {
		sl.AddSong(&Song{
			Name:     songret.Song.Name,
			Url:      songret.Song.Url,
			Artists:  songret.Song.Artist,
			Provider: gXiamiProvider,
		})
		return sl
	} else {
		return sl.SetAndLogErrorf("invalid xiami url: %s", url)
	}
}

func getXiamiSong(songID string) *SongList {
	url := gXiamiAPIUrlBase + gXiamiSongUrl + strings.TrimSpace(songID)
	ret := getXiamiUrl(gXiamiClient, url)
	return parseXiamiRetData(url, ret)
}

func GetXiamiSongList(params *ReqParams) *SongList {
	sl := NewSongList()
	for _, sid := range strings.Split(params.ID, gXiamiSongSplitter) {
		singleSL := getXiamiSong(strings.TrimSpace(sid))
		if singleSL.IsFailed() {
			return singleSL
		}
		sl.Concat(singleSL)
	}
	return sl
}

func GetXiamiCollect(params *ReqParams) *SongList {
	url := gXiamiAPIUrlBase + gXiamiCollectUrl + strings.TrimSpace(params.ID)
	ret := getXiamiUrl(gXiamiClient, url)
	return parseXiamiRetData(url, ret)
}

func GetXiamiAlbum(params *ReqParams) *SongList {
	url := gXiamiAPIUrlBase + gXiamiAlbumUrl + strings.TrimSpace(params.ID)
	ret := getXiamiUrl(gXiamiClient, url)
	return parseXiamiRetData(url, ret)
}
