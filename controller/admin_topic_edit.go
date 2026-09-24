package controller

import (
	"goyoubbs/model"
	"goyoubbs/views/admin"
	"html"
	"net/http"
	"strconv"

	"github.com/ego008/mdb"
	"github.com/gin-gonic/gin"
	"go.etcd.io/bbolt"
)

func (h *BaseHandler) AdminTopicEditPage(c *gin.Context) {
	curUser, _ := h.CurrentUser(c)
	if curUser.Flag < model.FlagAdmin {
		c.Redirect(302, "/admin")
		return
	}

	scf := h.App.Cf.Site

	evn := &admin.TopicAdd{}
	evn.CurrentUser = *curUser
	evn.SiteCf = scf
	evn.Title = "编辑帖子"
	evn.PageName = "admin_topic_edit"

	tid := c.Query("id") // 编辑帖子id
	tidI, err := strconv.ParseUint(tid, 10, 64)
	if err != nil {
		c.String(200, `{"Code":400,"Msg":"id 不是数字"}`)
		return
	}

	var showStr string
	var seeOther bool

	_ = h.App.Db.Update(func(tx *bbolt.Tx) error {

		//var rec model.Topic
		rec := model.TopicGetById(h.App.Db, tx, tidI)
		if rec.ID == 0 {
			showStr = `{"Code":400,"Msg":"该 id 帖子不存在"}`
			return nil
		}

		// 是不是删除
		isDel := c.Query("del") // 删除帖子
		if isDel == "1" {
			model.TopicDel(h.App.Mc, h.App.Db, tx, rec)
			if h.App.Cf.Site.SaveTopicIcon {
				// 删九宫格图片
				_ = h.App.Db.HDel(tx, "topic_icon", mdb.I2b(tidI))
			}
			seeOther = true
			return nil
		}

		var author model.User
		if rec.UserId > 0 {
			author, _ = model.UserGetById(h.App.Db, tx, rec.UserId)
		}
		if author.ID == 0 {
			author = evn.CurrentUser
		}

		evn.ReadMoreBreak = model.ReadMoreBreak
		// 编辑时内容转义
		rec.Content = html.EscapeString(rec.Content)
		evn.DefaultTopic = rec
		if evn.DefaultTopic.NodeId == 0 {
			evn.DefaultTopic.NodeId = 1
		}

		evn.DefaultUser = author
		// evn.DefaultNode, _ = model.NodeGetById(h.App.Db, evn.DefaultTopic.NodeId)
		// evn.DefaultNode = model.Node{}
		evn.NodeLst = model.NodeGetAll(h.App.Mc, h.App.Db, tx)

		// evn.UserLst = model.UserGetAllAdmin(h.App.Db)
		evn.UserLst = []model.User{author}

		if len(c.Query("back")) > 0 {
			evn.GoBack = true
		}

		evn.HasMsg = model.MsgCheckHasOne(h.App.Db, tx, curUser.ID)
		evn.HasTopicReview = model.CheckHasTopic2Review(h.App.Db, tx)
		evn.HasReplyReview = model.CheckHasComment2Review(h.App.Db, tx)

		return nil
	})

	if len(showStr) > 0 {
		c.String(200, showStr)
		return
	}
	if seeOther {
		c.Redirect(302, "/admin/topic/add")
		return
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusOK)
	admin.WritePageTemplate(c.Writer, evn)
}
