package model

import (
	"github.com/ego008/goutils/json"
	"github.com/ego008/mdb"
	"go.etcd.io/bbolt"
)

type IpInfo struct {
	Ip      string
	Names   string // join with `,`
	AddTime int64
	UpTime  int64 // last update time
}

func IpInfoGetByKeyStart(db *mdb.DB, tx *bbolt.Tx, keyStart string, limit int) (items []IpInfo) {
	var kst []byte
	if len(keyStart) > 0 {
		kst = mdb.S2b(keyStart)
	}

	_ = db.HScanFunc(tx, TbnIpInfo, kst, limit, func(key, val []byte) bool {
		item := IpInfo{}
		err := json.Unmarshal(val, &item)
		if err != nil {
			return true
		}
		items = append(items, item)
		return true
	})

	return
}
