package model

import (
	"github.com/ego008/mdb"
	"go.etcd.io/bbolt"

	"goyoubbs/util"
	"strconv"
	"time"
)

// SiteInfo 网站信息
type SiteInfo struct {
	WeekNum  string // 时间，周数
	Days     string // 建站时间
	UserNum  uint64 // 用户数
	NodeNum  uint64 // 节点数
	TagNum   uint64 // 标签数
	PostNum  uint64 // 帖子数
	ReplyNum uint64 // 回复数
}

func GetSiteInfo(db *mdb.DB, tx *bbolt.Tx) (si SiteInfo) {
	commentTb := "comment"
	siteCreateTb := "site_create_time"

	keys := [][]byte{
		[]byte(UserTbName),
		[]byte(NodeTbName),
		[]byte(TagTbName),
		[]byte(TopicTbName),
		[]byte(commentTb),
		[]byte(siteCreateTb),
	}

	tmpMap := map[string][]byte{}

	_ = db.HMGetFunc(tx, CountTb, keys, func(k []byte, v []byte) error {
		tmpMap[string(k)] = v
		return nil
	})

	if v, ok := tmpMap[UserTbName]; ok {
		si.UserNum = mdb.B2i(v)
	}

	if v, ok := tmpMap[NodeTbName]; ok {
		si.NodeNum = mdb.B2i(v)
	}
	if v, ok := tmpMap[TagTbName]; ok {
		si.TagNum = mdb.B2i(v)
	}
	if v, ok := tmpMap[TopicTbName]; ok {
		si.PostNum = mdb.B2i(v)
	}
	if v, ok := tmpMap[commentTb]; ok {
		si.ReplyNum = mdb.B2i(v)
	}

	var siteCreateTime uint64
	if v, ok := tmpMap[siteCreateTb]; ok {
		siteCreateTime = mdb.B2i(v)
	} else {
		// 冗余，一般不会发生
		siteCreateTime = uint64(time.Now().UTC().Unix())
	}

	///
	then := time.Unix(int64(siteCreateTime), 0)
	diff := time.Now().UTC().Sub(then)
	days := int(diff.Hours() / 24)
	years := days / 365
	day := days % 365
	if years > 0 {
		si.Days = strconv.Itoa(years) + "年"
	}
	if day > 0 {
		si.Days += strconv.Itoa(day) + "天"
	}
	if si.Days == "" {
		si.Days = "1天"
	}

	t := time.Now().UTC().Add(TimeOffSet)
	_, wkn := t.ISOWeek()
	si.WeekNum = util.TimeFmt(t.Unix(), "2006-01-02 ") + strconv.Itoa(wkn) + " week"
	return
}
