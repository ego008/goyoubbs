package model

import (
	"github.com/VictoriaMetrics/fastcache"
	"github.com/ego008/goutils/json"
	"github.com/ego008/mdb"
	"go.etcd.io/bbolt"

	"goyoubbs/util"
	"sort"
)

type Link struct {
	ID    uint64
	Name  string
	Url   string
	Score int
}

func LinkGetById(db *mdb.DB, tx *bbolt.Tx, lid string) Link {
	var item Link
	_ = db.HGetFunc(tx, "link", mdb.DS2b(lid), func(val []byte) error {
		_ = json.Unmarshal(val, &item)
		return nil
	})
	return item
}

func LinkSet(db *mdb.DB, tx *bbolt.Tx, obj Link) {
	if obj.ID == 0 {
		// add
		var newId uint64
		_ = db.HRScanFunc(tx, "link", nil, 1, func(key, _ []byte) bool {
			newId = mdb.B2i(key)
			return true
		})
		newId++
		obj.ID = newId
	}
	jb, _ := json.Marshal(obj)
	_ = db.HSet(tx, "link", mdb.I2b(obj.ID), jb)
}

func LinkList(mc *fastcache.Cache, db *mdb.DB, tx *bbolt.Tx, getAll bool) (objLst []Link) {
	mcKey := []byte("LinkList")
	if !getAll {
		if _, exist := util.ObjCachedGet(mc, mcKey, &objLst, false); exist {
			return
		}
	}

	itemMap := map[uint64]Link{}

	startKey := []byte("")

	for {
		var ok bool
		_ = db.HScanFunc(tx, "link", startKey, 20, func(key, val []byte) bool {
			startKey = key
			item := Link{}
			_ = json.Unmarshal(val, &item)
			if getAll {
				// included score == 0
				itemMap[mdb.B2i(key)] = item
			} else {
				if item.Score > 0 {
					itemMap[mdb.B2i(key)] = item
				}
			}
			ok = true
			return true
		})
		if !ok {
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
