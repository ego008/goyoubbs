package model

import (
	"github.com/VictoriaMetrics/fastcache"
	"github.com/ego008/sdb"
	"goyoubbs/util"
)

const TagTbName = "tag"

type TagFontSize struct {
	Name string
	Size int
}

func GetTagsForSide(mc *fastcache.Cache, db *sdb.DB, limit int) (tagLst []TagFontSize) {
	mcKey := []byte("GetTagsForSide")
	if _, exist := util.ObjCachedGet(mc, mcKey, &tagLst, false); exist {
		return
	}

	db.Zrscan("tag_article_num", nil, nil, limit).KvEach(func(key, value sdb.BS) {
		num := sdb.B2i(value)
		//fontSize := math.Ceil(3*math.Log(float64(num+1)) + tagBaseFontSize)
		tag := key.String()
		tagLst = append(tagLst, TagFontSize{
			Name: tag,
			Size: int(num)})
	})

	// set to mc
	util.ObjCachedSet(mc, mcKey, tagLst)

	return
}
