package model

import (
	"github.com/VictoriaMetrics/fastcache"
	"github.com/ego008/mdb"
	"go.etcd.io/bbolt"

	"goyoubbs/util"
)

const TagTbName = "tag"

type TagFontSize struct {
	Name string
	Size int
}

func GetTagsForSide(mc *fastcache.Cache, db *mdb.DB, tx *bbolt.Tx, limit int) (tagLst []TagFontSize) {
	mcKey := []byte("GetTagsForSide")
	if _, exist := util.ObjCachedGet(mc, mcKey, &tagLst, false); exist {
		return
	}

	_ = db.ZRScanFunc(tx, "tag_article_num", nil, 0, 0, limit, func(key []byte, score uint64) bool {
		tagLst = append(tagLst, TagFontSize{
			Name: string(key),
			Size: int(score)})
		return true
	})

	// set to mc
	util.ObjCachedSet(mc, mcKey, tagLst)

	return
}
