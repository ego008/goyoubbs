package controller

import (
	"goyoubbs/model"
	"goyoubbs/views/admin"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.etcd.io/bbolt"
)

func (h *BaseHandler) AdminLinkPage(c *gin.Context) {
	curUser, _ := h.CurrentUser(c)
	if curUser.Flag < model.FlagAdmin {
		c.Redirect(302, "/admin")
		return
	}

	scf := h.App.Cf.Site

	evn := &admin.Link{}
	evn.CurrentUser = *curUser
	evn.SiteCf = scf
	evn.Title = "链接管理"
	evn.PageName = "admin_link"

	_ = h.App.Db.View(func(tx *bbolt.Tx) error {
		evn.NodeLst = model.NodeGetAll(h.App.Mc, h.App.Db, tx)
		evn.LinkLst = model.LinkList(h.App.Mc, h.App.Db, tx, true)

		//
		evn.Act = "添加"
		evn.Link = model.Link{}
		ids := c.Query("id")
		if len(ids) > 0 {
			evn.Link = model.Link{}
			link := model.LinkGetById(h.App.Db, tx, ids)
			if link.ID > 0 {
				evn.Link = link
				evn.Act = "编辑"
			}
		}

		evn.HasMsg = model.MsgCheckHasOne(h.App.Db, tx, curUser.ID)
		evn.HasTopicReview = model.CheckHasTopic2Review(h.App.Db, tx)
		evn.HasReplyReview = model.CheckHasComment2Review(h.App.Db, tx)

		return nil
	})

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusOK)
	admin.WritePageTemplate(c.Writer, evn)
}

func (h *BaseHandler) AdminLinkPost(c *gin.Context) {
	curUser, _ := h.CurrentUser(c)
	if curUser.Flag < model.FlagAdmin {
		c.Redirect(302, "/login")
		return
	}

	var id uint64
	ids := c.Query("id")
	if len(ids) > 0 {
		idI, err := strconv.ParseUint(ids, 10, 64)
		if err == nil {
			id = idI
		}
	}

	obj := model.Link{}
	obj.ID = id
	obj.Name = c.PostForm("Name")
	obj.Score, _ = strconv.Atoi(c.PostForm("Score"))
	obj.Url = c.PostForm("Url")

	_ = h.App.Db.Update(func(tx *bbolt.Tx) error {
		model.LinkSet(h.App.Db, tx, obj)
		return nil
	})

	// 删除缓存
	h.App.Mc.Del([]byte("LinkList"))

	c.Redirect(302, "/admin/link")
}
