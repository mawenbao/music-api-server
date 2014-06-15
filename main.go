package main

import (
    "log"
    "io/ioutil"
    "net/http"
    "encoding/json"
)

type Song struct {
    Name string `json:"song_title"`
    Url string `json:"song_src"`
    Author string `json:"song_author"`
    Provider string `json:"song_provider"`
}

type SongList struct {
    Songs []Song `json:"songs"`
}

func (sl *SongList) AddSong(s Song) *SongList {
    sl.Songs = append(sl.Songs, s)
    return sl
}

func (sl *SongList) Concat(ol *SongList) *SongList {
    if ol == nil || ol.Songs == nil {
        return sl
    }
    if sl.Songs == nil {
        return ol
    }
    sl.Songs = append(sl.Songs, ol.Songs...)
    return sl
}

func (sl *SongList) ToString() string {
    jsonStr, err := json.Marshal(sl)
    if err != nil {
        log.Printf("error generating json string: %s", err)
        return "error"
    }
    return string(jsonStr)
}

func GetUrl(client *http.Client, url string) []byte {
    resp, err := client.Get(url)
    if err != nil {
        log.Printf("error get url %s: %s", url, err)
        return nil
    }
    defer resp.Body.Close()
    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        log.Printf("error getting response body from url %s: %s", url, err)
        return nil
    }
    return body
}

func main() {
    log.Println(GetXiamiSong("1772292423").ToString())
}

