package model

import (
	"slices"

	"github.com/VictoriaMetrics/fastcache"
	"github.com/ego008/goutils/json"
	"github.com/ego008/mdb"
	"github.com/tidwall/gjson"
	"go.etcd.io/bbolt"

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
	ReadMoreBreak     = "<!-- read more -->"
)

type Topic struct {
	ID         uint64
	NodeId     uint64
	UserId     uint64
	Title      string
	Content    string
	ClientIp   string
	Tags       string
	ReadAuthed bool
	ReadReply  bool
	AddTime    int64
	EditTime   int64
	Comments   uint64
}

type TopicRecForm struct {
	ID         uint64
	Act        string
	NodeId     uint64
	UserId     uint64
	Title      string
	Content    string
	Tags       string
	ReadAuthed bool
	ReadReply  bool
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
	TotalNum   uint64
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
func TopicSet(db *mdb.DB, tx *bbolt.Tx, obj Topic) Topic {
	jb, err := json.Marshal(obj)
	if err != nil {
		return obj
	}
	_ = db.HSet(tx, TopicTbName, mdb.I2b(obj.ID), jb)
	return obj
}

func TopicAdd(mc *fastcache.Cache, db *mdb.DB, tx *bbolt.Tx, obj Topic) Topic {
	newId, _ := db.HIncr(tx, CountTb, mdb.S2b(TopicTbName), 1) // id 自增
	obj.ID = newId
	jb, _ := json.Marshal(obj)
	_ = db.HSet(tx, TopicTbName, mdb.I2b(obj.ID), jb)
	// 添加时间轴
	// 首页
	_ = db.ZSet(tx, TbnPostUpdate, mdb.I2b(obj.ID), uint64(obj.AddTime))
	// 分类页
	_ = db.ZSet(tx, "topic_update:"+strconv.FormatUint(obj.NodeId, 10), mdb.I2b(obj.ID), uint64(obj.AddTime))
	// 个人主贴
	_ = db.ZSet(tx, "user_topic:"+strconv.FormatUint(obj.UserId, 10), mdb.I2b(obj.ID), uint64(obj.AddTime))
	// 分类归档，按id、添加时间排序
	_ = db.HSet(tx, "topic_node:"+strconv.FormatUint(obj.NodeId, 10), mdb.I2b(obj.ID), mdb.I2b(uint64(obj.AddTime)))
	// 该分类的文章数
	_, _ = db.HIncr(tx, NodeTopicNumTbName, mdb.I2b(obj.NodeId), 1)
	// 文章总数
	_, _ = db.HIncr(tx, CountTb, mdb.S2b(TopicTbName+":all_number"), 1)
	// 用户帖子数+1
	_, _ = db.HIncr(tx, TbnUserTopicNum, mdb.I2b(obj.UserId), 1)
	// 删除分类缓存
	mc.Del([]byte("NodeGetAll"))
	return obj
}

// TopicDel 删除文章
func TopicDel(mc *fastcache.Cache, db *mdb.DB, tx *bbolt.Tx, obj Topic) {
	// 首页
	_ = db.ZDel(tx, TbnPostUpdate, mdb.I2b(obj.ID))
	// 分类页
	_ = db.ZDel(tx, "topic_update:"+strconv.FormatUint(obj.NodeId, 10), mdb.I2b(obj.ID))
	// 个人主贴
	_ = db.ZDel(tx, "user_topic:"+strconv.FormatUint(obj.UserId, 10), mdb.I2b(obj.ID))
	// 分类归档，按id、添加时间排序
	_ = db.HDel(tx, "topic_node:"+strconv.FormatUint(obj.NodeId, 10), mdb.I2b(obj.ID))
	// 该分类的文章数 -1
	_, _ = db.HIncr(tx, NodeTopicNumTbName, mdb.I2b(obj.NodeId), -1)
	// 文章总数-1
	_, _ = db.HIncr(tx, CountTb, mdb.S2b(TopicTbName+":all_number"), -1)
	// 标签
	if len(obj.Tags) > 0 {
		// 删除标签，参见 cronjob/topic_tag
		for _, tag := range util.StringSplit(obj.Tags, ",") {
			tagLower := strings.ToLower(tag)
			tagLowerB := mdb.S2b(tagLower)
			_ = db.HDel(tx, "tag:"+tagLower, mdb.I2b(obj.ID))

			var ok bool
			_ = db.HScanFunc(tx, "tag:"+tagLower, nil, 1, func(key, val []byte) bool {
				ok = true
				_, _ = db.ZIncr(tx, "tag_article_num", tagLowerB, -1) // 热门标签排序
				return true
			})

			if !ok {
				// 删除
				_ = db.ZDel(tx, "tag_article_num", tagLowerB)
				_ = db.HDel(tx, "tag", tagLowerB)
				_, _ = db.HIncr(tx, "tag_count", mdb.S2b("tag"), -1)
			}
		}
	}
	// 文章实体
	_ = db.HDel(tx, TopicTbName, mdb.I2b(obj.ID))
	// 删除分类缓存
	mc.Del([]byte("NodeGetAll"))
	mc.Del([]byte("GetTagsForSide"))
}

func TopicGetById(db *mdb.DB, tx *bbolt.Tx, tid uint64) (obj Topic) {
	_ = db.HGetFunc(tx, TopicTbName, mdb.I2b(tid), func(val []byte) error {
		_ = json.Unmarshal(val, &obj)
		return nil
	})
	return
}

// TopicGetTitlesByIds 根据 ids 取 title ，返回id:title 的map
func TopicGetTitlesByIds(db *mdb.DB, tx *bbolt.Tx, ids []uint64) map[uint64]string {
	id2name := map[uint64]string{}
	if len(ids) == 0 {
		return id2name
	}
	var idsb [][]byte
	for _, k := range ids {
		idsb = append(idsb, mdb.I2b(k))
	}
	_ = db.HMGetFunc(tx, TopicTbName, idsb, func(key, val []byte) error {
		if len(val) == 0 {
			return nil
		}
		title := gjson.Get(string(val), "Title").String()
		id2name[mdb.B2i(key)] = title
		return nil
	})
	return id2name
}

func TopicGetRelative(mc *fastcache.Cache, db *mdb.DB, tx *bbolt.Tx, aid uint64, tags string) (topLst []TopicLi) {
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

	for _, tag := range util.StringSplit(tagsLow, ",") {
		_ = db.HRScanFunc(tx, "tag:"+tag, nil, scanMax, func(key, val []byte) bool {
			aid2 := mdb.B2i(key)
			if aid2 != aid {
				if _, ok := aidCount[aid2]; ok {
					aidCount[aid2] += 1
				} else {
					aidCount[aid2] = 1
				}
			}
			return true
		})
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
			akeys = append(akeys, mdb.I2b(kv.Key))
			j++
			if j == getMax {
				break
			}
		}

		_ = db.HMGetFunc(tx, TopicTbName, akeys, func(key, val []byte) error {
			if len(val) == 0 {
				return nil
			}
			item := TopicLi{}
			_ = json.Unmarshal(val, &item)
			topLst = append(topLst, item)
			return nil
		})
	}

	// set to mc
	if len(topLst) > 0 {
		util.ObjCachedSet(mc, mcKey, topLst)
	}

	return
}

// GetTopicList 分页获取首页、分类页的帖子
func GetTopicList(db *mdb.DB, tx *bbolt.Tx, cmd, tb, key, score string, limit int) TopicPageInfo {
	var items []TopicLstLi
	var keys [][]byte // key  按 score 从大到小排列
	var hasPrev, hasNext bool
	var firstKey, firstScore, lastKey, lastScore uint64 // firstScore 最大， lastScore 最小

	topicEditTimeMap := map[uint64]uint64{} // 文章编辑时间 / 最后评论时间
	keyStart := mdb.DS2b(key)
	scoreStart := mdb.DS2i(score)
	if cmd == "zrscan" {
		var i int
		// ZRScanFunc 降序扫描
		_ = db.ZRScanFunc(tx, tb, keyStart, scoreStart, 0, limit, func(key []byte, score uint64) bool {
			keys = append(keys, key) // score 大的在前
			// 分页游标
			ki := mdb.B2i(key)
			si := score
			topicEditTimeMap[ki] = si
			if i == 0 {
				firstKey = ki
				firstScore = si // 最大
			} else {
				lastKey = ki
				lastScore = si // 最小
			}
			i++
			return true
		})
	} else if cmd == "zscan" {
		var i int
		// ZScanFunc 升序扫描
		_ = db.ZScanFunc(tx, tb, keyStart, scoreStart, 0, limit, func(key []byte, score uint64) bool {
			keys = append(keys, key) // 目前 score 小的在前，要先大后小，等会倒转
			// 分页游标
			ki := mdb.B2i(key)
			si := score
			topicEditTimeMap[ki] = si
			if i == 0 {
				lastKey = ki
				lastScore = si // 最小
			} else {
				firstKey = ki
				firstScore = si // 最大
			}
			i++
			return true
		})

		// 倒转 keys
		slices.Reverse(keys)
	}

	//log.Println("keys len",len(keys))

	if len(keys) > 0 {
		// 评论数
		commentsMap := CommentGetNumByKeys(db, tx, keys)

		var aItems []Topic
		userNameMap := map[uint64]string{}
		nodeNameMap := map[uint64]string{}

		_ = db.HMGetFunc(tx, TopicTbName, keys, func(key, val []byte) error {
			if len(val) == 0 {
				return nil
			}
			item := Topic{}
			_ = json.Unmarshal(val, &item)
			item.Comments, _ = commentsMap[item.ID]
			aItems = append(aItems, item)
			userNameMap[item.UserId] = ""
			nodeNameMap[item.NodeId] = ""
			return nil
		})

		// 获取用户 id:名字
		userIds := make([]uint64, 0, len(userNameMap))
		for k := range userNameMap {
			userIds = append(userIds, k)
		}
		userNameMap = UserGetNamesByIds(db, tx, userIds)

		// 获取分类 id:名字
		nodeIds := make([]uint64, 0, len(nodeNameMap))
		for k := range nodeNameMap {
			nodeIds = append(nodeIds, k)
		}
		nodeNameMap = NodeGetNamesByIds(db, tx, nodeIds)

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

		_ = db.ZScanFunc(tx, tb, mdb.I2b(firstKey), firstScore, 0, 1, func(_ []byte, _ uint64) bool {
			hasPrev = true
			return true
		})
		_ = db.ZRScanFunc(tx, tb, mdb.I2b(lastKey), lastScore, 0, 1, func(_ []byte, _ uint64) bool {
			hasNext = true
			return true
		})
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
func GetTopicListArchives(db *mdb.DB, tx *bbolt.Tx, cmd, tb, key string, limit int) TopicPageInfo {
	var items []TopicLstLi
	var keys [][]byte
	var hasPrev, hasNext bool
	var firstKey, firstScore, lastKey, lastScore uint64

	keyStart := mdb.DS2b(key)
	if cmd == "zrscan" {
		_ = db.HRScanFunc(tx, tb, keyStart, limit, func(key, val []byte) bool {
			keys = append(keys, key)
			return true
		})
	} else if cmd == "zscan" {
		_ = db.HScanFunc(tx, tb, keyStart, limit, func(key, val []byte) bool {
			keys = append(keys, key)
			return true
		})
		// 倒转 keys
		slices.Reverse(keys)
	}

	if len(keys) > 0 {
		// 评论数
		commentsMap := CommentGetNumByKeys(db, tx, keys)

		var aitems []Topic
		userNameMap := map[uint64]string{}
		nodeNameMap := map[uint64]string{}

		_ = db.HMGetFunc(tx, TopicTbName, keys, func(key, val []byte) error {
			if len(val) == 0 {
				return nil
			}
			item := Topic{}
			_ = json.Unmarshal(val, &item)
			item.Comments, _ = commentsMap[item.ID]
			aitems = append(aitems, item)
			userNameMap[item.UserId] = ""
			nodeNameMap[item.NodeId] = ""
			return nil
		})

		// 获取用户信息
		userIds := make([]uint64, 0, len(userNameMap))
		for k := range userNameMap {
			userIds = append(userIds, k)
		}
		userNameMap = UserGetNamesByIds(db, tx, userIds)

		// 获取分类信息
		nodeIds := make([]uint64, 0, len(nodeNameMap))
		for k := range nodeNameMap {
			nodeIds = append(nodeIds, k)
		}
		nodeNameMap = NodeGetNamesByIds(db, tx, nodeIds)

		addYearMap := map[string]struct{}{}
		for _, article := range aitems {
			addTimeLst := util.StringSplit(util.TimeFmt(article.AddTime, "2006 Jan 02"), " ")
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

		_ = db.HScanFunc(tx, tb, mdb.I2b(firstKey), 1, func(_, _ []byte) bool {
			hasPrev = true
			return true
		})
		_ = db.HRScanFunc(tx, tb, mdb.I2b(lastKey), 1, func(_, _ []byte) bool {
			hasNext = true
			return true
		})
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
func SearchTopicList(mc *fastcache.Cache, db *mdb.DB, tx *bbolt.Tx, q string, limit int) (tInfo TopicPageInfo) {
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
		var ok bool
		_ = db.HRScanFunc(tx, TopicTbName, keyStart, 20, func(key, val []byte) bool {
			keyStart = key
			obj := Topic{}
			err := json.Unmarshal(val, &obj)
			if err != nil {
				return true
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
				keys = append(keys, key)
				aitems = append(aitems, obj)
				userNameMap[obj.UserId] = ""
				nodeNameMap[obj.NodeId] = ""
			}
			ok = true
			return true
		})
		if !ok || len(aitems) >= limit {
			break
		}
	}

	// 评论数
	commentsMap := CommentGetNumByKeys(db, tx, keys)
	// 获取用户信息
	userIds := make([]uint64, 0, len(userNameMap))
	for k := range userNameMap {
		userIds = append(userIds, k)
	}
	userNameMap = UserGetNamesByIds(db, tx, userIds)

	// 获取分类信息
	nodeIds := make([]uint64, 0, len(nodeNameMap))
	for k := range nodeNameMap {
		nodeIds = append(nodeIds, k)
	}
	nodeNameMap = NodeGetNamesByIds(db, tx, nodeIds)

	addYearMap := map[string]struct{}{}
	for _, article := range aitems {
		addTimeLst := util.StringSplit(util.TimeFmt(article.AddTime, "2006 Jan 02"), " ")
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

// GetMsgTopicList 站内信息帖子列表
func GetMsgTopicList(db *mdb.DB, tx *bbolt.Tx, uid uint64) (tpi TopicPageInfoMsg) {
	limit := 10 // 只取最早10条

	var items []TopicLstLiMsg
	var keys [][]byte // topic id

	tb := "user_msg:" + strconv.FormatUint(uid, 10)
	tidMsgMap := map[uint64]Msg{}

	_ = db.HScanFunc(tx, tb, nil, limit, func(key, val []byte) bool {
		obj := Msg{}
		err := json.Unmarshal(val, &obj)
		if err != nil {
			return true
		}
		tidMsgMap[obj.TopicId] = obj
		keys = append(keys, key)
		return true
	})

	if len(keys) == 0 {
		return
	}

	topicMap := map[uint64]Topic{}
	userNameMap := map[uint64]string{}
	nodeNameMap := map[uint64]string{}

	_ = db.HMGetFunc(tx, TopicTbName, keys, func(_, val []byte) error {
		if len(val) == 0 {
			return nil
		}
		topic := Topic{}
		_ = json.Unmarshal(val, &topic)
		topicMap[topic.ID] = topic
		userNameMap[topic.UserId] = ""
		nodeNameMap[topic.NodeId] = ""
		return nil
	})

	// 评论数
	commentsMap := CommentGetNumByKeys(db, tx, keys)

	// 获取用户信息
	userIds := make([]uint64, 0, len(userNameMap))
	for k := range userNameMap {
		userIds = append(userIds, k)
	}
	userNameMap = UserGetNamesByIds(db, tx, userIds)

	// 获取分类信息
	nodeIds := make([]uint64, 0, len(nodeNameMap))
	for k := range nodeNameMap {
		nodeIds = append(nodeIds, k)
	}
	nodeNameMap = NodeGetNamesByIds(db, tx, nodeIds)

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
func TopicGetV2ReviewNum(db *mdb.DB, tx *bbolt.Tx, uid uint64) int {
	// 限制 10 条，若有10条未审核的则不允许再发
	var n int
	_ = db.HScanFunc(tx, "review_topic:"+strconv.FormatUint(uid, 10), nil, 10, func(_, _ []byte) bool {
		n++
		return true
	})
	return n
}

// TopicGetV2Review 获取用户待审核帖子列表
func TopicGetV2Review(db *mdb.DB, tx *bbolt.Tx, uid uint64) (objLst []TopicRecForm) {
	var ids [][]byte
	var keyStart []byte
	for {
		var ok bool
		_ = db.HRScanFunc(tx, "review_topic:"+strconv.FormatUint(uid, 10), keyStart, 10, func(key, _ []byte) bool {
			ok = true
			keyStart = key
			ids = append(ids, key)
			return true
		})
		if !ok {
			break
		}
	}

	if len(ids) == 0 {
		return
	}

	_ = db.HMGetFunc(tx, TopicReviewTbName, ids, func(_, val []byte) error {
		obj := TopicRecForm{}
		err := json.Unmarshal(val, &obj)
		if err != nil {
			return nil
		}
		obj.AddTimeFmt = util.TimeFmt(obj.AddTime, "2006-01-02 15:04")
		objLst = append(objLst, obj)
		return nil
	})

	return
}

// CheckHasTopic2Review 检查有没有待审核帖子
func CheckHasTopic2Review(db *mdb.DB, tx *bbolt.Tx) bool {
	var ok bool
	_ = db.HScanFunc(tx, TopicReviewTbName, nil, 1, func(key, val []byte) bool {
		ok = true
		return true
	})
	return ok
}

// TopicGetForFeed 为 feed 取帖子
func TopicGetForFeed(db *mdb.DB, tx *bbolt.Tx, limit int) (objLst []TopicFeed) {
	_ = db.HRScanFunc(tx, TopicTbName, nil, limit, func(_, val []byte) bool {
		obj := TopicFeed{}
		err := json.Unmarshal(val, &obj)
		if err != nil {
			return true
		}
		obj.Title = html.EscapeString(obj.Title)
		obj.AddTimeFmt = util.TimeFmt(obj.AddTime, time.RFC3339)
		obj.EditTimeFmt = util.TimeFmt(obj.EditTime, time.RFC3339)
		obj.Des = html.EscapeString(util.GetDesc(obj.Content))
		objLst = append(objLst, obj)
		return true
	})
	return
}

// ArticleGetNearby 获取相邻文章
func ArticleGetNearby(db *mdb.DB, tx *bbolt.Tx, tid uint64) (oldObj, newObj TopicLi) {
	key := mdb.I2b(tid)
	_ = db.HRScanFunc(tx, TopicTbName, key, 1, func(_, val []byte) bool {
		_ = json.Unmarshal(val, &oldObj)
		return true
	})
	_ = db.HScanFunc(tx, TopicTbName, key, 1, func(_, val []byte) bool {
		_ = json.Unmarshal(val, &newObj)
		return true
	})
	return
}
