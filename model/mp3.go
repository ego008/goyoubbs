package model

import (
	"github.com/ego008/goutils/json"
	"github.com/ego008/sdb"
)

type Mp3Info struct {
	Title   string
	Song    string
	Word    string
	Singer0 string
	Singer  string
	Path    string
}

func Mp3InfoSet(db *sdb.DB, key string) {
	kb := []byte(key)
	if db.Hget(TbnMp3Info, kb).OK() {
		return
	}
	obj := Mp3Info{Path: key}
	if jb, err := json.Marshal(obj); err != nil {
		return
	} else {
		_ = db.Hset(TbnMp3Info, kb, jb)
	}
}
