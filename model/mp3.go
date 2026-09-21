package model

import (
	"github.com/ego008/goutils/json"
	"github.com/ego008/mdb"
	"go.etcd.io/bbolt"
)

type Mp3Info struct {
	Title   string
	Song    string
	Word    string
	Singer0 string
	Singer  string
	Path    string
}

func Mp3InfoSet(db *mdb.DB, tx *bbolt.Tx, key string) {
	kb := []byte(key)
	if db.HKeyExist(tx, TbnMp3Info, kb) {
		return
	}
	obj := Mp3Info{Path: key}
	if jb, err := json.Marshal(obj); err != nil {
		return
	} else {
		_ = db.HSet(tx, TbnMp3Info, kb, jb)
	}
}
