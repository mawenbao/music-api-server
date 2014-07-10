package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	gMusicQualityLow    = "low"
	gMusicQualityMedium = "medium"
	gMusicQualityHigh   = "high"
)

var (
	// command line options
	gFlagRedisAddr       = flag.String("redis", "localhost:6379", "address(host:port) of redis server")
	gFlagListenPort      = flag.Int("port", 9099, "port to listen on")
	gFlagLogfile         = flag.String("log", "", "path of the log file")
	gFlagCacheExpiration = flag.Int("expire", 3600, "expiry time(in seconds) of redis cache, default is one hour, 0 means no expiration")
	gCacheExpiryTime     = time.Hour
	// available music service functions
	gAvailableGetMusicFuncs = []GetMusicFunc{
		GetXiamiSongList,
		GetXiamiAlbum,
		GetXiamiCollect,
		GetNeteaseSongList,
		GetNeteaseAlbum,
		GetNeteasePlayList,
	}
	gGetMusicFuncMap = make(map[string]GetMusicFunc)
)

func init() {
	// init getmusic function map
	for _, f := range gAvailableGetMusicFuncs {
		gGetMusicFuncMap[getLowerFuncName(f)] = f
	}
}

type GetMusicFunc func(*ReqParams) *SongList

type ReqParams struct {
	ID,
	Callback,
	Provider,
	Quality,
	ReqType string
}

func ParseReqParams(queries url.Values) *ReqParams {
	params := &ReqParams{}
	params.Callback = strings.ToLower(queries.Get("c"))
	params.Provider = strings.ToLower(queries.Get("p"))
	params.ReqType = strings.ToLower(strings.TrimSpace(queries.Get("t")))
	params.ID = strings.TrimSpace(queries.Get("i"))
	params.Quality = strings.ToLower(queries.Get("q"))
	return params
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
	cacheKey := GenUrlCacheKey(url)
	if "" == cacheKey {
		return nil
	}
	// try to load from cache first
	body := GetCache(cacheKey, true)
	if nil != body {
		return body
	}

	// cache missed, do http request
	resp, err := client.Get(url)
	if err != nil {
		log.Printf("error get url %s: %s", url, err)
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

func showUsage() {
	fmt.Println("Usage %s [-redis <redis server port>][-port <listen port>][-log <log file>][-expire <expiry time>]")
	fmt.Println("Flags:")
	flag.PrintDefaults()
}

func getLowerFuncName(i interface{}) string {
	funcName := strings.ToLower(runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name())
	return strings.Split(funcName, ".")[1]
}

func createServMux() http.Handler {
	servMux := http.NewServeMux()
	servMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		params := ParseReqParams(r.URL.Query())
		myGetMusicFunc, ok := gGetMusicFuncMap["get"+params.Provider+params.ReqType]
		var result []byte
		if !ok {
			result = []byte(NewSongList().SetAndLogErrorf("invalid request arguments").ToJsonString())
		} else {
			// fetch and parse music data
			result = []byte(myGetMusicFunc(params).ToJsonString())
		}
		if "" != params.Callback {
			// jsonp
			result = []byte(params.Callback + "(" + string(result) + ");")
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

	// set cache expiry time
	gCacheExpiryTime = time.Duration(*gFlagCacheExpiration) * time.Second

	// init http server
	serverAddr := ":" + strconv.Itoa(*gFlagListenPort)
	httpServer := &http.Server{
		Addr:    serverAddr,
		Handler: createServMux(),
	}
	// start server
	log.Println("Start music api server ...")
	log.Printf("Listening at %s ...", serverAddr)
	log.Fatal(httpServer.ListenAndServe())
}
