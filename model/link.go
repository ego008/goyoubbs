package model

import (
	"github.com/VictoriaMetrics/fastcache"
	"github.com/ego008/goutils/json"
	"github.com/ego008/sdb"
	"goyoubbs/util"
	"sort"
)

type Link struct {
	ID    uint64
	Name  string
	Url   string
	Score int
}

func LinkGetById(db *sdb.DB, lid string) Link {
	var item Link
	rs := db.Hget("link", sdb.DS2b(lid))
	if rs.State == "ok" {
		_ = json.Unmarshal(rs.Data[0], &item)
	}
	return item
}

func LinkSet(db *sdb.DB, obj Link) {
	if obj.ID == 0 {
		// add
		var newId uint64
		db.Hrscan("link", nil, 1).KvEach(func(key, value sdb.BS) {
			newId = sdb.B2i(key.Bytes())
		})
		newId++
		obj.ID = newId
	}
	jb, _ := json.Marshal(obj)
	_ = db.Hset("link", sdb.I2b(obj.ID), jb)
}

func LinkList(mc *fastcache.Cache, db *sdb.DB, getAll bool) (objLst []Link) {
	mcKey := []byte("LinkList")
	if !getAll {
		if _, exist := util.ObjCachedGet(mc, mcKey, &objLst, false); exist {
			return
		}
	}

	itemMap := map[uint64]Link{}

	startKey := []byte("")

	for {
		rs := db.Hscan("link", startKey, 20)
		if rs.State == "ok" {
			for i := 0; i < len(rs.Data)-1; i += 2 {
				startKey = rs.Data[i]
				item := Link{}
				_ = json.Unmarshal(rs.Data[i+1], &item)
				if getAll {
					// included score == 0
					itemMap[sdb.B2i(rs.Data[i])] = item
				} else {
					if item.Score > 0 {
						itemMap[sdb.B2i(rs.Data[i])] = item
					}
				}
			}
		} else {
			break
		}
	}

	if len(itemMap) > 0 {
		type Kv struct {
			Key   uint64
			Value int
		}

		var ss []Kv
		for k, v := range itemMap {
			ss = append(ss, Kv{k, v.Score})
		}

		sort.Slice(ss, func(i, j int) bool {
			return ss[i].Value > ss[j].Value
		})

		for _, kv := range ss {
			objLst = append(objLst, itemMap[kv.Key])
		}

		// set to mc
		if !getAll {
			util.ObjCachedSet(mc, mcKey, objLst)
		}
	}

	return
}
