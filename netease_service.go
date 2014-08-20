package main

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
)

const (
	gNeteaseRetOk             = 200
	gNeteaseProvider          = "http://music.163.com/"
	gNeteaseAPIUrlBase        = "http://music.163.com/api"
	gNeteaseAlbumUrl          = "/album/"
	gNeteaseSongListUrl       = "/song/detail?ids=[%s]"
	gNeteasePlayListUrl       = "/playlist/detail?id="
	gNeteaseEIDCacheKeyPrefix = "163eid:" // encrypted dfsId
	gNeteaseMusicCDNUrlF      = "http://m1.music.126.net/%s/%s.mp3"
)

var (
	gNeteaseClient      = &http.Client{}
	gNeteaseEIDReplacer = strings.NewReplacer(
		"/", "_",
		"+", "-",
	)
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
	Artists            []NeteaseArtist    `json:"artists"`
	Name               string             `json:"name"`
	Url                string             `json:"mp3Url"`
	HighQualityMusic   NeteaseMusicDetail `json:"hMusic"`
	MediumQualityMusic NeteaseMusicDetail `json:"mMusic"`
	LowQualityMusic    NeteaseMusicDetail `json:"lMusic"`
}

func (song *NeteaseSong) UpdateUrl(quality string) *NeteaseSong {
	if "" == quality || gMusicQualityMedium == quality {
		return song
	}
	musicDetail := &song.HighQualityMusic
	if gMusicQualityLow == quality {
		musicDetail = &song.LowQualityMusic
	}
	song.Url = musicDetail.MakeUrl()
	return song
}

type NeteaseMusicDetail struct {
	Bitrate int `json:"bitrate"`
	DfsID   int `json:"dfsId"`
}

func (md *NeteaseMusicDetail) MakeUrl() string {
	strDfsID := strconv.Itoa(md.DfsID)
	// load eid from cache first
	eidKey := gCacheKeyPrefix + gNeteaseEIDCacheKeyPrefix + strDfsID
	eid := GetCache(eidKey, false)
	if nil == eid {
		// build encrypted dfsId, see https://github.com/yanunon/NeteaseCloudMusic/wiki/网易云音乐API分析#歌曲id加密代码
		byte1 := []byte("3go8&$8*3*3h0k(2)2")
		byte2 := []byte(strDfsID)
		byte1Len := len(byte1)
		for i := range byte2 {
			byte2[i] = byte2[i] ^ byte1[i%byte1Len]
		}
		sum := md5.Sum(byte2)
		var buff bytes.Buffer
		enc := base64.NewEncoder(base64.StdEncoding, &buff)
		_, err := enc.Write(sum[:])
		if nil != err {
			log.Printf("error encoding(base64) netease dfsId %s:%s", strDfsID, err)
			return ""
		}
		enc.Close()
		eid = []byte(gNeteaseEIDReplacer.Replace(buff.String()))
	}
	// update cache, no expiration, no compression
	SetCache(eidKey, eid, 0, false)
	return fmt.Sprintf(gNeteaseMusicCDNUrlF, eid, strDfsID)
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

func GetNeteaseAlbum(params *ReqParams) *SongList {
	url := gNeteaseAPIUrlBase + gNeteaseAlbumUrl + params.ID
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
		song := (&albumRet.Album.Songs[i]).UpdateUrl(params.Quality)
		sl.AddSong(&Song{
			Name:     song.Name,
			Url:      song.Url,
			Artists:  song.ArtistsString(),
			Provider: gNeteaseProvider,
		})
	}
	return sl
}

func GetNeteaseSongList(params *ReqParams) *SongList {
	url := fmt.Sprintf(gNeteaseAPIUrlBase+gNeteaseSongListUrl, params.ID)
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
		song := (&songlistRet.Songs[i]).UpdateUrl(params.Quality)
		sl.AddSong(&Song{
			Name:     song.Name,
			Url:      song.Url,
			Artists:  song.ArtistsString(),
			Provider: gNeteaseProvider,
		})
	}
	return sl
}

func GetNeteasePlayList(params *ReqParams) *SongList {
	url := gNeteaseAPIUrlBase + gNeteasePlayListUrl + params.ID
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
		song := (&playlistRet.Result.Songs[i]).UpdateUrl(params.Quality)
		sl.AddSong(&Song{
			Name:     song.Name,
			Url:      song.Url,
			Artists:  song.ArtistsString(),
			Provider: gNeteaseProvider,
		})
	}
	return sl

}
