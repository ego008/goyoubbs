package model

import (
	"github.com/ego008/goutils/json"
	"github.com/ego008/sdb"
	"strconv"
	"time"
)

// SiteInfo 网站信息
type SiteInfo struct {
	Days     string // 建站时间
	UserNum  uint64 // 用户数
	NodeNum  uint64 // 节点数
	TagNum   uint64 // 标签数
	PostNum  uint64 // 帖子数
	ReplyNum uint64 // 回复数
}

func GetSiteInfo(db *sdb.DB) (si SiteInfo) {
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

	tmpMap := db.Hmget(CountTb, keys).Dict()

	if v, ok := tmpMap[UserTbName]; ok {
		si.UserNum = sdb.B2i(v)
	}

	if v, ok := tmpMap[NodeTbName]; ok {
		si.NodeNum = sdb.B2i(v)
	}
	if v, ok := tmpMap[TagTbName]; ok {
		si.TagNum = sdb.B2i(v)
	}
	if v, ok := tmpMap[TopicTbName]; ok {
		si.PostNum = sdb.B2i(v)
	}
	if v, ok := tmpMap[commentTb]; ok {
		si.ReplyNum = sdb.B2i(v)
	}

	var siteCreateTime uint64
	if v, ok := tmpMap[siteCreateTb]; ok {
		siteCreateTime = sdb.B2i(v)
	} else {
		rs2 := db.Hscan(UserTbName, []byte(""), 1)
		if rs2.State == "ok" {
			user := User{}
			_ = json.Unmarshal(rs2.Data[1], &user)
			siteCreateTime = user.RegTime
		} else {
			siteCreateTime = uint64(time.Now().UTC().Unix())
		}
		_ = db.Hset(CountTb, []byte(siteCreateTb), sdb.I2b(siteCreateTime))
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

	// fix for new site
	if si.NodeNum == 0 {
		newCid, err2 := db.Hincr(CountTb, []byte(NodeTbName), 1)
		if err2 == nil {
			obj := Node{
				ID:    newCid,
				Name:  "默认分类",
				About: "默认第一个分类",
			}
			jb, _ := json.Marshal(obj)
			_ = db.Hset(NodeTbName, sdb.I2b(obj.ID), jb)
			si.NodeNum = 1
		}
		// link
		LinkSet(db, Link{
			Name:  "youBBS",
			Url:   "https://youbbs.org",
			Score: 100,
		})
	}
	return
}
