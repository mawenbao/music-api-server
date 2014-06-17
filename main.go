package main

import (
	"flag"
	"fmt"
	"log"
    "strconv"
	"encoding/json"
	"io/ioutil"
	"net/http"
)

var (
	gFlagRedisAddr = flag.String("redis", "localhost:6379", "address(host:port) of redis server")
	gListenPort    = flag.Int("port", 9099, "port to listen on")
)

type Song struct {
	Name     string `json:"song_title"`
	Url      string `json:"song_src"`
	LrcUrl   string `json:"song_lrc"`
	Author   string `json:"song_author"`
	Provider string `json:"song_provider"`
}

func (s *Song) ToString() string {
	jsonStr, err := json.Marshal(s)
	if err != nil {
		log.Printf("error generating song json string: %s", err)
		return "error"
	}
	return string(jsonStr)
}

type SongList struct {
	Songs []Song `json:"songs"`
}

func (sl *SongList) AddSong(s *Song) *SongList {
	sl.Songs = append(sl.Songs, *s)
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

func showUsage() {
	fmt.Println("Usage %s [-redis <redis server port>][-port <listen port>]")
	fmt.Println("Flags:")
	flag.PrintDefaults()
}

func main() {
	//log.Println(GetXiamiSongList("1772292423,20526").ToString())
	//log.Println(GetXiamiCollect("31181538").ToString())
	//log.Println(GetXiamiAlbum("2649").ToString())
	flag.Usage = showUsage
	flag.Parse()

    // start http server
    serverAddr = ":" + strconv.Itoa(*gListenPort)
    httpServer = &http.Server{
        Addr: serverAddr,
    }
    log.Printf("Listening on %s ...", serverAddr)
    log.Fatal(httpServer.ListenAndServe())
}

