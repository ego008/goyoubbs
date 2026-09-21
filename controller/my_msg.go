package controller

import (
	"goyoubbs/model"
	"goyoubbs/views/ybs"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.etcd.io/bbolt"
)

func (h *BaseHandler) MyMsgPage(c *gin.Context) {
	curUser, _ := h.CurrentUser(c)
	if curUser.ID == 0 {
		c.Redirect(302, "/login")
		return
	}

	db := h.App.Db
	scf := h.App.Cf.Site

	evn := &ybs.MyMsg{}
	evn.CurrentUser = *curUser
	evn.SiteCf = scf
	evn.Title = "未读信息"

	_ = db.View(func(tx *bbolt.Tx) error {
		evn.HasMsg = model.MsgCheckHasOne(db, tx, curUser.ID)
		evn.TopicPageInfo = model.GetMsgTopicList(db, tx, curUser.ID)

		evn.HasMsg = model.MsgCheckHasOne(db, tx, curUser.ID)
		if curUser.Flag >= model.FlagAdmin {
			evn.HasTopicReview = model.CheckHasTopic2Review(db, tx)
			evn.HasReplyReview = model.CheckHasComment2Review(db, tx)
		}
		return nil
	})

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusOK)
	ybs.WritePageTemplate(c.Writer, evn)
}
