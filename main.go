package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
	"runtime"
	"strconv"
	"strings"
)

var (
	gFailStringInvalidReq = []byte(`{status: "failed", msg: "invalid request argument"}`)
	gFlagRedisAddr        = flag.String("redis", "localhost:6379", "address(host:port) of redis server")
	gListenPort           = flag.Int("port", 9099, "port to listen on")
	gFuncMap              = map[string]interface{}{
		getLowerFuncName(GetXiamiSongList): GetXiamiSongList,
		getLowerFuncName(GetXiamiAlbum):    GetXiamiAlbum,
		getLowerFuncName(GetXiamiCollect):  GetXiamiCollect,
	}
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

func getLowerFuncName(i interface{}) string {
	funcName := strings.ToLower(runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name())
	return strings.Split(funcName, ".")[1]
}

func callFunc(myFunc, param interface{}) []reflect.Value {
	return reflect.ValueOf(myFunc).Call([]reflect.Value{reflect.ValueOf(param)})
}

func createServMux() http.Handler {
	servMux := http.NewServeMux()
	servMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		queries := r.URL.Query()
		callback := strings.ToLower(queries.Get("c"))
		provider := strings.ToLower(queries.Get("p"))
		reqType := strings.ToLower(strings.TrimSpace(queries.Get("t")))
		id := strings.TrimSpace(queries.Get("i"))
		// get cache first
		result := GetCache(provider, reqType, id)
		if nil == result {
			myGetFunc, ok := gFuncMap["get"+provider+reqType]
			if !ok {
				result = gFailStringInvalidReq
			} else {
				sl := callFunc(myGetFunc, id)[0].Interface().(*SongList)
				result = []byte(sl.ToString())
				// update cache
				SetCache(provider, reqType, id, result)
			}
		}
		if "" != callback {
			result = []byte(callback + "(" + string(result) + ");")
		}
		w.Write(result)
	})
	return servMux
}

func main() {
	flag.Usage = showUsage
	flag.Parse()

	// start http server
	serverAddr := ":" + strconv.Itoa(*gListenPort)
	httpServer := &http.Server{
		Addr:    serverAddr,
		Handler: createServMux(),
	}

	log.Printf("Listening at %s ...", serverAddr)
	log.Fatal(httpServer.ListenAndServe())
}
