package main

import (
	"bytes"
	"testing"
)

func TestGetSetCache(t *testing.T) {
	provider := "xiami"
	reqType := "songlist"
	id := "12345"
	val := []byte("hello world")
	ret := SetCache(provider, reqType, id, val)
	if !ret {
		t.Fatalf("failed to set redis cache")
	}
	realval := GetCache(provider, reqType, id)
	if 0 != bytes.Compare(val, realval) {
		t.Fatalf("GetCache & SetCache failed, expected %s, but got %s.", val, realval)
	}
}
