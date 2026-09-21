package model

import (
	"github.com/ego008/mdb"
	"go.etcd.io/bbolt"

	"strconv"
)

// Msg 站内 帖子被回复 或 被@ 信息提醒，当用户未读信息超过100 条时不再添加
// key 为 TopicId，记录最后一次提醒
type Msg struct {
	TopicId   uint64
	CommentId uint64
	AddTime   int64 // 被 @ 时间，仅做排序用
}

func MsgCheckHasOne(db *mdb.DB, tx *bbolt.Tx, uid uint64) bool {
	var ok bool
	_ = db.HScanFunc(tx, "user_msg:"+strconv.FormatUint(uid, 10), nil, 1, func(_, _ []byte) bool {
		ok = true
		return true
	})
	return ok
}
