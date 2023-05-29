package model

import (
	"github.com/ego008/goutils/json"
	"github.com/ego008/sdb"
)

type IpInfo struct {
	Ip      string
	Names   string // join with `,`
	AddTime int64
	UpTime  int64 // last update time
}

func IpInfoGetByKeyStart(db *sdb.DB, keyStart string, limit int) (items []IpInfo) {
	var kst []byte
	if len(keyStart) > 0 {
		kst = sdb.S2b(keyStart)
	}
	rs := db.Hscan(TbnIpInfo, kst, limit)
	if !rs.OK() {
		return
	}
	rs.KvEach(func(_, value sdb.BS) {
		item := IpInfo{}
		err := json.Unmarshal(value, &item)
		if err != nil {
			return
		}
		items = append(items, item)
	})

	return
}
