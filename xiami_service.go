package main

import (
    "net/http"
    "encoding/json"
    "log"
)

const (
    gXiamiProvider = "XiaMi Music Service"
    gXiamiAPIUrlBase = "http://www.xiami.com/app"
    gXiamiSongUrl = "/android/song/id/"
    gXiamiAlbumUrl = "/iphone/album/id/"
    gXiamiCollectUrl = "/android/collect?id="
)

var (
    gXiamiClient = &http.Client{}
)

func GetXiamiSong(songId string) *Song {
    url := gXiamiAPIUrlBase + gXiamiSongUrl + songId
    ret := GetUrl(gXiamiClient, url)
    if ret == nil {
        return nil
    }
    var jsonmap map[string]json.RawMessage
    err := json.Unmarshal(ret, &jsonmap)
    if err != nil {
        log.Printf("error parsing return data from url %s: %s", url, err)
        return nil
    }

    var songmap map[string]string
    err = json.Unmarshal(jsonmap["song"], &songmap)
    if err != nil {
        log.Printf("error parsing song info from url %s: %s", url, err)
        return nil
    }
    return &Song{
        Name: songmap["song_name"],
        Url: songmap["song_location"],
        Author: songmap["artist_name"],
        Provider: gXiamiProvider,
    }
}

func GetXiamiSongList(songs string) *SongList {
}

func GetXiamiCollect(collectId string) *SongList {
}

