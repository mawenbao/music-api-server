package main

import (
	"bytes"
	"compress/gzip"
	"github.com/garyburd/redigo/redis"
	"io/ioutil"
	"log"
	"strings"
	"time"
)

const (
	gCacheKeyPrefix    = "mas:"
	gUrlCacheKeyPrefix = "url:"
)

var (
	// minimize cache key length
	gUrlKeyReplacer = strings.NewReplacer(
		"http://", "",
		"www.xiami.com", "xiami",
		"/app/android", "",
		"/app/iphone", "",
		"music.163.com", "163",
		"/api", "",
	)

	gRedisPool = &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", *gFlagRedisAddr)
			if nil != err {
				log.Printf("failed to connect to redis server %s: %s", *gFlagRedisAddr, err)
				return nil, err
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			if nil != err {
				log.Printf("failed to ping redis server %s: %s", *gFlagRedisAddr, err)
			}
			return err
		},
	}
)

func GenUrlCacheKey(url string) string {
	if "" == url {
		log.Println("failed to generate url cache key: url is empty")
		return ""
	}
	return gCacheKeyPrefix + gUrlCacheKeyPrefix + gUrlKeyReplacer.Replace(url)
}

func GetCache(key string, decompress bool) []byte {
	if "" == key {
		return nil
	}

	// get from redis cache
	redisConn := gRedisPool.Get()
	defer redisConn.Close()
	value, err := redisConn.Do("GET", key)
	if nil != err {
		log.Printf("failed to get from redis server %s: %s", *gFlagRedisAddr, err)
		return nil
	}
	if nil == value {
		return nil
	}

	valueBytes := value.([]byte)
	if !decompress {
		return valueBytes
	}
	// decompress value
	buff := bytes.NewBuffer(valueBytes)
	gzipRdr, err := gzip.NewReader(buff)
	if nil != err {
		log.Printf("failed to decompress cached value: %s", err)
		return nil
	}
	defer gzipRdr.Close()
	valueBytes, err = ioutil.ReadAll(gzipRdr)
	if nil != err {
		log.Printf("failed to read cached value: %s", err)
		return nil
	}
	return valueBytes
}

func SetCache(key string, value []byte, expires time.Duration, compress bool) bool {
	if "" == key {
		return false
	}

	var err error
	if compress {
		// compress value
		var buff bytes.Buffer
		gzipWtr := gzip.NewWriter(&buff)
		_, err = gzipWtr.Write(value)
		if nil != err {
			log.Printf("failed to compress value: %s", err)
			return false
		}
		err = gzipWtr.Close()
		if nil != err {
			log.Printf("failed to compress value: %s", err)
			return false
		}
		value = buff.Bytes()
	}

	// save value in redis cache
	redisConn := gRedisPool.Get()
	defer redisConn.Close()
	if expires != 0 {
		_, err = redisConn.Do("SETEX", key, expires.Seconds(), value)
	} else {
		// no expiration if expires is 0
		_, err = redisConn.Do("SET", key, value)
	}
	if nil != err {
		log.Printf("failed to send value to redis server %s: %s", *gFlagRedisAddr, err)
		return false
	}
	return true
}
