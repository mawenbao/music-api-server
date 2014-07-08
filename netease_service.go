package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
)

const (
	gNeteaseRetOk       = 200
	gNeteaseProvider    = "netease"
	gNeteaseAPIUrlBase  = "http://music.163.com/api"
	gNeteaseAlbumUrl    = "/album/"
	gNeteaseSongListUrl = "/song/detail?ids=[%s]"
	gNeteasePlayListUrl = "/playlist/detail?id="
)

var (
	gNeteaseClient = &http.Client{}
)

func init() {
	// init netease http client
	cookies, err := cookiejar.New(nil)
	if nil != err {
		log.Fatal("failed to init netease httpclient cookiejar: %s", err)
	}
	apiUrl, err := url.Parse(gNeteaseAPIUrlBase)
	if nil != err {
		log.Fatal("failed to parse netease api url %s: %s", gNeteaseAPIUrlBase, err)
	}
	// netease api requires some cookies to work
	cookies.SetCookies(apiUrl, []*http.Cookie{
		&http.Cookie{Name: "appver", Value: "1.4.1.62460"},
		&http.Cookie{Name: "os", Value: "pc"},
		&http.Cookie{Name: "osver", Value: "Microsoft-Windows-7-Ultimate-Edition-build-7600-64bit"},
	})
	gNeteaseClient.Jar = cookies
}

type NeteaseRetStatus struct {
	StatusCode int    `json:"code"`
	Message    string `json:"message"`
}

type NeteaseAlbumRet struct {
	NeteaseRetStatus
	Album NeteaseAlbum `json:"album"`
}

type NeteaseAlbum struct {
	Songs []NeteaseSong `json:"songs"`
}

type NeteaseSongListRet struct {
	NeteaseRetStatus
	Songs []NeteaseSong `json:"songs"`
}

type NeteasePlayListRet struct {
	NeteaseRetStatus
	Result NeteasePlayList `json:"result"`
}

type NeteasePlayList struct {
	Songs []NeteaseSong `json:"tracks"`
}

type NeteaseSong struct {
	Artists []NeteaseArtist `json:"artists"`
	Name    string          `json:"name"`
	Url     string          `json:"mp3Url"`
}

func (ns *NeteaseSong) ArtistsString() string {
	arts := ""
	for i, _ := range ns.Artists {
		arts += (ns.Artists[i].Name + ",")
	}
	return strings.TrimRight(arts, ",")
}

type NeteaseArtist struct {
	Name string `json:"name"`
}

func GetNeteaseAlbum(albumId string) *SongList {
	url := gNeteaseAPIUrlBase + gNeteaseAlbumUrl + albumId
	ret := GetUrl(gNeteaseClient, url)
	sl := NewSongList()
	if nil == ret {
		return sl.SetAndLogErrorf("error accessing url %s", url)
	}

	var albumRet NeteaseAlbumRet
	err := json.Unmarshal(ret, &albumRet)
	if nil != err {
		return sl.SetAndLogErrorf("error parsing album return data from url %s: %s", url, err)
	}
	if gNeteaseRetOk != albumRet.StatusCode {
		return sl.SetAndLogErrorf("error getting url %s: %s", url, albumRet.Message)
	}

	for i, _ := range albumRet.Album.Songs {
		song := &albumRet.Album.Songs[i]
		sl.AddSong(&Song{
			Name:     song.Name,
			Url:      song.Url,
			Artists:  song.ArtistsString(),
			Provider: gNeteaseProvider,
		})
	}
	return sl
}

func GetNeteaseSongList(songs string) *SongList {
	url := fmt.Sprintf(gNeteaseAPIUrlBase+gNeteaseSongListUrl, songs)
	ret := GetUrl(gNeteaseClient, url)
	sl := NewSongList()
	if nil == ret {
		return sl.SetAndLogErrorf("error accessing url %s", url)
	}

	var songlistRet NeteaseSongListRet
	err := json.Unmarshal(ret, &songlistRet)
	if nil != err {
		return sl.SetAndLogErrorf("error parsing songlist return data from url %s: %s", url, err)
	}
	if gNeteaseRetOk != songlistRet.StatusCode {
		return sl.SetAndLogErrorf("error getting url %s: %s", url, songlistRet.Message)
	}

	for i, _ := range songlistRet.Songs {
		song := &songlistRet.Songs[i]
		sl.AddSong(&Song{
			Name:     song.Name,
			Url:      song.Url,
			Artists:  song.ArtistsString(),
			Provider: gNeteaseProvider,
		})
	}
	return sl
}

func GetNeteasePlayList(listId string) *SongList {
	url := gNeteaseAPIUrlBase + gNeteasePlayListUrl + listId
	ret := GetUrl(gNeteaseClient, url)
	sl := NewSongList()
	if nil == ret {
		return sl.SetAndLogErrorf("error accessing url %s", url)
	}

	var playlistRet NeteasePlayListRet
	err := json.Unmarshal(ret, &playlistRet)
	if nil != err {
		return sl.SetAndLogErrorf("error parsing playlist return data from url %s: %s", url, err)
	}
	if gNeteaseRetOk != playlistRet.StatusCode {
		return sl.SetAndLogErrorf("error getting url %s: %s", url, playlistRet.Message)
	}

	for i, _ := range playlistRet.Result.Songs {
		song := &playlistRet.Result.Songs[i]
		sl.AddSong(&Song{
			Name:     song.Name,
			Url:      song.Url,
			Artists:  song.ArtistsString(),
			Provider: gNeteaseProvider,
		})
	}
	return sl

}
