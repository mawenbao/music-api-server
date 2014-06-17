package main

import (
	"bytes"
	"compress/gzip"
	"github.com/garyburd/redigo/redis"
	"io/ioutil"
	"log"
	"time"
)

var (
	gProviderMap = map[string]string{
		"xiami":    "x",
		"netease":  "n",
		"163music": "n",
	}

	gReqTypeMap = map[string]string{
		"album":    "a",
		"collect":  "c",
		"songlist": "s",
	}

	gRedisPool = &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", *gFlagRedisAddr)
			if nil != err {
				log.Printf("failed to connect to redis server %s: %s", gFlagRedisAddr, err)
				return nil, err
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			if nil != err {
				log.Printf("failed to ping redis server %s: %s", gFlagRedisAddr, err)
			}
			return err
		},
	}
)

func GenCacheKey(provider, reqType, id string) string {
	if "" == id {
		log.Printf("failed to generate cache key: id is empty")
		return ""
	}
	provider, ok := gProviderMap[provider]
	if !ok {
		log.Printf("failed to generate cache key: provider %s not supported.", provider)
		return ""
	}
	reqType, ok = gReqTypeMap[reqType]
	if !ok {
		log.Printf("failed to generate cache key: request type %s not supported.", reqType)
		return ""
	}
	return provider + "|" + reqType + id
}

func GetCache(provider, reqType, id string) []byte {
	key := GenCacheKey(provider, reqType, id)
	if "" == key {
		return nil
	}

	redisConn := gRedisPool.Get()
	defer redisConn.Close()
	value, err := redisConn.Do("GET", key)
	if nil != err {
		log.Printf("failed to get from redis server %s: %s", gFlagRedisAddr, err)
		return nil
	}
	if nil == value {
		return nil
	}

	buff := bytes.NewBuffer(value.([]byte))
	gzipRdr, err := gzip.NewReader(buff)
	if nil != err {
		log.Printf("failed to decompress cached value: %s", err)
		return nil
	}
	defer gzipRdr.Close()
	data, err := ioutil.ReadAll(gzipRdr)
	if nil != err {
		log.Printf("failed to read cached value: %s", err)
		return nil
	}
	return data
}

func SetCache(provider, reqType, id string, value []byte) bool {
	key := GenCacheKey(provider, reqType, id)
	if "" == key {
		return false
	}

	var buff bytes.Buffer
	gzipWtr := gzip.NewWriter(&buff)
	_, err := gzipWtr.Write(value)
	if nil != err {
		log.Printf("failed to compress value: %s", err)
		return false
	}
	err = gzipWtr.Close()
	if nil != err {
		log.Printf("failed to compress value: %s", err)
		return false
	}

	redisConn := gRedisPool.Get()
	defer redisConn.Close()
	_, err = redisConn.Do("SET", key, buff.Bytes())
	if nil != err {
		log.Printf("failed to send value to redis server %s: %s", gFlagRedisAddr, err)
		return false
	}
	return true
}
