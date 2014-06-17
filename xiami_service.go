package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

const (
	gXiamiSongSplitter = ","
	gXiamiRetOK        = "ok"
	gXiamiRetFail      = "failed"
	gXiamiProvider     = "XiaMi Music Service"
	gXiamiAPIUrlBase   = "http://www.xiami.com/app"
	gXiamiSongUrl      = "/android/song/id/"
	gXiamiAlbumUrl     = "/iphone/album/id/"
	gXiamiCollectUrl   = "/android/collect?id="
)

var (
	gXiamiClient = &http.Client{}
)

type XiamiSongRet struct {
	Status  string    `json:"status"`
	Message string    `json:"msg"`
	Song    XiamiSong `json:"song"`
}

type XiamiSong struct {
	Name   string `json:"song_name"`
	Url    string `json:"song_location"`
	Lrc    string `json:"song_lrc"`
	Author string `json:"artist_name"`
}

type XiamiCollectRet struct {
	Status  string       `json:"status"`
	Message string       `json:"msg"`
	Collect XiamiCollect `json:"collect"`
}

type XiamiCollect struct {
	Songs []XiamiCollectSong `json:"songs"`
}

type XiamiCollectSong struct {
	Name   string `json:"name"`
	Url    string `json:"location"`
	Lrc    string `json:"lyric"`
	Author string `json:"singers"`
}

type XiamiAlbumRet struct {
	Status  string     `json:"status"`
	Message string     `json:"msg"`
	Album   XiamiAlbum `json:"album"`
}

type XiamiAlbum struct {
	Songs map[string]XiamiCollectSong
}

func GetXiamiSong(songId string) *Song {
	url := gXiamiAPIUrlBase + gXiamiSongUrl + strings.TrimSpace(songId)
	ret := GetUrl(gXiamiClient, url)
	if nil == ret {
		return nil
	}

	var songret XiamiSongRet
	err := json.Unmarshal(ret, &songret)
	if nil != err {
		log.Printf("error parsing song info from url %s: %s", url, err)
		return nil
	}
	if gXiamiRetOK != songret.Status {
		log.Printf("error getting url %s: %s", url, songret.Message)
		return nil
	}
	return &Song{
		Name:     songret.Song.Name,
		Url:      songret.Song.Url,
		Author:   songret.Song.Author,
		LrcUrl:   songret.Song.Lrc,
		Provider: gXiamiProvider,
	}
}

func GetXiamiSongList(songs string) *SongList {
	sl := &SongList{}
	for _, sid := range strings.Split(songs, gXiamiSongSplitter) {
		xiamiSong := GetXiamiSong(strings.TrimSpace(sid))
		if nil != xiamiSong {
			sl.AddSong(xiamiSong)
		}
	}
	return sl
}

func GetXiamiCollect(collectId string) *SongList {
	url := gXiamiAPIUrlBase + gXiamiCollectUrl + strings.TrimSpace(collectId)
	ret := GetUrl(gXiamiClient, url)
	if nil == ret {
		return nil
	}
	var collectRet XiamiCollectRet
	err := json.Unmarshal(ret, &collectRet)
	if nil != err {
		log.Printf("error parsing collect data from url %s: %s", url, err)
		return nil
	}
	if gXiamiRetOK != collectRet.Status {
		log.Printf("error getting url %s: %s", url, collectRet.Message)
		return nil
	}
	sl := &SongList{}
	for _, song := range collectRet.Collect.Songs {
		sl.AddSong(&Song{
			Name:     song.Name,
			Url:      song.Url,
			LrcUrl:   song.Lrc,
			Author:   song.Author,
			Provider: gXiamiProvider,
		})
	}
	return sl
}

func GetXiamiAlbum(albumId string) *SongList {
	url := gXiamiAPIUrlBase + gXiamiAlbumUrl + strings.TrimSpace(albumId)
	ret := GetUrl(gXiamiClient, url)
	if nil == ret {
		return nil
	}
	var albumRet XiamiAlbumRet
	err := json.Unmarshal(ret, &albumRet)
	if nil != err {
		log.Printf("error parsing album data from url %s: %s", url, err)
		return nil
	}
	if gXiamiRetOK != albumRet.Status {
		log.Printf("error getting url %s: %s", url, albumRet.Message)
		return nil
	}
	sl := &SongList{}
	for _, song := range albumRet.Album.Songs {
		sl.AddSong(&Song{
			Name:     song.Name,
			Url:      song.Url,
			LrcUrl:   song.Lrc,
			Author:   song.Author,
			Provider: gXiamiProvider,
		})
	}
	return sl
}
