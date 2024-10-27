package model

import (
	"github.com/VictoriaMetrics/fastcache"
	"github.com/ego008/goutils/json"
	"github.com/ego008/sdb"
	"goyoubbs/util"
	"strconv"
	"time"
)

const (
	CommentTbName       = "comment:"       // + tid // 存放内容
	CommentNumTbName    = "comment_count"  // 帖子评论数
	CommentReviewTbName = "review_comment" // 待审核列表
)

type Comment struct {
	ID       uint64
	ReplyId  uint64 // 回复几楼
	TopicId  uint64
	UserId   uint64
	AddTime  int64
	Content  string
	ClientIp string
}

type CommentFmt struct {
	Comment
	Name       string
	AddTimeFmt string
	Link       string // 该评论锚点 /t/[tid]#r[cid]
	ContentFmt string
}

// CommentReview 待审核评论
type CommentReview struct {
	Comment
	TopicTitle string
	AddTimeFmt string
	ContentFmt string
}

func CommentGetById(db *sdb.DB, tid, cid uint64) (obj Comment) {
	if rs := db.Hget(CommentTbName+strconv.FormatUint(tid, 10), sdb.I2b(cid)); rs.OK() {
		err := json.Unmarshal(rs.Bytes(), &obj)
		if err != nil {
			return
		}
	}
	return
}

// CommentSet 编辑评论，只修改Content
func CommentSet(db *sdb.DB, obj Comment) Comment {
	jb, _ := json.Marshal(obj)
	_ = db.Hset(CommentTbName+strconv.FormatUint(obj.TopicId, 10), sdb.I2b(obj.ID), jb)
	return obj
}

// CommentAdd 添加评论
func CommentAdd(mc *fastcache.Cache, db *sdb.DB, obj Comment) Comment {
	_, _ = db.Hincr(CountTb, []byte("comment"), 1)                  // 总评论数
	newId, _ := db.Hincr(CommentNumTbName, sdb.I2b(obj.TopicId), 1) // 帖子评论数
	obj.ID = newId
	jb, _ := json.Marshal(obj)
	_ = db.Hset(CommentTbName+strconv.FormatUint(obj.TopicId, 10), sdb.I2b(newId), jb)
	// 更新 topic 相关时间线
	topic := TopicGetById(db, obj.TopicId)
	// 首页
	_ = db.Zset(TbnPostUpdate, sdb.I2b(obj.TopicId), uint64(obj.AddTime))
	// 分类页
	_ = db.Zset("topic_update:"+strconv.FormatUint(topic.NodeId, 10), sdb.I2b(obj.TopicId), uint64(obj.AddTime))
	// 个人回复的帖子
	_ = db.Zset("user_comment:"+strconv.FormatUint(obj.UserId, 10), sdb.I2b(obj.TopicId), uint64(obj.AddTime))
	// 最近回复内容
	k := []byte(strconv.FormatInt(obj.AddTime, 10) + "_" + strconv.FormatUint(obj.TopicId, 10))
	_ = db.Hset("recent_comment", k, sdb.I2b(obj.ID))
	// topic 回复者，浏览权限
	_ = db.Hset(TbnPostReply+strconv.FormatUint(obj.TopicId, 10), sdb.I2b(obj.UserId), nil)
	// 删缓存
	mc.Del([]byte("CommentGetRecent"))
	mc.Del([]byte("TopicGetRelative:" + strconv.FormatUint(obj.TopicId, 10)))
	mc.Del([]byte(CommentTbName + strconv.FormatUint(topic.ID, 10)))

	// @ 某人 及 主帖作者
	var toNoteUserIdLst []uint64 // 待提醒的用户Id
	userNameLst := util.GetMention(obj.Content, []string{})
	if topic.UserId != obj.UserId { // 排除自己回复
		// 提醒主贴作者
		toNoteUserIdLst = append(toNoteUserIdLst, topic.UserId)
	}
	if len(userNameLst) > 0 {
		db.Hmget("user_name2uid", userNameLst).KvEach(func(_, value sdb.BS) {
			uid := sdb.B2i(value)
			if uid != obj.UserId && uid != topic.UserId {
				// 排除 @ 自己 与 主贴作者
				toNoteUserIdLst = append(toNoteUserIdLst, uid)
			}
		})
	}

	if len(toNoteUserIdLst) > 0 {
		msg := Msg{
			TopicId:   topic.ID,
			CommentId: obj.ID,
			AddTime:   util.GetCNTM(TimeOffSet),
		}
		jb, _ = json.Marshal(msg)
		for _, uid := range toNoteUserIdLst {
			_ = db.Hset("user_msg:"+strconv.FormatUint(uid, 10), sdb.I2b(topic.ID), jb)
			time.Sleep(2 * time.Microsecond)
		}
	}

	// @ end

	return obj
}

func GetAllTopicComment(mc *fastcache.Cache, db *sdb.DB, topic Topic, isPublic, canRead bool) (objLst []CommentFmt) {
	tbName := CommentTbName + strconv.FormatUint(topic.ID, 10)
	mcKey := []byte(tbName)
	if isPublic {
		if _, exist := util.ObjCachedGetBig(mc, mcKey, &objLst, false); exist {
			return
		}
	}

	userMap := map[uint64]User{}

	db.Hscan(tbName, nil, int(topic.Comments)).KvEach(func(_, value sdb.BS) {
		obj := CommentFmt{}
		err := json.Unmarshal(value.Bytes(), &obj)
		if err != nil {
			return
		}
		obj.AddTimeFmt = util.TimeFmt(obj.AddTime, "2006-01-02 15:04")
		if isPublic {
			obj.ContentFmt = util.ContentFmt(obj.Content)
		} else {
			if canRead {
				obj.ContentFmt = util.ContentFmt(obj.Content)
			} else {
				publicCon, privateCon := util.GetPublicCon(topic.Content)
				obj.ContentFmt = util.ContentFmt(publicCon)
				if len(privateCon) > 0 {
					obj.ContentFmt += `<p>为了防止爬虫，本主题 "` + topic.Title + `" 的发布者已设置浏览权限，需要“登录”或“回复”才能完整浏览，剩余的内容包含个` + strconv.Itoa(len(privateCon)) + `字</p>`
				}
			}
		}
		obj.Link = "/t/" + strconv.FormatUint(obj.TopicId, 10) + "#r" + strconv.FormatUint(obj.ID, 10)
		objLst = append(objLst, obj)
		userMap[obj.UserId] = User{}
	})

	// 获取用户信息
	userIds := make([]uint64, 0, len(userMap))
	for k := range userMap {
		userIds = append(userIds, k)
	}
	for _, obj := range UserGetByIds(db, userIds) {
		userMap[obj.ID] = obj
	}
	// 设置comment name
	for i, v := range objLst {
		if user, found := userMap[v.UserId]; found {
			v.Name = user.Name
			objLst[i] = v
		}
	}

	if isPublic && len(objLst) > 0 {
		util.ObjCachedSetBig(mc, mcKey, objLst)
	}

	return
}

// CheckHasComment2Review 检查有没有待审核评论
func CheckHasComment2Review(db *sdb.DB) bool {
	if db.Hscan(CommentReviewTbName, nil, 1).OK() {
		return true
	}
	return false
}

// CommentGetNumByKeys 通过 []idByte 取评论数
func CommentGetNumByKeys(db *sdb.DB, keys [][]byte) map[uint64]uint64 {
	commentsMap := map[uint64]uint64{}
	db.Hmget(CommentNumTbName, keys).KvEach(func(key, value sdb.BS) {
		commentsMap[sdb.B2i(key)] = sdb.B2i(value)
	})
	return commentsMap
}

// CommentGetRecent 取最近评论
func CommentGetRecent(mc *fastcache.Cache, db *sdb.DB, limit int) (objLst []CommentFmt) {
	// get from mc
	mcKey := []byte("CommentGetRecent")
	if _, exist := util.ObjCachedGet(mc, mcKey, &objLst, false); exist {
		return
	}

	var sortKeyLst []string                  // 记录顺序 []tidCid
	commentMap := map[string]Comment{}       // tidCid:Comment
	topicCommentMap := map[string][][]byte{} // topicIdStr: [][]byte(commentId) // 无序
	db.Hrscan("recent_comment", nil, limit).KvEach(func(key, value sdb.BS) {
		tidStr := util.StringSplit(key.String(), "_")[1]
		k := CommentTbName + tidStr
		tidCid := tidStr + "_" + strconv.FormatUint(sdb.B2i(value.Bytes()), 10)
		sortKeyLst = append(sortKeyLst, tidCid)
		topicCommentMap[k] = append(topicCommentMap[k], value.Bytes()) // 一个帖子多条回复
	})

	if len(sortKeyLst) == 0 {
		return
	}

	uidMap := map[uint64]struct{}{}
	var userIds []uint64
	for k := range topicCommentMap {
		db.Hmget(k, topicCommentMap[k]).KvEach(func(key, value sdb.BS) {
			obj := Comment{}
			err := json.Unmarshal(value.Bytes(), &obj)
			if err != nil {
				return
			}
			tidCid := strconv.FormatUint(obj.TopicId, 10) + "_" + strconv.FormatUint(obj.ID, 10)
			commentMap[tidCid] = obj
			if _, ok := uidMap[obj.UserId]; !ok {
				uidMap[obj.UserId] = struct{}{}
				userIds = append(userIds, obj.UserId)
			}
		})
	}

	// get user name
	userNameMap := UserGetNamesByIds(db, userIds)
	// out put
	for _, k := range sortKeyLst {
		obj, _ := commentMap[k]
		// 截取内容
		obj.Content = util.GetShortCon(obj.Content)
		objLst = append(objLst, CommentFmt{
			Comment:    obj,
			Name:       userNameMap[obj.UserId],
			AddTimeFmt: util.TimeFmt(obj.AddTime, "2006-01-02 15:04"),
			ContentFmt: obj.Content,
			Link:       "/t/" + strconv.FormatUint(obj.TopicId, 10) + "#r" + strconv.FormatUint(obj.ID, 10),
		})
	}

	// set to mc
	util.ObjCachedSet(mc, mcKey, objLst)

	return
}

// CommentGetReviewNum 取个人待审核评论条数
func CommentGetReviewNum(db *sdb.DB, uid uint64) int {
	// 限制 10 条，若有10条未审核的则不允许再发
	return db.Hscan(CommentReviewTbName+":"+strconv.FormatUint(uid, 10), nil, 10).KvLen()
}

// CommentGetReview 取个人待审核评论
func CommentGetReview(db *sdb.DB, uid uint64) (objLst []CommentReview) {
	var topicIds []uint64
	var ids [][]byte
	var keyStart []byte

	for {
		if rs := db.Hrscan(CommentReviewTbName+":"+strconv.FormatUint(uid, 10), keyStart, 10); rs.OK() {
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

	commentMap := map[string]Comment{}
	db.Hmget(CommentReviewTbName, ids).KvEach(func(key, value sdb.BS) {
		obj := Comment{}
		err := json.Unmarshal(value, &obj)
		if err != nil {
			return
		}
		commentMap[key.String()] = obj
		topicIds = append(topicIds, obj.TopicId)
	})

	// get topic title
	topicTitleMap := TopicGetTitlesByIds(db, topicIds)

	for _, v := range ids {
		vStr := string(v)
		comment := commentMap[vStr]
		objLst = append(objLst, CommentReview{
			Comment:    comment,
			TopicTitle: topicTitleMap[comment.TopicId],
			AddTimeFmt: util.TimeFmt(comment.AddTime, ""),
			ContentFmt: util.ContentFmt(comment.Content),
		})
	}
	return
}
