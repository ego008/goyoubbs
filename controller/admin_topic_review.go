package controller

import (
	"bytes"
	"goyoubbs/model"
	"goyoubbs/views/admin"
	"net/http"
	"strconv"

	"github.com/ego008/goutils/json"
	"github.com/gin-gonic/gin"
	"go.etcd.io/bbolt"
)

func (h *BaseHandler) AdminTopicReviewPage(c *gin.Context) {
	curUser, _ := h.CurrentUser(c)
	if curUser.Flag < model.FlagAdmin {
		c.Redirect(302, "/admin")
		return
	}

	scf := h.App.Cf.Site

	evn := &admin.TopicAdd{}
	evn.CurrentUser = *curUser
	evn.SiteCf = scf
	evn.Title = "待审核帖子"
	evn.PageName = "admin_topic_review"

	act := c.Query("act")
	var delKey []byte // 待删除的key
	// 取待审核信息
	var rec model.Topic

	_ = h.App.Db.Update(func(tx *bbolt.Tx) error {

		_ = h.App.Db.HScanFunc(tx, model.TopicReviewTbName, nil, 1, func(key, val []byte) bool {
			delKey = bytes.Clone(key)
			err := json.Unmarshal(val, &rec)
			if err != nil {
				return true
			}
			return true
		})

		if act == "del" {
			// 删掉管理员列表
			_ = h.App.Db.HDel(tx, model.TopicReviewTbName, delKey)
			// 删掉个人待审核列表
			_ = h.App.Db.HDel(tx, "review_topic:"+strconv.FormatUint(rec.UserId, 10), delKey)
			c.Redirect(302, "/admin/topic/review")
			return nil
		}

		var author model.User
		if rec.UserId > 0 {
			author, _ = model.UserGetById(h.App.Db, tx, rec.UserId)
		}
		if author.ID == 0 {
			author = evn.CurrentUser
		}

		evn.DefaultTopic = model.Topic{
			NodeId:  rec.NodeId,
			UserId:  author.ID,
			Title:   rec.Title,
			Content: rec.Content,
			AddTime: rec.AddTime,
		}
		if evn.DefaultTopic.NodeId == 0 {
			evn.DefaultTopic.NodeId = 1
		}

		evn.DefaultUser = author
		// evn.DefaultNode, _ = model.NodeGetById(h.App.Db, evn.DefaultTopic.NodeId)
		// evn.DefaultNode = model.Node{}
		evn.NodeLst = model.NodeGetAll(h.App.Mc, h.App.Db, tx)

		// evn.UserLst = model.UserGetAllAdmin(h.App.Db)
		evn.UserLst = []model.User{author}

		evn.HasMsg = model.MsgCheckHasOne(h.App.Db, tx, curUser.ID)
		evn.HasTopicReview = model.CheckHasTopic2Review(h.App.Db, tx)
		evn.HasReplyReview = model.CheckHasComment2Review(h.App.Db, tx)

		return nil
	})

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusOK)
	admin.WritePageTemplate(c.Writer, evn)
}
