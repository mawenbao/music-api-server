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
	gFlagCacheExpiration = flag.Int("expire", 3600, "expiry time(in seconds) of redis cache, default is one hour, 0 means no expiration")
	// available music service functions
	gAvailableGetMusicFuncs = []interface{}{
		GetXiamiSongList,
		GetXiamiAlbum,
		GetXiamiCollect,
		GetNeteaseSongList,
		GetNeteaseAlbum,
		GetNeteasePlayList,
	}
	gGetMusicFuncMap = make(map[string]interface{})
)

func init() {
	// init getmusic function map
	for _, f := range gAvailableGetMusicFuncs {
		gGetMusicFuncMap[getLowerFuncName(f)] = f
	}
}

type Song struct {
	Name     string `json:"name"`
	Url      string `json:"url"`
	LrcUrl   string `json:"lrc_url"`
	Artists  string `json:"artists"`
	Provider string `json:"provider"`
}

func (s *Song) ToJsonString() string {
	jsonStr, err := json.Marshal(s)
	if err != nil {
		log.Printf("error generating song json string: %s", err)
		return "error"
	}
	return string(jsonStr)
}

type SongList struct {
	Status string `json:"status"`
	ErrMsg string `json:"msg"`
	Songs  []Song `json:"songs"`
}

func NewSongList() *SongList {
	return &SongList{
		Status: "ok",
		ErrMsg: "",
		Songs:  []Song{},
	}
}

func (sl *SongList) SetAndLogErrorf(format string, args ...interface{}) *SongList {
	sl.Status = "failed"
	sl.ErrMsg = fmt.Sprintf(format, args...)
	log.Printf(sl.ErrMsg)
	return sl
}

func (sl *SongList) IsFailed() bool {
	if "failed" == sl.Status || "" != sl.ErrMsg {
		return true
	}
	return false
}

func (sl *SongList) CheckStatus() *SongList {
	if "" == sl.Status && "" == sl.ErrMsg && 0 != len(sl.Songs) {
		sl.Status = "ok"
	}
	return sl
}

func (sl *SongList) AddSong(s *Song) *SongList {
	sl.Songs = append(sl.Songs, *s)
	return sl
}

func (sl *SongList) Concat(ol *SongList) *SongList {
	if ol == nil || ol.Songs == nil {
		return sl
	}
	if ol.IsFailed() || nil == sl.Songs {
		return ol
	}
	sl.Songs = append(sl.Songs, ol.Songs...)
	return sl
}

func (sl *SongList) ToJsonString() string {
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
				result = []byte(callGetMusicFunc(myGetMusicFunc, id).CheckStatus().ToJsonString())
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
	log.Printf("Listening at %s ...\n", serverAddr)
	log.Fatal(httpServer.ListenAndServe())
}
