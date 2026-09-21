package model

import (
	"github.com/ego008/goutils/json"
	"github.com/ego008/mdb"
	"go.etcd.io/bbolt"
)

type CustomRouter struct {
	Router   string
	MimeType string
	Content  string
}

func CustomRouterGetAll(db *mdb.DB, tx *bbolt.Tx) (objLst []CustomRouter) {
	var keyStart []byte
	for {
		var ok bool
		_ = db.HScanFunc(tx, "custom_router", keyStart, 20, func(key, val []byte) bool {
			keyStart = key
			obj := CustomRouter{}
			_ = json.Unmarshal(val, &obj)
			objLst = append(objLst, obj)
			ok = true
			return true
		})
		if !ok {
			break
		}
	}
	return
}

func CustomRouterSet(db *mdb.DB, tx *bbolt.Tx, obj CustomRouter) {
	jb, _ := json.Marshal(obj)
	_ = db.HSet(tx, "custom_router", []byte(obj.Router), jb)
}

func CustomRouterGetByKey(db *mdb.DB, tx *bbolt.Tx, key []byte) (obj CustomRouter) {
	_ = db.HGetFunc(tx, "custom_router", key, func(val []byte) error {
		_ = json.Unmarshal(val, &obj)
		return nil
	})
	return
}
