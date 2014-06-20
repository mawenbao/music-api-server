package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var (
	gFailStringInvalidReq = []byte(`{status: "failed", msg: "invalid request argument"}`)
	// command line options
	gFlagRedisAddr       = flag.String("redis", "localhost:6379", "address(host:port) of redis server")
	gFlagListenPort      = flag.Int("port", 9099, "port to listen on")
	gFlagLogfile         = flag.String("log", "", "path of the log file")
	gFlagCacheExpiration = flag.Int("expire", 7200, "expiry time(in seconds) of redis cache, default is two hours")
	// available music service functions
	gGetMusicFuncMap = map[string]interface{}{
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
	fmt.Println("Usage %s [-redis <redis server port>][-port <listen port>][-log <log file>][-expire <expiry time>]")
	fmt.Println("Flags:")
	flag.PrintDefaults()
}

func getLowerFuncName(i interface{}) string {
	funcName := strings.ToLower(runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name())
	return strings.Split(funcName, ".")[1]
}

func callGetMusicFunc(myFunc, param interface{}) *SongList {
	return reflect.ValueOf(myFunc).Call(
		[]reflect.Value{
			reflect.ValueOf(param),
		})[0].Interface().(*SongList)
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
			// cache missed
			myGetMusicFunc, ok := gGetMusicFuncMap["get"+provider+reqType]
			if !ok {
				result = gFailStringInvalidReq
			} else {
				// fetch and parse music data
				sl := callGetMusicFunc(myGetMusicFunc, id)
				if nil == sl {
					result = gFailStringInvalidReq
				} else {
					result = []byte(sl.ToString())
				}
				// update cache
				SetCache(provider, reqType, id, time.Duration(*gFlagCacheExpiration)*time.Second, result)
			}
		}
		if "" != callback {
			// jsonp
			result = []byte(callback + "(" + string(result) + ");")
		}
		w.Write(result)
	})
	return servMux
}

func main() {
	flag.Usage = showUsage
	flag.Parse()

	// init log
	if "" != *gFlagLogfile {
		logfile, err := os.OpenFile(*gFlagLogfile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if nil != err {
			log.Fatalf("failed to open/create log file %s: %s", *gFlagLogfile, err)
		}
		defer logfile.Close()
		log.SetOutput(logfile)
	}

	// init http server
	serverAddr := ":" + strconv.Itoa(*gFlagListenPort)
	httpServer := &http.Server{
		Addr:    serverAddr,
		Handler: createServMux(),
	}
	// start server
	log.Printf("Start music api server ...")
	fmt.Printf("Listening at %s ...\n", serverAddr)
	log.Fatal(httpServer.ListenAndServe())
}
