package controller

import (
	"goyoubbs/model"
	"goyoubbs/util"
	"goyoubbs/views/admin"
	"html"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.etcd.io/bbolt"
)

func (h *BaseHandler) AdminCommentEditPage(c *gin.Context) {
	curUser, _ := h.CurrentUser(c)
	if curUser.Flag < model.FlagAdmin {
		c.Redirect(302, "/admin")
		return
	}

	scf := h.App.Cf.Site
	db := h.App.Db

	tid, cid := c.Query("tid"), c.Query("cid")
	tidI, err := strconv.ParseUint(tid, 10, 64)
	if err != nil {
		c.Redirect(302, "/admin")
		return
	}
	cidI, err := strconv.ParseUint(cid, 10, 64)
	if err != nil {
		c.Redirect(302, "/admin")
		return
	}

	evn := &admin.CommentEdit{}
	evn.CurrentUser = *curUser
	evn.SiteCf = scf
	evn.Title = "评论修改"
	evn.PageName = "admin_comment_edit"

	_ = db.View(func(tx *bbolt.Tx) error {
		comment := model.CommentGetById(db, tx, tidI, cidI)
		author, _ := model.UserGetById(db, tx, comment.UserId)

		if author.ID == 0 {
			author = evn.CurrentUser
		}

		evn.ReadMoreBreak = model.ReadMoreBreak
		evn.NodeLst = model.NodeGetAll(h.App.Mc, h.App.Db, tx)
		evn.DefaultTopic = model.TopicGetById(db, tx, comment.TopicId)
		comment.Content = html.EscapeString(comment.Content) // 转义
		evn.DefaultComment = model.CommentFmt{
			Comment:    comment,
			Name:       author.Name,
			AddTimeFmt: util.TimeFmt(comment.AddTime, ""),
			ContentFmt: comment.Content,
		}
		evn.DefaultUser = author

		if len(c.Query("back")) > 0 {
			evn.GoBack = true
		}

		evn.HasMsg = model.MsgCheckHasOne(db, tx, curUser.ID)
		evn.HasTopicReview = model.CheckHasTopic2Review(h.App.Db, tx)
		evn.HasReplyReview = model.CheckHasComment2Review(h.App.Db, tx)

		return nil
	})

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusOK)
	admin.WritePageTemplate(c.Writer, evn)
}
