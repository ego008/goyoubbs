package model

import (
	"github.com/VictoriaMetrics/fastcache"
	"github.com/ego008/goutils/json"
	"github.com/ego008/sdb"
	"github.com/tidwall/gjson"
	"goyoubbs/util"
	"html"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	TopicTbName       = "topic"
	TopicReviewTbName = "review_topic"
)

type Topic struct {
	ID       uint64
	NodeId   uint64
	UserId   uint64
	Title    string
	Content  string
	ClientIp string
	Tags     string
	AddTime  int64
	EditTime int64
	Comments uint64
}

type TopicRecForm struct {
	ID         uint64
	Act        string
	NodeId     uint64
	UserId     uint64
	Title      string
	Content    string
	Tags       string
	AddTime    int64
	AddTimeFmt string
}

// TopicTag 文章添加、编辑后传给后台任务的信息
type TopicTag struct {
	ID      uint64
	OldTags string
	NewTags string
}

// TopicLoc for site map
type TopicLoc struct {
	Id      uint64
	Title   string
	Slug    string
	AddTime int64
}

type TopicLi struct {
	ID    uint64
	Title string
}

type TopicLstLi struct {
	Topic
	FirstCon    string // 第一段，摘要
	AuthorName  string
	AddTimeFmt  string
	EditTimeFmt string
	NodeName    string
	AddYearShow string
	AddYear     string // 2013 添加年份，归档显示用
	AddMonth    string // NOV 添加月，归档显示用
	AddDate     string // 9 添加日，归档显示用
}

// TopicLstLiMsg for msg
type TopicLstLiMsg struct {
	TopicLstLi
	Href string // 点击链接 /t/10 | /t/10#comment-4
}

type TopicPageInfo struct {
	Items      []TopicLstLi
	HasPrev    bool
	HasNext    bool
	FirstKey   uint64
	FirstScore uint64
	LastKey    uint64
	LastScore  uint64
}

type TopicPageInfoMsg struct {
	Items []TopicLstLiMsg
}

// TopicFmt 详情页
type TopicFmt struct {
	Topic
	Name        string
	Views       uint64
	AddTimeFmt  string
	EditTimeFmt string
	ContentFmt  string
	ClockEmoji  string
	Relative    []TopicLi // 相关文章
}

type TopicFeed struct {
	Topic
	AddTimeFmt  string
	EditTimeFmt string
	Des         string
}

// TopicSet 单纯保存
func TopicSet(db *sdb.DB, obj Topic) Topic {
	jb, err := json.Marshal(obj)
	if err != nil {
		return obj
	}
	_ = db.Hset(TopicTbName, sdb.I2b(obj.ID), jb)
	return obj
}

func TopicAdd(mc *fastcache.Cache, db *sdb.DB, obj Topic) Topic {
	newId, _ := db.Hincr(CountTb, sdb.S2b(TopicTbName), 1) // id 自增
	obj.ID = newId
	jb, _ := json.Marshal(obj)
	_ = db.Hset(TopicTbName, sdb.I2b(obj.ID), jb)
	// 添加时间轴
	// 首页
	_ = db.Zset(TbnPostUpdate, sdb.I2b(obj.ID), uint64(obj.AddTime))
	// 分类页
	_ = db.Zset("topic_update:"+strconv.FormatUint(obj.NodeId, 10), sdb.I2b(obj.ID), uint64(obj.AddTime))
	// 个人主贴
	_ = db.Zset("user_topic:"+strconv.FormatUint(obj.UserId, 10), sdb.I2b(obj.ID), uint64(obj.AddTime))
	// 分类归档，按id、添加时间排序
	_ = db.Hset("topic_node:"+strconv.FormatUint(obj.NodeId, 10), sdb.I2b(obj.ID), sdb.I2b(uint64(obj.AddTime)))
	// 该分类的文章数
	_, _ = db.Hincr(NodeTopicNumTbName, sdb.I2b(obj.NodeId), 1)
	// 文章总数
	_, _ = db.Hincr(CountTb, sdb.S2b(TopicTbName+":all_number"), 1)
	// 删除分类缓存
	mc.Del([]byte("NodeGetAll"))
	return obj
}

// TopicDel 删除文章
func TopicDel(mc *fastcache.Cache, db *sdb.DB, obj Topic) {
	// 首页
	_ = db.Zdel(TbnPostUpdate, sdb.I2b(obj.ID))
	// 分类页
	_ = db.Zdel("topic_update:"+strconv.FormatUint(obj.NodeId, 10), sdb.I2b(obj.ID))
	// 个人主贴
	_ = db.Zdel("user_topic:"+strconv.FormatUint(obj.UserId, 10), sdb.I2b(obj.ID))
	// 分类归档，按id、添加时间排序
	_ = db.Hdel("topic_node:"+strconv.FormatUint(obj.NodeId, 10), sdb.I2b(obj.ID))
	// 该分类的文章数 -1
	_, _ = db.Hincr(NodeTopicNumTbName, sdb.I2b(obj.NodeId), -1)
	// 文章总数-1
	_, _ = db.Hincr(CountTb, sdb.S2b(TopicTbName+":all_number"), -1)
	// 标签
	if len(obj.Tags) > 0 {
		// 删除标签，参见 cronjob/topic_tag
		for _, tag := range strings.Split(obj.Tags, ",") {
			tagLower := strings.ToLower(tag)
			tagLowerB := sdb.S2b(tagLower)
			_ = db.Hdel("tag:"+tagLower, sdb.I2b(obj.ID))

			if db.Hscan("tag:"+tagLower, nil, 1).OK() {
				_, _ = db.Zincr("tag_article_num", tagLowerB, -1) // 热门标签排序
			} else {
				// 删除
				_ = db.Zdel("tag_article_num", tagLowerB)
				_ = db.Hdel("tag", tagLowerB)
				_, _ = db.Hincr("tag_count", sdb.S2b("tag"), -1)
			}
		}
	}
	// 文章实体
	_ = db.Hdel(TopicTbName, sdb.I2b(obj.ID))
	// 删除分类缓存
	mc.Del([]byte("NodeGetAll"))
	mc.Del([]byte("GetTagsForSide"))
}

func TopicGetById(db *sdb.DB, tid uint64) (obj Topic) {
	rs := db.Hget(TopicTbName, sdb.I2b(tid))
	if !rs.OK() {
		return
	}
	err := json.Unmarshal(rs.Bytes(), &obj)
	if err != nil {
		return
	}
	return
}

// TopicGetTitlesByIds 根据 ids 取 title ，返回id:title 的map
func TopicGetTitlesByIds(db *sdb.DB, ids []uint64) map[uint64]string {
	id2name := map[uint64]string{}
	if len(ids) == 0 {
		return id2name
	}
	var idsb [][]byte
	for _, k := range ids {
		idsb = append(idsb, sdb.I2b(k))
	}
	db.Hmget(TopicTbName, idsb).KvEach(func(key, value sdb.BS) {
		title := gjson.Get(value.String(), "Title").String()
		id2name[sdb.B2i(key.Bytes())] = title
	})
	return id2name
}

func TopicGetRelative(mc *fastcache.Cache, db *sdb.DB, aid uint64, tags string) (topLst []TopicLi) {
	if len(tags) == 0 {
		return
	}

	// get from mc
	mcKey := []byte("TopicGetRelative:" + strconv.FormatUint(aid, 10))
	if _, exist := util.ObjCachedGet(mc, mcKey, &topLst, false); exist {
		return
	}

	getMax := 10
	scanMax := 100

	tagsLow := strings.ToLower(tags)

	aidCount := map[uint64]int{}

	for _, tag := range strings.Split(tagsLow, ",") {
		rs := db.Hrscan("tag:"+tag, nil, scanMax)
		if rs.KvLen() > 0 {
			for i := 0; i < len(rs.Data)-1; i += 2 {
				aid2 := sdb.B2i(rs.Data[i].Bytes())
				if aid2 != aid {
					if _, ok := aidCount[aid2]; ok {
						aidCount[aid2] += 1
					} else {
						aidCount[aid2] = 1
					}
				}
			}
		}
	}

	if len(aidCount) > 0 {

		type Kv struct {
			Key   uint64
			Value int
		}

		var ss []Kv
		for k, v := range aidCount {
			ss = append(ss, Kv{k, v})
		}

		sort.Slice(ss, func(i, j int) bool {
			return ss[i].Value > ss[j].Value
		})

		var akeys [][]byte
		j := 0
		for _, kv := range ss {
			akeys = append(akeys, sdb.I2b(kv.Key))
			j++
			if j == getMax {
				break
			}
		}

		rs := db.Hmget(TopicTbName, akeys)
		if rs.KvLen() > 0 {
			for i := 0; i < len(rs.Data)-1; i += 2 {
				item := TopicLi{}
				_ = json.Unmarshal(rs.Data[i+1].Bytes(), &item)
				topLst = append(topLst, item)
			}
		}
	}

	// set to mc
	if len(topLst) > 0 {
		util.ObjCachedSet(mc, mcKey, topLst)
	}

	return
}

// GetTopicList 分页获取首页、分类页的帖子
func GetTopicList(db *sdb.DB, cmd, tb, key, score string, limit int) TopicPageInfo {
	var items []TopicLstLi
	var keys [][]byte
	var hasPrev, hasNext bool
	var firstKey, firstScore, lastKey, lastScore uint64

	topicEditTimeMap := map[uint64]uint64{} // 文章编辑时间 / 最后评论时间
	keyStart := sdb.DS2b(key)
	scoreStart := sdb.DS2b(score)
	if cmd == "zrscan" {
		rs := db.Zrscan(tb, keyStart, scoreStart, limit)
		if rs.KvLen() > 0 {
			for i := 0; i < (len(rs.Data) - 1); i += 2 {
				keys = append(keys, rs.Data[i].Bytes())
				// 分页游标
				ki := sdb.B2i(rs.Data[i].Bytes())
				si := sdb.B2i(rs.Data[i+1].Bytes())
				topicEditTimeMap[ki] = si
				if i == 0 {
					firstKey = ki
					firstScore = si
				} else {
					lastKey = ki
					lastScore = si
				}
			}
		}
	} else if cmd == "zscan" {
		rs := db.Zscan(tb, keyStart, scoreStart, limit)
		if rs.KvLen() > 0 {
			for i := len(rs.Data) - 2; i >= 0; i -= 2 {
				keys = append(keys, rs.Data[i].Bytes())
				// 分页游标
				ki := sdb.B2i(rs.Data[i].Bytes())
				si := sdb.B2i(rs.Data[i+1].Bytes())
				topicEditTimeMap[ki] = si
				if i == len(rs.Data)-2 {
					firstKey = ki
					firstScore = si
				} else {
					lastKey = ki
					lastScore = si
				}
			}
		}
	}

	//log.Println("keys len",len(keys))

	if len(keys) > 0 {
		// 评论数
		commentsMap := CommentGetNumByKeys(db, keys)

		var aItems []Topic
		userNameMap := map[uint64]string{}
		nodeNameMap := map[uint64]string{}

		rs := db.Hmget(TopicTbName, keys)
		if rs.KvLen() > 0 {
			for i := 0; i < (len(rs.Data) - 1); i += 2 {
				item := Topic{}
				_ = json.Unmarshal(rs.Data[i+1].Bytes(), &item)
				item.Comments, _ = commentsMap[item.ID]
				aItems = append(aItems, item)
				userNameMap[item.UserId] = ""
				nodeNameMap[item.NodeId] = ""
			}
		}

		// 获取用户 id:名字
		userIds := make([]uint64, 0, len(userNameMap))
		for k := range userNameMap {
			userIds = append(userIds, k)
		}
		userNameMap = UserGetNamesByIds(db, userIds)

		// 获取分类 id:名字
		nodeIds := make([]uint64, 0, len(nodeNameMap))
		for k := range nodeNameMap {
			nodeIds = append(nodeIds, k)
		}
		nodeNameMap = NodeGetNamesByIds(db, nodeIds)

		for _, article := range aItems {
			item := TopicLstLi{
				Topic: article,
				//FirstCon:    util.GetDesc(article.Content),
				AuthorName:  userNameMap[article.UserId],
				AddTimeFmt:  util.TimeFmt(article.AddTime, time.RFC3339),
				EditTimeFmt: util.TimeHuman(topicEditTimeMap[article.ID], TimeOffSet),
				NodeName:    nodeNameMap[article.NodeId],
			}

			items = append(items, item)
		}

		if db.Zscan(tb, sdb.I2b(firstKey), sdb.I2b(firstScore), 1).KvLen() > 0 {
			hasPrev = true
		}
		if db.Zrscan(tb, sdb.I2b(lastKey), sdb.I2b(lastScore), 1).KvLen() > 0 {
			hasNext = true
		}
	}

	return TopicPageInfo{
		Items:      items,
		HasPrev:    hasPrev,
		HasNext:    hasNext,
		FirstKey:   firstKey,
		FirstScore: firstScore,
		LastKey:    lastKey,
		LastScore:  lastScore,
	}
}

// GetTopicListSortById 首页帖子列表，按ID降序排列
func GetTopicListSortById(db *sdb.DB, cmd, tb, key, score string, limit int) TopicPageInfo {
	var items []TopicLstLi
	var keys [][]byte
	var hasPrev, hasNext bool
	var firstKey, firstScore, lastKey, lastScore uint64

	keyStart := sdb.DS2b(key)

	var topicLst []Topic
	userNameMap := map[uint64]string{}
	nodeNameMap := map[uint64]string{}

	if cmd == "zrscan" {
		rs := db.Hrscan(TopicTbName, keyStart, limit)
		if rs.State == "ok" {
			for i := 0; i < (len(rs.Data) - 1); i += 2 {
				keys = append(keys, rs.Data[i].Bytes())
				obj := Topic{}
				err := json.Unmarshal(rs.Data[i+1].Bytes(), &obj)
				if err == nil {
					topicLst = append(topicLst, obj)
					userNameMap[obj.UserId] = ""
					nodeNameMap[obj.NodeId] = ""
				}
			}
		}
	} else if cmd == "zscan" {
		rs := db.Hscan(TopicTbName, keyStart, limit)
		if rs.State == "ok" {
			for i := len(rs.Data) - 2; i >= 0; i -= 2 {
				keys = append(keys, rs.Data[i].Bytes())
				obj := Topic{}
				err := json.Unmarshal(rs.Data[i+1].Bytes(), &obj)
				if err == nil {
					topicLst = append(topicLst, obj)
					userNameMap[obj.UserId] = ""
					nodeNameMap[obj.NodeId] = ""
				}
			}
		}
	}

	if len(topicLst) > 0 {
		// 评论数
		commentsMap := CommentGetNumByKeys(db, keys)

		// 获取用户信息
		userIds := make([]uint64, 0, len(userNameMap))
		for k := range userNameMap {
			userIds = append(userIds, k)
		}
		userNameMap = UserGetNamesByIds(db, userIds)

		// 获取分类信息
		nodeIds := make([]uint64, 0, len(nodeNameMap))
		for k := range nodeNameMap {
			nodeIds = append(nodeIds, k)
		}
		nodeNameMap = NodeGetNamesByIds(db, nodeIds)

		for _, article := range topicLst {
			article.Comments, _ = commentsMap[article.ID]
			item := TopicLstLi{
				Topic:      article,
				FirstCon:   util.GetDesc(article.Content),
				AuthorName: userNameMap[article.UserId],
				// JAN 15TH, 2015
				AddTimeFmt:  util.TimeFmt(article.AddTime, time.RFC3339), // 2006-01-02T15:04:05Z07:00
				EditTimeFmt: util.TimeHuman(article.EditTime, TimeOffSet),
				NodeName:    nodeNameMap[article.NodeId],
			}

			items = append(items, item)
			if firstKey == 0 {
				firstKey = item.ID
				firstScore = uint64(item.AddTime)
			}
			lastKey = item.ID
			lastScore = uint64(item.AddTime)
		}

		if db.Hscan(TopicTbName, sdb.I2b(firstKey), 1).KvLen() > 0 {
			hasPrev = true
		}
		if db.Hrscan(TopicTbName, sdb.I2b(lastKey), 1).KvLen() > 0 {
			hasNext = true
		}
	}

	return TopicPageInfo{
		Items:      items,
		HasPrev:    hasPrev,
		HasNext:    hasNext,
		FirstKey:   firstKey,
		FirstScore: firstScore,
		LastKey:    lastKey,
		LastScore:  lastScore,
	}
}

// GetTopicListArchives 分页获取归档页：分类页、tag 的帖子
// 兼容接口，score 忽略
func GetTopicListArchives(db *sdb.DB, cmd, tb, key, score string, limit int) TopicPageInfo {
	var items []TopicLstLi
	var keys [][]byte
	var hasPrev, hasNext bool
	var firstKey, firstScore, lastKey, lastScore uint64

	keyStart := sdb.DS2b(key)
	if cmd == "zrscan" {
		rs := db.Hrscan(tb, keyStart, limit)
		if rs.KvLen() > 0 {
			// i := 0; i < (len(rs.Data) - 1); i += 2
			for i := 0; i < (len(rs.Data) - 1); i += 2 {
				keys = append(keys, rs.Data[i].Bytes())
			}
		}
	} else if cmd == "zscan" {
		rs := db.Hscan(tb, keyStart, limit)
		if rs.KvLen() > 0 {
			// i := len(rs.Data) - 2; i >= 0; i -= 2
			for i := len(rs.Data) - 2; i >= 0; i -= 2 {
				keys = append(keys, rs.Data[i].Bytes())
			}
		}
	}

	if len(keys) > 0 {
		// 评论数
		commentsMap := CommentGetNumByKeys(db, keys)

		var aitems []Topic
		userNameMap := map[uint64]string{}
		nodeNameMap := map[uint64]string{}

		rs := db.Hmget(TopicTbName, keys)
		if rs.KvLen() > 0 {
			for i := 0; i < (len(rs.Data) - 1); i += 2 {
				item := Topic{}
				_ = json.Unmarshal(rs.Data[i+1].Bytes(), &item)
				item.Comments, _ = commentsMap[item.ID]
				aitems = append(aitems, item)
				userNameMap[item.UserId] = ""
				nodeNameMap[item.NodeId] = ""
			}
		}

		// 获取用户信息
		userIds := make([]uint64, 0, len(userNameMap))
		for k := range userNameMap {
			userIds = append(userIds, k)
		}
		userNameMap = UserGetNamesByIds(db, userIds)

		// 获取分类信息
		nodeIds := make([]uint64, 0, len(nodeNameMap))
		for k := range nodeNameMap {
			nodeIds = append(nodeIds, k)
		}
		nodeNameMap = NodeGetNamesByIds(db, nodeIds)

		addYearMap := map[string]struct{}{}
		for _, article := range aitems {
			addTimeLst := strings.Split(util.TimeFmt(article.AddTime, "2006 Jan 02"), " ")
			item := TopicLstLi{
				Topic:       article,
				FirstCon:    util.GetDesc(article.Content),
				AuthorName:  userNameMap[article.UserId],
				EditTimeFmt: util.TimeFmt(article.EditTime, time.Stamp),
				NodeName:    nodeNameMap[article.NodeId],
				AddYear:     addTimeLst[0],
				AddMonth:    addTimeLst[1],
				AddDate:     addTimeLst[2],
			}
			addYear := addTimeLst[0]
			if _, ok := addYearMap[addYear]; !ok {
				// 重复的年不取
				addYearMap[addYear] = struct{}{}
				item.AddYearShow = addYear
			}

			items = append(items, item)
			if firstKey == 0 {
				firstKey = item.ID
			}
			lastKey = item.ID
		}

		if db.Hscan(tb, sdb.I2b(firstKey), 1).KvLen() > 0 {
			hasPrev = true
		}
		if db.Hrscan(tb, sdb.I2b(lastKey), 1).KvLen() > 0 {
			hasNext = true
		}
	}

	return TopicPageInfo{
		Items:      items,
		HasPrev:    hasPrev,
		HasNext:    hasNext,
		FirstKey:   firstKey,
		FirstScore: firstScore,
		LastKey:    lastKey,
		LastScore:  lastScore,
	}
}

// SearchTopicList 搜索
func SearchTopicList(mc *fastcache.Cache, db *sdb.DB, q string, limit int) (tInfo TopicPageInfo) {
	qInContent := strings.HasPrefix(q, "c:")
	if qInContent {
		q = strings.TrimSpace(q[2:])
	}
	if len(q) == 0 {
		return
	}
	qLow := strings.ToLower(q)

	var mcKey []byte
	if qInContent {
		mcKey = []byte("SearchTopicList:c:" + qLow)
	} else {
		mcKey = []byte("SearchTopicList:" + qLow)
	}
	if _, exist := util.ObjCachedGet(mc, mcKey, &tInfo, false); exist {
		return
	}

	var items []TopicLstLi
	var keys [][]byte
	var hasPrev, hasNext bool
	var firstKey, firstScore, lastKey, lastScore uint64

	var aitems []Topic
	userNameMap := map[uint64]string{}
	nodeNameMap := map[uint64]string{}

	var keyStart []byte
	for {
		if rs := db.Hrscan(TopicTbName, keyStart, 20); rs.OK() {
			rs.KvEach(func(key, value sdb.BS) {
				keyStart = key.Bytes()
				obj := Topic{}
				err := json.Unmarshal(value.Bytes(), &obj)
				if err != nil {
					return
				}
				var getIt bool
				if qInContent {
					if strings.Contains(strings.ToLower(obj.Content), qLow) {
						getIt = true
					}
				} else {
					if strings.Contains(strings.ToLower(obj.Title), qLow) {
						getIt = true
					}
				}
				if getIt && len(aitems) < limit {
					keys = append(keys, key.Bytes())
					aitems = append(aitems, obj)
					userNameMap[obj.UserId] = ""
					nodeNameMap[obj.NodeId] = ""
				}
			})
			if len(aitems) >= limit {
				break
			}
		} else {
			break
		}
	}

	// 评论数
	commentsMap := CommentGetNumByKeys(db, keys)
	// 获取用户信息
	userIds := make([]uint64, 0, len(userNameMap))
	for k := range userNameMap {
		userIds = append(userIds, k)
	}
	userNameMap = UserGetNamesByIds(db, userIds)

	// 获取分类信息
	nodeIds := make([]uint64, 0, len(nodeNameMap))
	for k := range nodeNameMap {
		nodeIds = append(nodeIds, k)
	}
	nodeNameMap = NodeGetNamesByIds(db, nodeIds)

	addYearMap := map[string]struct{}{}
	for _, article := range aitems {
		addTimeLst := strings.Split(util.TimeFmt(article.AddTime, "2006 Jan 02"), " ")
		article.Comments, _ = commentsMap[article.ID]
		item := TopicLstLi{
			Topic:       article,
			FirstCon:    util.GetDesc(article.Content),
			AuthorName:  userNameMap[article.UserId],
			EditTimeFmt: util.TimeFmt(article.EditTime, time.Stamp),
			NodeName:    nodeNameMap[article.NodeId],
			AddYear:     addTimeLst[0],
			AddMonth:    addTimeLst[1],
			AddDate:     addTimeLst[2],
		}
		addYear := addTimeLst[0]
		if _, ok := addYearMap[addYear]; !ok {
			// 重复的年不取
			addYearMap[addYear] = struct{}{}
			item.AddYearShow = addYear
		}

		items = append(items, item)
		if firstKey == 0 {
			firstKey = item.ID
			firstScore = uint64(item.EditTime)
		}
		lastKey = item.ID
		lastScore = uint64(item.EditTime)
	}

	tInfo = TopicPageInfo{
		Items:      items,
		HasPrev:    hasPrev,
		HasNext:    hasNext,
		FirstKey:   firstKey,
		FirstScore: firstScore,
		LastKey:    lastKey,
		LastScore:  lastScore,
	}

	// set to mc
	if len(tInfo.Items) > 0 {
		util.ObjCachedSet(mc, mcKey, tInfo)
	}

	return
}

// SearchTopicListFenCi 搜索 with fenCi
// db, where, qLow, words, h.App.Cf.Site.PageShowNum
func SearchTopicListFenCi(db *sdb.DB, where, kw string, words []string, limit int) (tInfo TopicPageInfo) {
	var items []TopicLstLi
	var keys [][]byte
	var hasPrev, hasNext bool
	var firstKey, firstScore, lastKey, lastScore uint64

	var aitems []Topic
	userNameMap := map[uint64]string{}
	nodeNameMap := map[uint64]string{}

	gotAid := map[uint64]struct{}{}
	// words
	if where == "title" && len(words) > 0 {
		// 根据 words 标签取
		scanMax := 200
		aidCount := map[uint64]int{}
		for _, tag := range words {
			tag = strings.ToLower(tag)
			db.Hrscan("tag:"+tag, nil, scanMax).KvEach(func(key, _ sdb.BS) {
				aid := sdb.B2i(key.Bytes())
				if _, ok := aidCount[aid]; ok {
					aidCount[aid] += 1
				} else {
					aidCount[aid] = 1
				}
			})
		}

		if len(aidCount) > 0 {
			type kv struct {
				Key   uint64
				Value int
			}
			var ss []kv
			for k, v := range aidCount {
				ss = append(ss, kv{k, v})
			}

			sort.Slice(ss, func(i, j int) bool {
				return ss[i].Value > ss[j].Value
			})

			var keys2 [][]byte // topic keys byte
			for _, kv := range ss {
				keys2 = append(keys2, sdb.I2b(kv.Key))
				if len(keys2) >= limit*2 {
					break
				}
			}

			db.Hmget(TopicTbName, keys2).KvEach(func(key, value sdb.BS) {
				if len(aitems) >= limit {
					return
				}
				obj := Topic{}
				err := json.Unmarshal(value.Bytes(), &obj)
				if err != nil {
					return
				}
				if _, ok := gotAid[obj.ID]; ok {
					return
				}

				keys = append(keys, key.Bytes())
				aitems = append(aitems, obj)
				userNameMap[obj.UserId] = ""
				nodeNameMap[obj.NodeId] = ""

				gotAid[obj.ID] = struct{}{}
			})
		}
	}

	if len(aitems) < limit {

		var keyStart []byte
		for {
			if rs := db.Hrscan(TopicTbName, keyStart, 20); rs.OK() {
				rs.KvEach(func(key, value sdb.BS) {
					keyStart = key.Bytes()
					obj := Topic{}
					err := json.Unmarshal(value.Bytes(), &obj)
					if err != nil {
						return
					}

					if _, ok := gotAid[obj.ID]; ok {
						return
					}

					var getIt bool
					if where == "content" {
						conLow := strings.ToLower(obj.Content)
						for _, wd := range words {
							if strings.Contains(conLow, wd) {
								getIt = true
								break
							}
						}
						if !getIt {
							if strings.Contains(conLow, kw) {
								getIt = true
							}
						}
					} else {
						titleLow := strings.ToLower(obj.Title)
						for _, wd := range words {
							if strings.Contains(titleLow, wd) {
								getIt = true
								break
							}
						}
						if !getIt {
							if strings.Contains(titleLow, kw) {
								getIt = true
							}
						}
					}
					if getIt && len(aitems) < limit {
						keys = append(keys, key.Bytes())
						aitems = append(aitems, obj)
						userNameMap[obj.UserId] = ""
						nodeNameMap[obj.NodeId] = ""

						gotAid[obj.ID] = struct{}{}
					}
				})
				if len(aitems) >= limit {
					break
				}
			} else {
				break
			}
		}
	}

	// 评论数
	commentsMap := CommentGetNumByKeys(db, keys)
	// 获取用户信息
	userIds := make([]uint64, 0, len(userNameMap))
	for k := range userNameMap {
		userIds = append(userIds, k)
	}
	userNameMap = UserGetNamesByIds(db, userIds)

	// 获取分类信息
	nodeIds := make([]uint64, 0, len(nodeNameMap))
	for k := range nodeNameMap {
		nodeIds = append(nodeIds, k)
	}
	nodeNameMap = NodeGetNamesByIds(db, nodeIds)

	addYearMap := map[string]struct{}{}
	for _, article := range aitems {
		addTimeLst := strings.Split(util.TimeFmt(article.AddTime, "2006 Jan 02"), " ")
		article.Comments, _ = commentsMap[article.ID]
		item := TopicLstLi{
			Topic:       article,
			FirstCon:    util.GetDesc(article.Content),
			AuthorName:  userNameMap[article.UserId],
			EditTimeFmt: util.TimeFmt(article.EditTime, time.Stamp),
			NodeName:    nodeNameMap[article.NodeId],
			AddYear:     addTimeLst[0],
			AddMonth:    addTimeLst[1],
			AddDate:     addTimeLst[2],
		}
		addYear := addTimeLst[0]
		if _, ok := addYearMap[addYear]; !ok {
			// 重复的年不取
			addYearMap[addYear] = struct{}{}
			item.AddYearShow = addYear
		}

		items = append(items, item)
		if firstKey == 0 {
			firstKey = item.ID
			firstScore = uint64(item.EditTime)
		}
		lastKey = item.ID
		lastScore = uint64(item.EditTime)
	}

	tInfo = TopicPageInfo{
		Items:      items,
		HasPrev:    hasPrev,
		HasNext:    hasNext,
		FirstKey:   firstKey,
		FirstScore: firstScore,
		LastKey:    lastKey,
		LastScore:  lastScore,
	}

	return
}

// GetMsgTopicList 站内信息帖子列表
func GetMsgTopicList(db *sdb.DB, uid uint64) (tpi TopicPageInfoMsg) {
	limit := 10 // 只取最早10条

	var items []TopicLstLiMsg
	var keys [][]byte // topic id

	tb := "user_msg:" + strconv.FormatUint(uid, 10)
	tidMsgMap := map[uint64]Msg{}
	db.Hscan(tb, nil, limit).KvEach(func(key, value sdb.BS) {
		obj := Msg{}
		err := json.Unmarshal(value.Bytes(), &obj)
		if err != nil {
			return
		}
		tidMsgMap[obj.TopicId] = obj
		keys = append(keys, key)
	})

	if len(keys) == 0 {
		return
	}

	topicMap := map[uint64]Topic{}
	userNameMap := map[uint64]string{}
	nodeNameMap := map[uint64]string{}

	rs := db.Hmget(TopicTbName, keys)
	if rs.KvLen() > 0 {
		for i := 0; i < (len(rs.Data) - 1); i += 2 {
			topic := Topic{}
			_ = json.Unmarshal(rs.Data[i+1].Bytes(), &topic)
			topicMap[topic.ID] = topic
			userNameMap[topic.UserId] = ""
			nodeNameMap[topic.NodeId] = ""
		}
	}

	// 评论数
	commentsMap := CommentGetNumByKeys(db, keys)

	// 获取用户信息
	userIds := make([]uint64, 0, len(userNameMap))
	for k := range userNameMap {
		userIds = append(userIds, k)
	}
	userNameMap = UserGetNamesByIds(db, userIds)

	// 获取分类信息
	nodeIds := make([]uint64, 0, len(nodeNameMap))
	for k := range nodeNameMap {
		nodeIds = append(nodeIds, k)
	}
	nodeNameMap = NodeGetNamesByIds(db, nodeIds)

	// 对 topicId 排序，按 Msg.AddTime 排序
	type kv struct {
		Key   uint64
		Value int64
	}

	var ss []kv
	for k, v := range tidMsgMap {
		ss = append(ss, kv{k, v.AddTime})
	}
	sort.Slice(ss, func(i, j int) bool {
		return ss[i].Value > ss[j].Value // 降序
		// return ss[i].Value > ss[j].Value  // 升序
	})

	for _, kv := range ss {
		topic := topicMap[kv.Key]
		msg := tidMsgMap[topic.ID]
		href := "/t/" + strconv.FormatUint(msg.TopicId, 10)
		if msg.CommentId > 0 {
			href += "#comment-" + strconv.FormatUint(msg.CommentId, 10)
		}
		topic.Comments, _ = commentsMap[topic.ID]
		item := TopicLstLiMsg{
			TopicLstLi: TopicLstLi{
				Topic:       topic,
				AuthorName:  userNameMap[topic.UserId],
				EditTimeFmt: util.TimeHuman(topic.EditTime, TimeOffSet),
				NodeName:    nodeNameMap[topic.NodeId],
			},
			Href: href,
		}

		items = append(items, item)
	}

	tpi.Items = items

	return
}

// TopicGetV2ReviewNum 获取用户待审核帖子列表条数
func TopicGetV2ReviewNum(db *sdb.DB, uid uint64) int {
	// 限制 10 条，若有10条未审核的则不允许再发
	return db.Hscan("review_topic:"+strconv.FormatUint(uid, 10), nil, 10).KvLen()
}

// TopicGetV2Review 获取用户待审核帖子列表
func TopicGetV2Review(db *sdb.DB, uid uint64) (objLst []TopicRecForm) {
	var ids [][]byte
	var keyStart []byte
	for {
		if rs := db.Hrscan("review_topic:"+strconv.FormatUint(uid, 10), keyStart, 10); rs.OK() {
			rs.KvEach(func(key, _ sdb.BS) {
				keyStart = key
				ids = append(ids, key)
			})
		} else {
			break
		}
	}

	if len(ids) == 0 {
		return
	}

	db.Hmget(TopicReviewTbName, ids).KvEach(func(_, value sdb.BS) {
		obj := TopicRecForm{}
		err := json.Unmarshal(value, &obj)
		if err != nil {
			return
		}
		obj.AddTimeFmt = util.TimeFmt(obj.AddTime, "2006-01-02 15:04")
		objLst = append(objLst, obj)
	})

	return
}

// CheckHasTopic2Review 检查有没有待审核帖子
func CheckHasTopic2Review(db *sdb.DB) bool {
	if db.Hscan(TopicReviewTbName, nil, 1).OK() {
		return true
	}
	return false
}

// TopicGetForFeed 为 feed 取帖子
func TopicGetForFeed(db *sdb.DB, limit int) (objLst []TopicFeed) {
	db.Hrscan(TopicTbName, nil, limit).KvEach(func(_, value sdb.BS) {
		obj := TopicFeed{}
		err := json.Unmarshal(value, &obj)
		if err != nil {
			return
		}
		obj.Title = html.EscapeString(obj.Title)
		obj.AddTimeFmt = util.TimeFmt(obj.AddTime, time.RFC3339)
		obj.EditTimeFmt = util.TimeFmt(obj.EditTime, time.RFC3339)
		obj.Des = html.EscapeString(util.GetDesc(obj.Content))
		objLst = append(objLst, obj)
	})
	return
}

// ArticleGetNearby 获取相邻文章
func ArticleGetNearby(db *sdb.DB, tid uint64) (oldObj, newObj TopicLi) {
	key := sdb.I2b(tid)
	if rs := db.Hrscan(TopicTbName, key, 1); rs.OK() {
		_ = json.Unmarshal(rs.Data[1].Bytes(), &oldObj)
	}
	if rs := db.Hscan(TopicTbName, key, 1); rs.OK() {
		_ = json.Unmarshal(rs.Data[1].Bytes(), &newObj)
	}
	return
}
