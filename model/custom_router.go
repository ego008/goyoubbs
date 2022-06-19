package model

import (
	"github.com/ego008/goutils/json"
	"github.com/ego008/sdb"
)

type CustomRouter struct {
	Router   string
	MimeType string
	Content  string
}

func CustomRouterGetAll(db *sdb.DB) (objLst []CustomRouter) {
	var keyStart []byte
	for {
		rs := db.Hscan("custom_router", keyStart, 20)
		if !rs.OK() {
			break
		}
		rs.KvEach(func(key, value sdb.BS) {
			keyStart = key
			obj := CustomRouter{}
			_ = json.Unmarshal(value, &obj)
			objLst = append(objLst, obj)
		})
	}
	return
}

func CustomRouterSet(db *sdb.DB, obj CustomRouter) {
	jb, _ := json.Marshal(obj)
	_ = db.Hset("custom_router", []byte(obj.Router), jb)
}

func CustomRouterGetByKey(db *sdb.DB, key []byte) (obj CustomRouter) {
	rs := db.Hget("custom_router", key)
	if !rs.OK() {
		return
	}
	_ = json.Unmarshal(rs.Bytes(), &obj)
	return
}
